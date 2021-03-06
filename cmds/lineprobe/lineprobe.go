/*
This command is the client portion of the pipeline plotter.

Lineprobe supports two modes. By default, it reads values from Stdin. As an
alternate to reading from Stdin, lineprobe can execute a given command and read
the command's Stdout. Each line is interprted as a single floating point value.

If executing a given command, the command may output a single value and exit, or
multiple. If the command exits with status 0, it will be executed again after a
configurable interval. If the command exists non-zero, then lineprobe exits
also.

Command Options

Each section below describes the lineprobe options.

Typically, the lineprobe and lineviewer will run on the same system. When this
is the case, default values work well. However, lineprobe can target a
collector running on any system. To specify an alternate collector use:

   -hostname=localhost:3131

The default is for lineprobe to read from Stdin. However, lineprobe can execute
a given command and read from the command's Stdout instead. After the command
exits with a zero status, lineprobe waits an interval before re-executing the
command. If the command exists non-zero, lineprobe also exits.

   -interval=1.0
   -command=

By default, lineprobe reports on the values it receives and sends. To silence
this, use:

   -quiet

For additional debug logging:

   -debug

Data Options

The default operation for lineprobe is to forward values directly to the
lineviewer collector. Lineprobe also has the capability to perform simple
operations on the raw data. Operations have this form:

   -noop
   -percentile=<samples>,<percentile>
   -avg=<samples>[,<showStdev>]
   -scale=1.0
   -shift=0.0

Operation -percentile requires two arguments separated by a comma. The first is
the number of samples to operate over, and the second is the percentile to take
from those samples.

Operation -avg requires one argument and an optional second separated by a
comma. The first is the number of samples to average. If the second option is
provided, a value of 'true' indicates that +/- standard deviation of the
samples should be displayed along with the average.

The -scale is applied to a calculated value. The -shift is applied after the
-scale.

Plot Options

Lineprobe can send line and plot style hints to the lineviewer.

If not given, a line color is chosen automatically by the lineviewer. To
specify the line color use a format like:

   -color="#ffffff"

Lines are given a default name based on the command executed or current PID and
operation. To specify a custom name use:

   -label=<name>

To name the X and Y axis use:

   -xlabel=<label>
   -ylabel=<label>

By default, all lines are added to a single axis. To create multiple axes
specify a distinct axis name. Other lineprobes can use the same name to add
lines to the same axis. The axis name is not user visible.

   -axis=default

For a given axis name, you can specify the the minimum and maximum values for
the Y axis.

   -ylimit=<ymin>:<ymax>

Or, plot the Y-axis on a log scale.

   -ylog

Server Options

Normally the lineviewer runs indefinitely. When profiling the server, it is
necessary to cause the server to exit normally. This option tells the
lineviewer to exit.

   -exit

Examples

Reading data from stdin:
 while /usr/bin/true; do ps ax | wc -l ; sleep 1 ; done | lineprobe

Reading data from command output:
    lineprobe --command "ps ax | wc -l" --interval 2.0

*/
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	// third-party
	"github.com/stephen-soltesz/go/collection/token"
	"github.com/stephen-soltesz/go/lineserver"
)

var (
	hostname    = flag.String("hostname", "localhost:3131", "Host and port of view server.")
	command     = flag.String("command", "", "Command to run every second.")
	interval    = flag.Float64("interval", 1.0, "How often to run command.")
	figTitle    = flag.String("title", "", "Title for complete figure.")
	axisName    = flag.String("axis", "default", "Name of axis to associate this line.")
	xlabelName  = flag.String("xlabel", "", "X-Label for axis.")
	ylabelName  = flag.String("ylabel", "", "Y-Label for axis.")
	ylimitValue = flag.String("ylimit", "", "Y-limits for axis 'ymin:ymax'.")
	yaxisScale  = flag.Bool("ylog", false, "Y-axis should be log scale.")
	lineName    = flag.String("label", "", "Line label name on axis.")
	lineColor   = flag.String("color", "", "Color of line. Chosen automatically by default.")
	quietOutput = flag.Bool("quiet", false, "Whether to echo values sent to server.")
	exitServer  = flag.Bool("exit", false, "Tell the lineviewer to exit.")
	debug       = flag.Bool("debug", false, "Enable debug messages on stderr.")
	debugLogger *log.Logger

	calcScale = flag.Float64("scale", 1.0, "Multiplies the given scale value with each sample.")
	calcShift = flag.Float64("shift", 0.0, "Adds the given shift value to each sample.")

	// Initialized in initFlags().
	calcNone = &NoOperation{}
	calcAvg  = &AvgOperation{}
	calcPerc = &PercOperation{}
	calcNoop = flag.Bool("noop", false, "Enable raw data plot when other operations are specified.")
)

func initFlags() {
	flag.Var(calcPerc, "percentile",
		"Return the percentile over the given sample size: <samples>,<percentile>.")
	flag.Var(calcAvg, "avg",
		"Return the avg over the given sample size: <samples>[,<showStdev>].")
}

type sampleSet struct {
	samples []float64
	pos     int
}

type ValueWriter struct {
	writer  *bufio.ReadWriter
	server  lineserver.Server
	op      Operation
	samples *sampleSet
}

type ValueReader struct {
	reader    *bufio.Reader
	cmd       *exec.Cmd
	fromStdin bool
}

func checkFlags() {
	if *debug {
		debugLogger = log.New(os.Stderr, "", 0)
	} else {
		debugLogger = log.New(ioutil.Discard, "", 0)
	}
	if len(calcOperations) == 0 || *calcNoop {
		// Create one default operation (if no others were specified, or user specified the -noop plot).
		calcNone.Set("")
	}
}

func printAllFlagsAndValues() {
	flag.VisitAll(func(f *flag.Flag) {
		debugLogger.Printf("%-30s %s\n", fmt.Sprintf("-%s=%s", f.Name, f.Value), f.Usage)
	})
}

func newSampleSet(sampleCount int64) *sampleSet {
	return &sampleSet{make([]float64, sampleCount), 0}
}

func (ss *sampleSet) Count() int {
	if ss.Full() {
		return len(ss.samples)
	} else {
		return ss.pos
	}
}

func (ss *sampleSet) Full() bool {
	return ss.pos >= len(ss.samples)-1
}

func (ss *sampleSet) Append(val float64) {
	ss.samples[ss.pos%len(ss.samples)] = val
	ss.pos += 1
}

func (ss *sampleSet) Mean() float64 {
	count := ss.Count()
	var s float64 = 0
	for i := 0; i < count; i++ {
		s += ss.samples[i]
	}
	avg := s / float64(count)
	debugLogger.Println("avg:", avg)
	return avg
}

func (ss *sampleSet) Current() float64 {
	return ss.samples[ss.pos%len(ss.samples)]
}

func (ss *sampleSet) Stdev() float64 {
	count := ss.Count()
	avg := ss.Mean()
	var variance float64 = 0

	for i := 0; i < count; i++ {
		variance += math.Pow((ss.samples[i] - avg), 2)
	}
	stdev := math.Sqrt(variance / float64(count))
	debugLogger.Println("stdev:", stdev)
	return stdev
}

func (ss *sampleSet) Percentile(pct int64) float64 {
	count := ss.Count()
	var calc = make([]float64, count)
	copy(calc, ss.samples)

	i := int(math.Ceil(float64(count-1) * float64(pct) / 101.0))
	sort.Float64s(calc)
	debugLogger.Println("size", count, "i", i, "pct:", calc[i])
	return calc[i]
}

func NewValueWriter(op Operation) *ValueWriter {
	var err error
	debugLogger.Printf("%#v\n", op)
	writer := ValueWriter{}
	writer.op = op
	writer.samples = newSampleSet(op.Samples())
	writer.server = lineserver.NewServer(*hostname)
	if writer.writer, err = writer.server.Connect(); err != nil {
		log.Fatal(err)
	}
	return &writer
}

func SetupValueWriter(op Operation) *ValueWriter {
	writer := NewValueWriter(op)
	if err := writer.sendClientSettings(); err != nil {
		log.Fatal(err)
	}
	return writer
}

func SetupValueWriters() []*ValueWriter {
	var writers []*ValueWriter
	for _, op := range calcOperations {
		writers = append(writers, SetupValueWriter(op))
	}
	return writers
}

func (w *ValueWriter) SendValue(val float64) error {
	var f float64
	w.samples.Append(val)
	switch w.op.(type) {
	case *NoOperation:
		f = w.samples.Current()
	case *AvgOperation:
		avgOp := w.op.(*AvgOperation)
		if avgOp.Extra() == 0 {
			f = w.samples.Mean()
		} else if avgOp.Extra() <= -1 {
			f = w.samples.Mean() - w.samples.Stdev()
		} else if avgOp.Extra() >= 1 {
			f = w.samples.Mean() + w.samples.Stdev()
		}
	case *PercOperation:
		f = w.samples.Percentile(w.op.Extra())
	}

	if *calcScale != 1.0 {
		debugLogger.Printf("Scale %f*%f=%f", f, *calcScale, *calcScale*f)
	}
	f *= *calcScale
	if *calcShift != 0.0 {
		debugLogger.Printf("Shift %f+%f=%f", f, *calcShift, *calcShift+f)
	}
	f += *calcShift

	out := formatFloat(f)
	fmt.Printf("Sending: %s %s", w.op, string(out))
	_, err := w.writer.Write(out)
	if err != nil {
		return err
	}
	w.writer.Flush()
	return nil
}

func (w *ValueWriter) sendClientSettings() error {
	var err error
	if *exitServer {
		debugLogger.Println("Sending exit")
		_, err := w.writer.WriteString(token.Exit + "\n")
		if err != nil {
			return err
		}
		w.writer.Flush()
		// create new connection to trigger exit.
		time.Sleep(time.Second)
		NewValueWriter(newNoOperation())
		os.Exit(0)
	}
	axisSetting := token.Axis + ":" + *axisName + ":" + *xlabelName + ":" + *ylabelName + "\n"
	if _, err = w.writer.WriteString(axisSetting); err != nil {
		return err
	}

	if *figTitle != "" {
		figTitleSetting := token.FigureTitle + ":" + *figTitle + "\n"
		if _, err := w.writer.WriteString(figTitleSetting); err != nil {
			return err
		}
	}

	if *ylimitValue != "" {
		limitSetting := token.AxisYLimit + ":" + *ylimitValue + "\n"
		if _, err := w.writer.WriteString(limitSetting); err != nil {
			return err
		}
	}

	if *yaxisScale {
		yaxisScale := token.AxisScale + ":" + token.ScaleLog + "\n"
		if _, err := w.writer.WriteString(yaxisScale); err != nil {
			return err
		}
	}

	name := ""
	if len(calcOperations) > 1 {
		name = fmt.Sprintf("%s", w.op)
	}

	if *lineName != "" {
		name = token.LineLabel + ":" + *lineName + name + "\n"
	} else if *command != "" {
		name = token.LineLabel + ":" + *command + name + "\n"
	} else {
		name = token.LineLabel + ":" + fmt.Sprintf("Thread-%d", os.Getpid()) + name + "\n"
	}

	if _, err = w.writer.WriteString(name); err != nil {
		return err
	}

	if *lineColor != "" {
		colorSetting := fmt.Sprintf("%s:%s\n", token.LineColor, *lineColor)
		if _, err := w.writer.WriteString(colorSetting); err != nil {
			return err
		}
	}
	return nil
}

func NewValueReader() *ValueReader {
	reader := ValueReader{}
	reader.Setup()
	return &reader
}

func (r *ValueReader) HandleEOF() error {
	if r.fromStdin {
		// just pass along EOF if reading from Stdin.
		return io.EOF
	}

	// reading from a command.
	r.cmd.Wait() // must run Wait() after EOF to avoid races with Read().
	if !r.cmd.ProcessState.Success() {
		fmt.Println("Non-zero exit value from: " + *command)
		// TODO: exit with child exit status.
		os.Exit(1)
	}

	// pause before re-executing command.
	time.Sleep(time.Duration(*interval) * time.Second)
	return r.Setup()
}

func (r *ValueReader) Setup() error {
	r.fromStdin = ("" == *command)

	if r.fromStdin {
		r.reader = bufio.NewReader(os.Stdin)
	} else {
		r.cmd = exec.Command("bash", "-c", *command)
		output, err := r.cmd.StdoutPipe()
		if err != nil {
			debugLogger.Println("failed StdoutPipe", err)
			return err
		}
		if err := r.cmd.Start(); err != nil {
			debugLogger.Println("failed to Start command", err)
			return err
		}
		r.reader = bufio.NewReader(output)
	}
	return nil
}

func (r *ValueReader) ReadValue() (float64, error) {
	var err error
	var lineTooLong bool
	var out []byte

	out, lineTooLong, err = r.reader.ReadLine()
	if lineTooLong {
		return 0.0, errors.New("Line too long.")
	} else if err == io.EOF {
		err := r.HandleEOF()
		if err != nil {
			return 0.0, err
		}
		return r.ReadValue()
	} else if err != nil {
		fmt.Println("Unknown failure:", err)
		return 0.0, err
	}

	f, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0.0, err
	}
	if !*quietOutput {
		fmt.Printf("Received: %f\n", f)
	}
	return f, nil
}

func formatFloat(f float64) []byte {
	return []byte(fmt.Sprintf("%f\n", f))
}

func main() {
	var err error
	var val float64

	initFlags()
	flag.Parse()
	checkFlags()

	if !*quietOutput {
		printAllFlagsAndValues()
	}

	// allocate value reader and writers.
	reader := NewValueReader()
	writers := SetupValueWriters()

	for {
		// read one value.
		if val, err = reader.ReadValue(); err != nil {
			if err != io.EOF {
				log.Println(err) // only report non-EOF.
			}
			break
		}
		// and, send value to all writers.
		for _, writer := range writers {
			if err = writer.SendValue(val); err != nil {
				log.Fatal(err)
			}
		}
	}
	return
}
