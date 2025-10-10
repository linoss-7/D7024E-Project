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

func TestUDPNetwork_DialInvalid(t *testing.T) {
	net := NewUDPNetwork()
	addr := Address{IP: "invalid", Port: 9000} // Invalid IP

	_, err := net.Dial(addr)
	if err == nil {
		t.Fatalf("Dial to invalid address should have failed")
	}
}

func TestUDPNetwork_PartitionHeal(t *testing.T) {
	net1 := NewUDPNetwork()
	net2 := NewUDPNetwork()
	addr1 := Address{IP: "127.0.0.1", Port: 9005}
	addr2 := Address{IP: "127.0.0.1", Port: 9006}

	// start listeners

	conn1, err := net1.Listen(addr1)
	if err != nil {
		t.Fatalf("Listen failed for addr1: %v", err)
	}
	defer conn1.Close()

	conn2, err := net2.Listen(addr2)
	if err != nil {
		t.Fatalf("Listen failed for addr2: %v", err)
	}
	defer conn2.Close()

	// simulate network partition
	net1.Partition([]Address{addr1}, []Address{addr2})

	var msgString = "Hello from conn1"

	// conn1 sends to conn2
	msg := Message{
		From:    addr1,
		To:      addr2,
		Payload: []byte(msgString),
	}
	if err := conn1.Send(msg); err == nil {
		t.Fatalf("Send should have failed due to network partition")
	}

	// restore network
	net1.Heal()

	// conn1 sends to conn2 again
	if err := conn1.Send(msg); err != nil {
		t.Fatalf("Send failed after healing: %v", err)
	}
	// conn2 receives
	recvMsg, err := conn2.Recv()
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}
	if string(recvMsg.Payload) != msgString {
		t.Errorf("Received message mismatch. Got: %s, Want: %s", string(recvMsg.Payload), msgString)
	}

	// Partition network 2 and ensure it does not recieve
	net2.Partition([]Address{addr2}, []Address{addr1})

	msg2 := Message{
		From:    addr1,
		To:      addr2,
		Payload: []byte("Hello from conn1"),
	}
	if err := conn1.Send(msg2); err != nil {
		t.Fatalf("Send failed when other network is partitioned: %v", err)
	}

	// conn2 should not receive
	_, err = conn2.Recv()
	if err == nil {
		t.Fatalf("Recv should have failed due to network partition")
	}

	// heal network 2 and ensure it can recieve again
	net2.Heal()
	if err := conn1.Send(msg2); err != nil {
		t.Fatalf("Send failed after healing: %v", err)
	}
	recvMsg, err = conn2.Recv()
	if err != nil {
		t.Fatalf("Recv failed after healing: %v", err)
	}
	if string(recvMsg.Payload) != "Hello from conn1" {
		t.Errorf("Received message mismatch. Got: %s, Want: %s", string(recvMsg.Payload), "Hello from conn1")
	}
}
