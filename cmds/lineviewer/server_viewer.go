package main

import (
	"bytes"
	//  "encoding/base64"
	"encoding/json"
	//  "errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	// third-party
	"github.com/gorilla/websocket"
	"github.com/stephen-soltesz/go/collection"
)

func startViewServer(host string, port int) {
	//	go hub.run()
	addr := fmt.Sprintf("%s:%d", host, port)
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	http.HandleFunc("/svg", serveSvg)
	http.HandleFunc("/config", serveConfig)
	debugLogger.Printf("HTTP: listen on: %s\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// hub maintains the set of active connections and broadcasts messages to the
// connections.
/*
type hubset struct {
	// Registered connections.
	connections map[*connection]bool

	// Inbound messages from the connections.
	broadcast chan []byte

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection
}

var hub = hubset{
	broadcast:   make(chan []byte),
	register:    make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]bool),
}

func (h *hubset) run() {
	for {
		select {
		case c := <-hub.register:
			hub.connections[c] = true
		case c := <-hub.unregister:
			delete(hub.connections, c)
			close(c.send)
		case m := <-hub.broadcast:
			for c := range hub.connections {
				select {
				case c.send <- m:
				default:
					close(c.send)
					delete(hub.connections, c)
				}
			}
		}
	}
}
*/

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// directory of local resources
	localPrefix = "resources"
)

// connection is an middleman between the websocket connection and the hub.
type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
/*
func (c *connection) readPump() {
	defer func() {
		hub.unregister <- c
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		hub.broadcast <- message
	}
}
*/

// write writes a message with the given message type and payload.
func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

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

/*
func pngToUri(data []byte) []byte {
	var uridata bytes.Buffer

	encoded := base64.StdEncoding.EncodeToString(data)
	//data:image/svg+xml;charset=utf-8;base64,
	//uridata.WriteString("data:image/png;base64,")
	uridata.WriteString("data:image/svg+xml;charset=utf-8;base64,")
	uridata.WriteString(encoded)

	return uridata.Bytes()
}

func getPngUri() []byte {
	var pngimg bytes.Buffer
	collector := collection.Default()
	if err := collector.Plot(&pngimg, *plotWidth, *plotHeight, 0); err != nil {
		return nil
	}
	return pngToUri(pngimg.Bytes())
}

func (c *connection) genMessages() {
	count := 0
	for {
		count += 1
		msg := getPngUri()
		if msg == nil {
			panic(errors.New("failed to convert png"))
		}
		c.write(websocket.TextMessage, msg)
		time.Sleep(time.Second)
	}
}
*/

// writePump pumps messages from the hub to the websocket connection.
/*
func (c *connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
*/

// serverWs handles webocket requests from the peer.
func serveWs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "Origin not allowed", 403)
		return
	}
	debugLogger.Printf("ws:GET %s\n", r.Host)
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		debugLogger.Printf("ws:Error %s %s\n", r.Host, err)
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		debugLogger.Printf("ws:Error %s %s\n", r.Host, err)
		return
	}
	c := &connection{send: make(chan []byte, 256), ws: ws}
	//hub.register <- c
	debugLogger.Printf("ws:Starting read/write pumps %s\n", r.Host)
	//go c.writePump()
	// send canvas size as first message.
	msgStr := fmt.Sprintf("%d,%d\n", *plotWidth, *plotHeight)
	c.write(websocket.TextMessage, []byte(msgStr))
	//go c.genMessages()
	//c.readPump()
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
