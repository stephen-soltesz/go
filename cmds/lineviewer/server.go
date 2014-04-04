/*
This command is the server portion of the pipeline plotter.

Lineview listens on two ports. One is the collector port, to which lineprobe
clients connect and send data corresponding to data streams. The second port is
a simple http server that allows multiple viewers of the collected data in
real-time.

Style options.

    axis:axisname:axis_xlabel:axis_ylabel
    label:linename
    color:name
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"io/ioutil"

	"github.com/stephen-soltesz/go/collection"
  //"github.com/vdobler/chart"
  "github.com/stephen-soltesz/chart"
)

var (
	bindAddress = flag.String("hostname", "localhost", "Interface to bind to.")
	viewerPort = flag.Int("viewer_port", 8080, "Port for web viewer.")
	collectorPort = flag.Int("collector_port", 3131, "Port for data collector.")
	timestamp = flag.Bool("timestamp", false, "Use timestamps as x-axis.")
	debug = flag.Bool("debug", false, "Enable debug messages on stderr.")
	plotWidth = flag.Int("plot_width", 600, "Plot width in pixels.")
	plotHeight = flag.Int("plot_height", 400, "Plot height in pixels.")
	debugLogger *log.Logger
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

	collector := collection.Default()
	go startCollectorServer(*bindAddress, *collectorPort, collector)
	go startViewServer(*bindAddress, *viewerPort)
	select {}
}
