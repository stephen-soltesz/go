/*
This command is the client portion of the pipeline plotter.

Lineprobe supports two modes. By default, it reads values from Stdin. Each line
is a single value. If the argument `-command` is used, rather than reading from
Stdin, lineprobe runs the given command and reads its output. The command
should return a single value and exit with status 0. The command will be
executed every 'interval' seconds.

Options

These are command-line options.

    -hostname=localhost:3131       Host and port of view server.
    -interval=1.0                  How often to run command.
    -command=                      Command to run every second.
    -axis=default                  Name of axis to associate this line.
    -xlabel=                       X-Label for axis.
    -ylabel=                       Y-Label for axis.
    -label=                        Line label name on axis.
    -color=                        Color of line. Chosen automatically by default.
    -q                             Silence the echo of values sent to the server.

Examples

Reading data from stdin:
    while /usr/bin/true; do ps ax | wc -l ; sleep 1 ; done | lineprobe 

Reading data from command output:
    lineprobe --command "ps ax | wc -l" --interval 2.0

*/
package main

import (
  "bufio"
  "flag"
  "fmt"
  "log"
	"io"
  "os"
  "os/exec"
  "time"

	// third-party
	"github.com/stephen-soltesz/go/lineserver"
)

var (
  hostname = flag.String("hostname", "localhost:3131", "Host and port of view server.")
  command = flag.String("command", "", "Command to run every second.")
  interval = flag.Float64("interval", 1.0, "How often to run command.")
  axisName = flag.String("axis", "default", "Name of axis to associate this line.")
  xlabelName = flag.String("xlabel", "", "X-Label for axis.")
  ylabelName = flag.String("ylabel", "", "Y-Label for axis.")
  lineName = flag.String("label", "", "Line label name on axis.")
  lineColor = flag.String("color", "", "Color of line. Chosen automatically by default.")
  quietOutput = flag.Bool("q", false, "Whether to echo values sent to server.")
)

type ValueReader struct {
	reader *bufio.Reader
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
    cmd := exec.Command("bash", "-c", *command)
    output, err := cmd.StdoutPipe();
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		go cmd.Wait()
		r.reader = bufio.NewReader(output)
	}
	return nil
}

func (r *ValueReader) GetValue() []byte {
  var err error
  var lineTooLong bool
  var out []byte

  out, lineTooLong, err = r.reader.ReadLine()
  out = append(out, '\n')
  if lineTooLong {
    return nil
  } else if err == io.EOF {
    time.Sleep(time.Duration(*interval) * time.Second)
		err := r.Setup()
		if err != nil {
			log.Fatal(err)
		}
		return r.GetValue()
	} else if err != nil {
    log.Fatal(err)
  }
  return out
}

func SendValue(writer io.Writer, out []byte) {
  _, err := writer.Write(out)
  if err != nil {
    log.Fatal(err)
  }
}

func sendClientSettings(writer *bufio.ReadWriter) {
	axisSetting := "axis:" + *axisName + ":" + *xlabelName + ":" + *ylabelName + "\n"
  SendValue(writer, []byte(axisSetting))
	if *lineName != "" {
		SendValue(writer, []byte(fmt.Sprintf("label:%s\n", *lineName)))
	}
	if *lineColor != "" {
		SendValue(writer, []byte(fmt.Sprintf("color:%s\n", *lineColor)))
	}
}

func main() {
	var writer *bufio.ReadWriter
	var out []byte
	var err error

  flag.Parse()
	if !*quietOutput {
		flag.VisitAll(func (f *flag.Flag) {
			fmt.Printf("%-30s %s\n", fmt.Sprintf("-%s=%s", f.Name, f.Value), f.Usage)
		})
	}

  server := lineserver.NewServer(*hostname)
	if writer, err = server.Connect(); err != nil {
		log.Fatal(err)
	}

	reader := NewValueReader()
	sendClientSettings(writer)

  for {
    if out = reader.GetValue(); out == nil {
      continue
    }
		if !*quietOutput {
			fmt.Print("Sending: ", string(out))
		}
    SendValue(writer, out)
	  writer.Flush()
  }

  return
}
