package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	//"time"

	// third-party
	//"github.com/gorilla/websocket"
	"github.com/stephen-soltesz/go/collection"
)

func startViewServer(host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	debugLogger.Printf("HTTP: listen on: %s\n", addr)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/svg", serveSvg)
	http.HandleFunc("/config", serveConfig)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

const (
	// directory of local resources
	localPrefix = "resources"
)

func getSvg(offset, samples int64) []byte {
	var img bytes.Buffer
	collector := collection.Default()
	err := collector.Plot(&img, *plotWidth, *plotHeight, float64(offset), float64(samples))
	if err != nil {
		return nil
	}
	return img.Bytes()
}

type Config struct {
	Success      bool `json:"success"`
	WidthPixels  int  `json:"width"`
	HeightPixels int  `json:"height"`
	PlotSamples  int  `json:"samples"`
}

func serveConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	cfg := &Config{Success: true, WidthPixels: *plotWidth,
		HeightPixels: *plotHeight, PlotSamples: *plotSamples}
	msg, err := json.Marshal(cfg)
	if err != nil {
		http.Error(w, "Server Error", 500)
		return
	}
	w.Write(msg)
	return
}

func getFormValue(formVals url.Values, key string, defVal int64) (int64, error) {
	var val int64
	var err error

	valStr, ok := formVals[key]
	if !ok {
		return defVal, nil
	}

	if len(valStr) > 0 {
		val, err = strconv.ParseInt(valStr[0], 10, 64)
		if err != nil {
			return defVal, err
		}
	} else {
		val = defVal
	}
	return val, nil
}

func splitAll(path string) (string, string, string) {
	dir, file := filepath.Split(path)
	if file == "" {
		file = "home.html"
	}
	ext := strings.Trim(filepath.Ext(file), ".")
	return dir, file, ext
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var err error

	if r.Method != "GET" {
		http.Error(w, "Method nod allowed", 405)
		return
	}
	_, file, ext := splitAll(r.URL.Path)

	resourcefile := fmt.Sprintf("%s/%s", localPrefix, file)
	if *debug {
		data, err = ioutil.ReadFile(resourcefile)
	} else {
		data, err = Asset(resourcefile)
	}

	if err != nil {
		debugLogger.Printf("Error: requested: %s\n", r.URL.Path)
		http.Error(w, "Not found", 404)
		return
	}

	tmpl := template.Must(template.New(resourcefile).Parse(string(data)))
	if ext == "html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	} else if ext == "js" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	} else if ext == "css" {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	tmpl.Execute(w, r.Host)
}

func serveSvg(w http.ResponseWriter, r *http.Request) {
	var err error
	var offset int64
	var samples int64

	if r.Method != "GET" {
		http.Error(w, "Method nod allowed", 405)
		return
	}
	formVals, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	if offset, err = getFormValue(formVals, "offset", 0); err != nil {
		http.Error(w, "Bad Request. Could not convert parameters.", 400)
		return
	}

	if samples, err = getFormValue(formVals, "samples", 240); err != nil {
		http.Error(w, "Bad Request. Could not convert parameters.", 400)
		return
	}

	svg := getSvg(offset, samples)
	if svg != nil {
		if *debug {
			// save the current image to a file.
			fmt.Println("writing debug.svg")
			ioutil.WriteFile("debug.svg", svg, 0644)
		}
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
		w.Write(svg)
	}
}
