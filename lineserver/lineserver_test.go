package lineserver

import (
	"testing"
)

var clientMessage = "clientV12345\n"
var serverAddress = "localhost:8311"

func TestCreate(t *testing.T) {
	syncAccept = make(chan bool)
	go setupServer(t)
	setupClient(t)
}

func setupClient(t *testing.T) {
	s := NewServer(serverAddress)
	// wait until the server is waiting
	<-syncAccept

	writer := s.Connect()
	_, err := writer.Write([]byte(clientMessage))
	if err != nil {
		t.Errorf("Failed write")
		return
	}
	writer.Flush()

	<-syncAccept
	s.Close()
}

func setupServer(t *testing.T) {
	s := NewServer(serverAddress)
	reader, err := s.Accept()
	if err != nil {
		t.Errorf("Failed accept")
		return
	}

	val, err := reader.ReadString('\n')
	syncAccept <- true
	if err != nil {
		t.Errorf("Failed read %s", err)
		return
	}

	if val != clientMessage {
		t.Errorf("Failed string compare")
		return
	}

	s.Close()
	return
}
