package network

import (
	"testing"
)

func TestUDPNetwork_SendRecvListen(t *testing.T) {
	net := NewUDPNetwork()
	addr1 := Address{IP: "127.0.0.1", Port: 9002}
	addr2 := Address{IP: "127.0.0.1", Port: 9003}

	// start listeners

	conn1, err := net.Listen(addr1)
	if err != nil {
		t.Fatalf("Listen failed for addr1: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Listen(addr2)
	if err != nil {
		t.Fatalf("Listen failed for addr2: %v", err)
	}
	defer conn2.Close()

	var msgString = "Hello from conn1"

	// conn1 sends to conn2
	msg := Message{
		From:    addr1,
		To:      addr2,
		Payload: []byte(msgString),
	}
	if err := conn1.Send(msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// conn2 receives
	recvMsg, err := conn2.Recv()
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}
	if string(recvMsg.Payload) != msgString {
		t.Errorf("Received message mismatch. Got: %s, Want: %s", string(recvMsg.Payload), msgString)
	}

	// close connections
	conn1.Close()
	conn2.Close()
}

func TestUDPNetwork_SendRecvDial(t *testing.T) {
	net := NewUDPNetwork()
	addr1 := Address{IP: "127.0.0.1", Port: 9000}
	addr2 := Address{IP: "127.0.0.1", Port: 9001}

	// start listeners

	conn1, err := net.Dial(addr2)
	if err != nil {
		t.Fatalf("Dial failed for addr1: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Listen(addr2)
	if err != nil {
		t.Fatalf("Listen failed for addr2: %v", err)
	}
	defer conn2.Close()

	var msgString = "Hello from conn1"

	// conn1 sends to conn2
	msg := Message{
		From:    addr1,
		To:      addr2,
		Payload: []byte(msgString),
	}
	if err := conn1.Send(msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// conn2 receives
	recvMsg, err := conn2.Recv()
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}
	if string(recvMsg.Payload) != msgString {
		t.Errorf("Received message mismatch. Got: %s, Want: %s", string(recvMsg.Payload), msgString)
	}

	// close connections
	conn1.Close()
	conn2.Close()
}
