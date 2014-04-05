package main

import (
	"strconv"
	"strings"

	// third-party
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jquery"
)

var jQuery = jquery.NewJQuery
var document = js.Global.Get("document")

func appendLog(msg jquery.JQuery) {
	var log = jQuery("#log")
	d := log.Underlying().Index(0)
	msg.AppendTo(log)
	scrollTop := d.Get("scrollTop").Int()
	scrollHeight := d.Get("scrollHeight").Int()
	clientHeight := d.Get("clientHeight").Int()
	doScroll := (scrollTop < scrollHeight - clientHeight)
	if doScroll {
		d.Set("scrollTop", scrollHeight - clientHeight)
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

func addCanvas(containerName, canvasName string, width, height int) {
	canvas := document.Call("createElement", "canvas")
	canvas.Set("id", canvasName)
	canvas.Set("width", width)
	canvas.Set("height", height)
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

func newWebSocket(url string) js.Object {
	websocket := js.Global.Get("WebSocket")
	if websocket != nil {
		return websocket.New(url)
	}
	return nil
}

func wsOnClose(evt js.Object) {
		appendLog(jQuery("<div><b>Connection closed.</b></div>"))
}

var firstRun = true
func wsOnMessage(containerName string, evt js.Object) {
	if firstRun {
		sizeString := strings.TrimSpace(evt.Get("data").String())
		setupCanvas(containerName, sizeString)
		firstRun = false
	} else {
		uri := evt.Get("data").String()
		updateCanvas("#mycanvas", uri)
	}
}

// opens a websocket to socketUrl and adds a canvas to containerName
func setupSocket(socketUrl, containerName string) {
	conn := newWebSocket(socketUrl)
	if conn == nil {
		appendLog(jQuery("<div><b>Your browser does not support WebSockets.</b></div>"))
		return
	}
	conn.Set("onclose", wsOnClose)
	conn.Set("onmessage", func(evt js.Object) {
		wsOnMessage(containerName, evt)
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
	js.Global.Set("setupSocket", setupSocket)
	js.Global.Set("addCanvas", addCanvas)
	js.Global.Set("updateCanvas", updateCanvas)
	js.Global.Set("appendLog", appendLog)
	js.Global.Set("saveImage", saveImage)
}
