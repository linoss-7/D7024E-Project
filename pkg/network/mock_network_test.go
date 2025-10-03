package network

import (
	"testing"
)

func TestMockNetwork_ListenAndDial(t *testing.T) {
	net := NewMockNetwork(0.0)
	addr := Address{IP: "127.0.0.1", Port: 8000}

	_, err := net.Listen(addr)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	// Dial should succeed after Listen
	dialConn, err := net.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	if dialConn == nil {
		t.Fatalf("Dial returned nil connection")
	}
}

func TestMockConnection_SendRecv(t *testing.T) {
	net := NewMockNetwork(0.0)
	addr1 := Address{IP: "127.0.0.1", Port: 8000}
	addr2 := Address{IP: "127.0.0.1", Port: 8001}

	conn1, _ := net.Listen(addr1)
	conn2, _ := net.Listen(addr2)

	msg := Message{From: addr1, To: addr2, Payload: []byte("hello")}
	err := conn1.Send(msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	recvMsg, err := conn2.Recv()
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}
	if string(recvMsg.Payload) != "hello" {
		t.Errorf("Recv got %s, want hello", string(recvMsg.Payload))
	}
}

func TestMockNetwork_Partition(t *testing.T) {
	net := NewMockNetwork(0.0)
	addr1 := Address{IP: "127.0.0.1", Port: 8000}
	addr2 := Address{IP: "127.0.0.1", Port: 8001}

	conn1, _ := net.Listen(addr1)
	conn2, _ := net.Listen(addr2)

	net.Partition([]Address{addr1}, []Address{addr2})

	msg := Message{From: addr1, To: addr2, Payload: []byte("test")}
	err := conn1.Send(msg)
	if err == nil {
		t.Errorf("Send should fail when partitioned")
	}

	net.Heal()

	err = conn1.Send(msg)
	if err != nil {
		t.Fatalf("Send failed after healing: %v", err)
	}

	recvMsg, err := conn2.Recv()
	if err != nil {
		t.Fatalf("Recv failed after healing: %v", err)
	}
	if string(recvMsg.Payload) != "test" {
		t.Errorf("Recv got %s, want test", string(recvMsg.Payload))
	}
}

func TestMockConnection_Close(t *testing.T) {
	net := NewMockNetwork(0.0)
	addr := Address{IP: "127.0.0.1", Port: 8000}
	conn, _ := net.Listen(addr)

	err := conn.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	_, err = conn.Recv()
	if err == nil {
		t.Errorf("Recv should fail after close")
	}
}
