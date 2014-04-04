package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	//"math/rand"
	"github.com/stephen-soltesz/go/collection"
	"github.com/stephen-soltesz/go/lineserver"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var configPattern = regexp.MustCompile("axis|reset|exit|color|label")
var configError = errors.New("Matches configPattern")

type CollectorClient struct {
	reader    *bufio.ReadWriter
	collector *collection.Collection
	axis      *collection.Axis
	line      *collection.Line
	id        int
}

func startCollectorServer(host string, port int, collector *collection.Collection) {
	addr := fmt.Sprintf("%s:%d", host, port)
	serv := lineserver.NewServer(addr)
	client_count := 0
	for {
		client_count += 1
		reader, err := serv.Accept()
		if err == io.EOF {
			break
		} else if err != nil {
			// TODO: what other errors can be handled here?
			debugLogger.Println(err)
			panic(err)
		}
		client := CollectorClient{}
		client.reader = reader
		client.collector = collector
		client.id = client_count
		go handleClient(&client)
	}
}

func getNextXvalue(last_value float64) float64 {
	var x float64
	if *timestamp {
		x = float64(time.Now().Unix())
	} else {
		x = last_value + 1.0
	}
	return x
}

func (client *CollectorClient) readSettings(val string) {
	fields := strings.Split(val, ":")
	if len(fields) == 1 {
		// single command
		if fields[0] == "EXIT" {
			os.Exit(0)
		} else if fields[0] == "RESET" {
			fmt.Println("NOT YET SUPPORTED")
		} else {
			// unknown command
			fmt.Println("Unknown command.", fields[0])
		}
	} else if len(fields) >= 2 {
		// this is a key-value setting.
		if fields[0] == "axis" {
			debugLogger.Print("CLIENT: axis name: ", fields[1])
			client.axis = client.collector.GetAxis(fields[1])
			if len(fields) >= 4 {
				client.axis.XLabel = fields[2]
				client.axis.YLabel = fields[3]
			}
		} else if fields[0] == "label" {
			if client.axis != nil {
				debugLogger.Print("CLIENT: label name: ", fields[1])
				client.line = client.axis.GetLine(fields[1])
			}
		} else if fields[0] == "color" {
			if client.line != nil {
				debugLogger.Print("CLIENT: color: ", fields[1])
				client.line.SetColor(fields[1])
			}
		}
	}
}

func (client *CollectorClient) getNextYvalue() (float64, error) {
	val, err := client.reader.ReadString('\n')
	debugLogger.Print("CLIENT: received: ", val)
	if err != nil {
		return 0.0, err
	}
	if len(val) > 0 && !((val[0] >= '0' && val[0] <= '9') || val[0] == '.' || val[0] == '-') {
		// read settings
		debugLogger.Print("CLIENT: reading settings: ", val)
		client.readSettings(strings.TrimSpace(val))
		return client.getNextYvalue()
	} else if y, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err != nil {
		ferr := err.(*strconv.NumError)
		return 0.0, ferr.Err
	} else {
		return y, nil
	}
}

func handleClient(client *CollectorClient) {
	debugLogger.Println("handleClient")

	x := 0.0
	for {
		debugLogger.Println("getting xy vals")
		x = getNextXvalue(x)
		y, err := client.getNextYvalue()
		if err == io.EOF {
			debugLogger.Println("Client EOF")
			break
		} else if err == strconv.ErrSyntax || err == strconv.ErrRange {
			// ignore parse errors.
			debugLogger.Println("Ignoring parse error:", err)
			continue
		} else if err != nil {
			// all other errors. TODO: are any fatal?
			debugLogger.Println(err)
			continue
		}
		if client.axis == nil {
			client.axis = client.collector.GetAxis("default")
		}
		if client.line == nil {
			client.line = client.axis.GetLine(fmt.Sprintf("Thread-%d", client.id))
		}
		client.line.Append(x, y)
	}
}

/*
func randRange(min, max int) float64 {
    y := float64(rand.Intn(max-min))
    return float64(min)+y
}

func newFilter(size int) (func(float64) float64) {
  b := make([]float64, size)
  i := 0
  return func(f float64) float64 {
    b[i%len(b)] = f
    i++
    sum := 0.0
    for _, value := range b {
          sum += value
    }
    return sum/float64(len(b))
  }
}

func generateData(min, max int) {
	axis := collection.Default().GetAxis("default")
	line := axis.AddLine("Thread-gen")
  filt := newFilter(3)
  count := 0.0
  for {
    //ts := float64(time.Now().Unix()/10)
    y := filt(randRange(min, max))
    // mock client: add a new point every second.
    line.Append(count, y)
    count++
    time.Sleep(time.Second)
  }
}
*/
