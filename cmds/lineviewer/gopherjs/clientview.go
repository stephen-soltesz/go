package main

import (
	"math"
	"strconv"
	"strings"

	// third-party
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jquery"
)

var jQuery = jquery.NewJQuery
var document = js.Global.Get("document")
var xOffset float64 = 0
var plotSamples int64 = 240
var updateInterval float64 = 1000

func getUpdateInterval() float64 {
	return updateInterval
}

func appendLog(msg jquery.JQuery) {
	var log = jQuery("#log")
	d := log.Underlying().Index(0)
	msg.AppendTo(log)
	scrollTop := d.Get("scrollTop").Int()
	scrollHeight := d.Get("scrollHeight").Int()
	clientHeight := d.Get("clientHeight").Int()
	doScroll := (scrollTop < scrollHeight-clientHeight)
	if doScroll {
		d.Set("scrollTop", scrollHeight-clientHeight)
	}
}

type Image struct {
	js.Object
}

func newImage(src string) *Image {
	img := document.Call("createElement", "img")
	img.Set("src", src)
	return &Image{img}
}

func (img *Image) addEventListener(event string, capture bool, callback func()) {
	img.Call("addEventListener", event, callback, capture)
}

func getXOffset(inc bool) int64 {
	if inc && xOffset < 0 {
		xOffset += (updateInterval / 1000.0)
	}
	return int64(math.Floor(xOffset))
}

func getPlotSamples() int64 {
	return plotSamples
}

func addCanvas(containerName, canvasName string, width, height int) {
	canvas := document.Call("createElement", "canvas")
	canvas.Set("id", canvasName)
	canvas.Set("width", width)
	canvas.Set("height", height)

	var mouseDown bool = false
	var mousePrevPos js.Object = nil
	document.Set("onmousedown", func(evt js.Object) {
		mouseDown = true
		mousePrevPos = nil
		updateInterval = 200
	})
	document.Set("onmouseup", func(evt js.Object) {
		mouseDown = false
		updateInterval = 1000
	})
	document.Set("onmousemove", func(evt js.Object) {
		if mouseDown {
			// movement attributes are not supported universally.
			if mousePrevPos != nil {
				curr := evt.Get("clientX").Int()
				prev := mousePrevPos.Get("clientX").Int()
				xOffset += float64(curr - prev)
			}
			mousePrevPos = evt
		}
	})
	document.Set("onmousewheel", func(evt js.Object) {
		scroll := evt.Get("wheelDeltaY").Int()
		if scroll > 0 && plotSamples < 1000 {
			plotSamples += 10
		} else if scroll < 0 && plotSamples > 60 {
			plotSamples -= 10
		}
	})
	jQuery(containerName).Prepend(canvas)
}

func updateCanvas(name, uri string) {
	canvas := jQuery(name).Underlying().Index(0)
	context := canvas.Call("getContext", "2d")

	img := newImage(uri)
	img.addEventListener("load", false, func() {
		context.Call("drawImage", img.Object, 0, 0)
	})
}

func setupCanvas(containerName, sizeString string) {
	var width int = 600
	var height int = 400
	var err error

	sizes := strings.Split(sizeString, ",")
	if len(sizes) == 2 {
		if width, err = strconv.Atoi(sizes[0]); err != nil {
			width = 600
		}
		if height, err = strconv.Atoi(sizes[1]); err != nil {
			height = 400
		}
	}
	addCanvas(containerName, "mycanvas", width, height)
}

var firstRun = true

func jsOnConfig(containerName string, data js.Object) {
	if firstRun {
		width := data.Get("width").Int()
		height := data.Get("height").Int()
		plotSamples = data.Get("samples").Int64()
		addCanvas(containerName, "mycanvas", width, height)
		firstRun = false
	}
}

// opens a websocket to socketUrl and adds a canvas to containerName
func setupSocket(socketUrl, containerName string) {
	jquery.GetJSON("/config", func(data js.Object) {
		jsOnConfig(containerName, data)
	})
}

// converts the canvas to an octet-stream downloadable image.
func saveImage(canvasName, linkName string) {
	url := jQuery(canvasName).Get(0).Call("toDataURL", "image/png")
	url = url.Call("replace", "image/png", "image/octet-stream")
	jQuery(linkName).Get(0).Set("href", url)
}

func main() {
	// export function names globally.
	js.Global.Set("getXOffset", getXOffset)
	js.Global.Set("getPlotSamples", getPlotSamples)
	js.Global.Set("getUpdateInterval", getUpdateInterval)
	js.Global.Set("setupSocket", setupSocket)
	js.Global.Set("addCanvas", addCanvas)
	js.Global.Set("updateCanvas", updateCanvas)
	js.Global.Set("appendLog", appendLog)
	js.Global.Set("saveImage", saveImage)
}
