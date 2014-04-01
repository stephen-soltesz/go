package main

import (
	"errors"
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"path/filepath"
	"strings"
	"time"

	// third-party
	"github.com/gorilla/websocket"
	"github.com/stephen-soltesz/go/collection"
)

func startViewServer(host string, port int) {
	go hub.run()
	addr := fmt.Sprintf("%s:%d", host, port)
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
  debugLogger.Printf("HTTP: listen on: %s\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// hub maintains the set of active connections and broadcasts messages to the
// connections.
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

// write writes a message with the given message type and payload.
func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func pngToUri(data []byte) []byte {
	var uridata bytes.Buffer

	encoded := base64.StdEncoding.EncodeToString(data)
	uridata.WriteString("data:image/png;base64,")
	uridata.WriteString(encoded)

	return uridata.Bytes()
}

func getPngUri() []byte {
	var pngimg bytes.Buffer
	collector := collection.Default()
	if err := collector.Plot(&pngimg, 600, 400, *timestamp); err != nil {
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

// writePump pumps messages from the hub to the websocket connection.
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
	hub.register <- c
	debugLogger.Printf("ws:Starting read/write pumps %s\n", r.Host)
	go c.writePump()
	go c.genMessages()
	c.readPump()
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
	if r.Method != "GET" {
		http.Error(w, "Method nod allowed", 405)
		return
	}
	_, file, ext := splitAll(r.URL.Path)

	resourcefile := fmt.Sprintf("%s/%s", localPrefix, file)
	data, err := Asset(resourcefile)
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
