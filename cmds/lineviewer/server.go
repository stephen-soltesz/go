/*
This command is the server portion of the pipeline plotter.

Lineview listens on two ports. One is the collector port, to which lineprobe
clients connect and send data corresponding to data streams. The second port is
a simple http server that allows multiple viewers of the collected data in
real-time.

Command Options

Lineviewer runs two servers on different ports. The first is the collector.
Lineprobe clients connect directly to the connector port. The second port is
the display and shows the image via a simple http server.

By default, lineviewer binds to all interfaces. This permits lineprobe clients
from other systems, or for viewers on other systems.

  -hostname=0.0.0.0
  -viewer_port=8080         Port for web viewer.
  -collector_port=3131      Port for data collector.

The server also has an option for addition debug information. And, you can
enable collection of profiling information.

  -debug
  -profile                  Writes to lineviewer.prof.

Plot Options

By default, the lineviewer treats the X-axis as the sequence number of each
sample. However, the lineviewer can track the timestamp of when each value is
received.

  -timestamp                Collect and display samples with a timestamp.
  -plot_width=<int>         Width in pixels.
  -plot_height=<int>        Height in pixels.
  -samples=<int>            Initial number of samples (or, seconds for -timestamp)
                            to use in plots for new clients. Clients can adjust
                            this interactively.

Examples

An example of invoking lineviewer.

   lineviewer --timestamp --plot_width 800 --plot_height 600

*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime/pprof"

	"github.com/stephen-soltesz/chart"
	"github.com/stephen-soltesz/go/collection"
)

var (
	bindAddress   = flag.String("hostname", "0.0.0.0", "Interface to bind to.")
	viewerPort    = flag.Int("viewer_port", 8080, "Port for web viewer.")
	collectorPort = flag.Int("collector_port", 3131, "Port for data collector.")
	timestamp     = flag.Bool("timestamp", false, "Use timestamps as x-axis.")
	plotWidth     = flag.Int("plot_width", 600, "Plot width in pixels.")
	plotHeight    = flag.Int("plot_height", 400, "Plot height in pixels.")
	plotSamples   = flag.Int("samples", 240, "Number of samples wide to make plots.")
	debug         = flag.Bool("debug", false, "Enable debug messages on stderr.")
	debugLogger   *log.Logger
	profile       = flag.Bool("profile", false, "Enable profiling.")
)

func checkFlags() {
	if *viewerPort == *collectorPort {
		fmt.Println("Error: viewer and collector cannot use the same port.")
		os.Exit(1)
	}
	if *debug {
		debugLogger = log.New(os.Stderr, "", 0)
	} else {
		debugLogger = log.New(ioutil.Discard, "", 0)
	}
	chart.DebugLogger = debugLogger
}

func main() {
	flag.Parse()
	checkFlags()

	if *profile {
		f, err := os.Create("lineviewer.prof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	collector := collection.Default()
	collector.Usetime = *timestamp
	go startViewServer(*bindAddress, *viewerPort)
	startCollectorServer(*bindAddress, *collectorPort)
	//select {}
}
