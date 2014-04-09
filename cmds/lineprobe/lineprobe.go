/*
This command is the client portion of the pipeline plotter.

Lineprobe supports two modes. By default, it reads values from Stdin. Each line
is a single value. If the argument `-command` is used, rather than reading from
Stdin, lineprobe runs the given command and reads its output. The command
should return a single value and exit with status 0. The command will be
executed every 'interval' seconds.

Options

These are command-line options.

    -hostname=localhost:3131  Host and port of view server.
    -interval=1.0             How often to run command.
    -command=                 Command to run every second.
    -axis=default             Name of axis to associate this line.
    -xlabel=                  X-Label for axis.
    -ylabel=                  Y-Label for axis.
    -label=                   Line label name on axis.
    -color=                   Color of line as "#ffffff". Chosen automatically by default.
    -q                        Silence the echo of values sent to the server.
    -exit                     Signal the lineviewer to exit. Useful when profiling the server.
		-operation=<kind>,<samples>[,<percentile>]
                              Perform an optional operation on collected samples 
                              before sending to lineviewer.
                              <kind> is one of "avg", "stdev", "perc".
                              <samples> is an integer specifying the number of samples to operate over.
                              <percentile> if kind=perc, then this is the percentile to
                                 report from the last <samples> values.
    -debug                    Print extra debug information.

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
	"time"

	// third-party
	"github.com/stephen-soltesz/go/lineserver"
)

var (
	hostname    = flag.String("hostname", "localhost:3131", "Host and port of view server.")
	command     = flag.String("command", "", "Command to run every second.")
	interval    = flag.Float64("interval", 1.0, "How often to run command.")
	axisName    = flag.String("axis", "default", "Name of axis to associate this line.")
	xlabelName  = flag.String("xlabel", "", "X-Label for axis.")
	ylabelName  = flag.String("ylabel", "", "Y-Label for axis.")
	lineName    = flag.String("label", "", "Line label name on axis.")
	lineColor   = flag.String("color", "", "Color of line. Chosen automatically by default.")
	quietOutput = flag.Bool("q", false, "Whether to echo values sent to server.")
	exitServer  = flag.Bool("exit", false, "Tell the lineviewer to exit.")

	calcOperations operationSlice

	debug         = flag.Bool("debug", false, "Enable debug messages on stderr.")
	debugLogger   *log.Logger
)

type sampleSet struct {
	samples []float64
	pos int
}

type ValueWriter struct {
	writer *bufio.ReadWriter
	server *lineserver.Server
	op *Operation
	samples *sampleSet
}

type ValueReader struct {
	reader *bufio.Reader
	cmd *exec.Cmd
}

func initFlags() {
	flag.Var(&calcOperations, "operation", "An operation to run on samples.")
}

func checkFlags() {
	if *debug {
		debugLogger = log.New(os.Stderr, "", 0)
	} else {
		debugLogger = log.New(ioutil.Discard, "", 0)
	}
	if len(calcOperations) == 0 {
		calcOperations = append(calcOperations, &Operation{OpPercentile, 1, 50})
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
	avg := s/float64(count)
	debugLogger.Println("avg:", avg)
	return avg
}

func (ss *sampleSet) Stdev() float64 {
	count := ss.Count()
	avg := ss.Mean()
	var variance float64 = 0

	for i := 0; i < count; i++ {
		variance += math.Pow((ss.samples[i] - avg), 2)
	}
	stdev := math.Sqrt(variance/float64(count))
	debugLogger.Println("stdev:", stdev)
	return stdev
}

func (ss *sampleSet) Percentile(pct int64) float64 {
	count := ss.Count()
	var calc = make([]float64, count)
	copy(calc, ss.samples)

	i := int(math.Ceil(float64(count-1) * float64(pct)/101.0))
	sort.Float64s(calc)
	debugLogger.Println("size", count, "i", i)
	debugLogger.Println("pct:", calc[i])
	return calc[i]
}

func NewValueWriter(op *Operation) *ValueWriter {
	var err error
	writer := ValueWriter{}
	writer.op = op
	fmt.Printf("%#v\n", op)
	writer.samples = newSampleSet(op.samples)
	writer.server = lineserver.NewServer(*hostname)
	if writer.writer, err = writer.server.Connect(); err != nil {
		log.Fatal(err)
	}
	return &writer
}

func SetupValueWriters() []*ValueWriter {
	var writers []*ValueWriter
	for _, op := range calcOperations {
		writer := NewValueWriter(op)
		if err := writer.sendClientSettings(); err != nil {
			log.Fatal(err)
		}
		writers = append(writers, writer)
	}
	return writers
}

func (w *ValueWriter) SendValue(val float64) error {
	var f float64
	w.samples.Append(val)
	if w.op.operation == OpMean {
		f = w.samples.Mean()
	} else if w.op.operation == OpStdev {
		f = w.samples.Stdev()
	} else if w.op.operation == OpPercentile {
		f = w.samples.Percentile(w.op.percentile)
	}

	out := formatFloat(f)
	fmt.Printf("Sending: %s", string(out))
	_, err := w.writer.Write(out)
	if err != nil {
		return err
	}
	w.writer.Flush()
	return nil
}

func (w *ValueWriter) sendClientSettings() error {
	if *exitServer {
		fmt.Println("Sending exit")
		_, err := w.writer.WriteString("EXIT\n")
		if err != nil {
			return err
		}
		w.writer.Flush()
		// create new connection to trigger exit.
		time.Sleep(time.Second)
		NewValueWriter(&Operation{OpNone, 0, 0})
		os.Exit(0)
	}
	axisSetting := "axis:" + *axisName + ":" + *xlabelName + ":" + *ylabelName + "\n"
	_, err := w.writer.WriteString(axisSetting)
	if err != nil {
		return err
	}

	name := ""
	if w.op.operation > OpNone {
		name = fmt.Sprintf("%s-", w.op)
	}

	if *lineName != "" {
		name = "label:" + *lineName + "\n"
	} else if *command != "" {
		name = "label:" + name + *command + "\n"
	} else {
		name = "label:" + name + fmt.Sprintf("Thread-%d\n", os.Getpid())
	}

	_, err = w.writer.WriteString(name)
	if err != nil {
		return err
	}

	if *lineColor != "" {
		colorSetting := fmt.Sprintf("color:%s\n", *lineColor)
		_, err := w.writer.WriteString(colorSetting)
		if err != nil {
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

func (r *ValueReader) Setup() error {
	var fromStdin bool = ("" == *command)

	if fromStdin {
		r.reader = bufio.NewReader(os.Stdin)
	} else {
		r.cmd = exec.Command("bash", "-c", *command)
		output, err := r.cmd.StdoutPipe()
		if err != nil {
			return err
		}
		if err := r.cmd.Start(); err != nil {
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
		return 0.0, errors.New("line too long")
	} else if err == io.EOF {
		r.cmd.Wait()	// must run after EOF to avoid races with Read()
		time.Sleep(time.Duration(*interval) * time.Second)
		err := r.Setup()
		if err != nil {
			fmt.Println("reader setup failed:", err)
			return 0.0, err
		}
		return r.ReadValue()
	} else if err != nil {
		fmt.Println("unknown failure:", err)
		return 0.0, err
	}

	f, err := strconv.ParseFloat(string(out), 64)
	if err != nil {
		return 0.0, err
	}
	if !*quietOutput {
		fmt.Printf("Received: %f\n", f)
	}
	return f, nil
}

func parseFloat(val []byte) float64 {
	fmt.Println("%v", val)
	f, err := strconv.ParseFloat(string(val), 64)
	if err != nil {
		debugLogger.Println(err)
		return 0.0
	}
	return f
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
		if val, err = reader.ReadValue(); err != nil {
			log.Println(err)
			break
		}
		for _, writer := range writers {
			if err = writer.SendValue(val); err != nil {
				log.Fatal(err)
			}
		}
	}
	return
}
