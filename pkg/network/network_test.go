package network

import "testing"

func TestNetwork_ReplyString(t *testing.T) {
	net := NewMockNetwork(0.0)

	// Make a new message
	msg := Message{
		From:    Address{IP: "127.0.0.1", Port: 8000},
		To:      Address{IP: "127.0.0.1", Port: 8001},
		Payload: []byte("greet:Hello, Alice!"),
		Network: net,
	}

	// Listen on the From address to receive the reply

	listener1, err := net.Listen(msg.From)
	if err != nil {
		t.Fatalf("Failed to listen on address %v: %v", msg.From, err)
	}
	listener2, err := net.Listen(msg.To)
	if err != nil {
		t.Fatalf("Failed to listen on address %v: %v", msg.To, err)
	}
	conn1, err := net.Dial(msg.To)
	if err != nil {
		t.Fatalf("Failed to dial address %v: %v", msg.To, err)
	}

	defer listener1.Close()
	defer listener2.Close()
	defer conn1.Close()

	// Send the message
	if err := conn1.Send(msg); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for the reply
	reply, err := listener2.Recv()
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	reply.ReplyString("response", "Hello, Bob!")

	// Receive the reply
	finalReply, err := listener1.Recv()

	if err != nil {
		t.Fatalf("Failed to receive final reply: %v", err)
	}

	expectedPayload := "response:Hello, Bob!"
	if string(finalReply.Payload) != expectedPayload {
		t.Errorf("Final reply payload mismatch. Got: %s, Want: %s", string(finalReply.Payload), expectedPayload)
	}

}
