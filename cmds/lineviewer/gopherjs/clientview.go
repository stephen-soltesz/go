package main

import "github.com/gopherjs/gopherjs/js"
import "github.com/gopherjs/jquery"

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

	addCanvas(containerName, "mycanvas", 600, 400);

	websocket := js.Global.Get("WebSocket")
	if (websocket != nil) {
		conn = websocket.New("ws://{{$}}/ws")
		conn.Set("onclose", func (evt js.Object) {
			appendLog(jQuery("<div><b>Connection closed.</b></div>"))
		})
		conn.Set("onmessage", func (evt js.Object) {
			uri := evt.Get("data").String()
			//println(uri)
			updateCanvas("#mycanvas", uri)
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
