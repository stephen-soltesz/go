/*
Package lineserver provides convenience functions for string-based, line
communication between server and client.

To setup a server:
    server := lineserver.NewServer("localhost:8080")
    // listen on server address, and block to accept a new client.
    connection, err := server.Accept()

On the client side:
    server := lineserver.NewServer("localhost:8080")
    // connect to a server.
    connection, err := server.Connect()

*/
package lineserver

import (
	"bufio"
	"net"
)

var Version = "0.1"

type Server interface {
	Accept() (*bufio.ReadWriter, error) // for servers.
	Connect() (*bufio.ReadWriter, error) // for clients.
	Close()
}

type implServer struct {
	address    string
	connection net.Conn
	listener   net.Listener
}

var syncAccept chan bool

// NewServer returns a new Server object ready to Connect to or Accept on the
// given address. The address should be in the form "<host>:<port>".
func NewServer(address string) Server {
	return &implServer{address, nil, nil}
}

// For clients, Connect will make a tcp connection to the server address.
func (s *implServer) Connect() (*bufio.ReadWriter, error) {
	var err error
	s.connection, err = net.Dial("tcp", s.address)
	if err != nil {
		return nil, err
	}
	return bufio.NewReadWriter(bufio.NewReader(s.connection),
		bufio.NewWriter(s.connection)), nil
}

// For servers, Accept listens for tcp connections on the server address.
// Then, Accept blocks waiting for a client to connect.
func (s *implServer) Accept() (*bufio.ReadWriter, error) {
	var err error
	if s.listener == nil {
		tcpAddr, err := net.ResolveTCPAddr("tcp", s.address)
		if err != nil {
			return nil, err
		}

		s.listener, err = net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return nil, err
		}

		// used for testing.
		if syncAccept != nil {
			syncAccept <- true
		}
	}

	s.connection, err = s.listener.Accept()
	if err != nil {
		return nil, err
	}

	readwriter := bufio.NewReadWriter(bufio.NewReader(s.connection),
		bufio.NewWriter(s.connection))
	return readwriter, nil
}

func (s *implServer) Close() {
	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}

	if s.connection != nil {
		s.connection.Close()
		s.connection = nil
	}
}
