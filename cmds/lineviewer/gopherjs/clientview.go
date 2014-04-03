package main

import "github.com/gopherjs/gopherjs/js"
import "github.com/gopherjs/jquery"
import "strings"
import "strconv"

var jQuery = jquery.NewJQuery

var conn js.Object

func appendLog(msg jquery.JQuery) {
  var log = jQuery("#log")
	d := log.Underlying().Index(0)
	msg.AppendTo(log)
	scrollTop := d.Get("scrollTop").Int()
	scrollHeight := d.Get("scrollHeight").Int()
	clientHeight := d.Get("clientHeight").Int()
	doScroll := (scrollTop < scrollHeight - clientHeight)
	if (doScroll) {
		d.Set("scrollTop", scrollHeight - clientHeight)
	}
}

func addCanvas(containerName, canvasName string, width, height int) {
	document := js.Global.Get("document")
	canvas := document.Call("createElement", "canvas")
	canvas.Set("id", canvasName)
	canvas.Set("width", width)
	canvas.Set("height", height)
	//println("create canvas")
	jQuery(containerName).Prepend(canvas)
}

func updateCanvas(name, uri string) {
	canvas := jQuery(name)
	//println(canvas)
	context := canvas.Underlying().Index(0).Call("getContext", "2d")

	img := js.Global.Get("Image").New()
	img.Set("src", uri)
	img.Call("addEventListener", "load", func() {
		context.Call("drawImage", img, 0, 0)
	}, false)
}

func setupSocket(containerName string) {
	var firstRun = true;

	websocket := js.Global.Get("WebSocket")
	if (websocket != nil) {
		conn = websocket.New("ws://{{$}}/ws")
		conn.Set("onclose", func (evt js.Object) {
			appendLog(jQuery("<div><b>Connection closed.</b></div>"))
		})
		conn.Set("onmessage", func (evt js.Object) {
			if firstRun {
				println("adding canvas")
				sizeString := strings.TrimSpace(evt.Get("data").String())
				println("sizestring:", sizeString)
				sizes := strings.Split(sizeString, ",")
				if len(sizes) == 2 {
					var width int
					var height int
					var err error
					if width, err = strconv.Atoi(sizes[0]); err != nil {
						width = 600
					}
					if height, err = strconv.Atoi(sizes[1]); err != nil {
						height = 400
					}
					println("addCanvas:", width, ":", height)
					addCanvas(containerName, "mycanvas", width, height)
				} else {
					// use defaults
					println("addCanvas: defaults")
					addCanvas(containerName, "mycanvas", 600, 400)
				}
				firstRun = false;
			} else {
				//println(uri)
				uri := evt.Get("data").String()
				updateCanvas("#mycanvas", uri)
			}
		})
	} else {
		appendLog(jQuery("<div><b>Your browser does not support WebSockets.</b></div>"))
	}
}

func saveImage(canvasName, linkName string) {
	url := jQuery(canvasName).Get(0).Call("toDataURL", "image/png")
	url = url.Call("replace", "image/png", "image/octet-stream")
	jQuery(linkName).Get(0).Set("href", url)
}

func main() {
	js.Global.Set("setupSocket", setupSocket)
	js.Global.Set("addCanvas", addCanvas)
	js.Global.Set("updateCanvas", updateCanvas)
	js.Global.Set("appendLog", appendLog)
	js.Global.Set("saveImage", saveImage)
}
