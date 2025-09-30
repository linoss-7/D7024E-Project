package node

import (
	"fmt"
	"testing"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/network"
)

func TestNodeSimple(t *testing.T) {
	// Create network and nodes
	net := network.NewMockNetwork(0.0)
	alice, _ := NewNode(net, network.Address{IP: "127.0.0.1", Port: 8080})
	bob, _ := NewNode(net, network.Address{IP: "127.0.0.1", Port: 8081})
	// Channel for synchronization
	done := make(chan struct{})

	// Alice says hello when she receives a message
	alice.Handle("hello", func(msg network.Message) error {
		fmt.Printf("Alice: Hello %s!\n", msg.From.IP)
		return msg.ReplyString("reply", "Nice to meet you!")
	})

	// Bob prints replies
	bob.Handle("reply", func(msg network.Message) error {
		fmt.Printf("Bob: %s\n", string(msg.Payload)[6:]) // Skip "reply:" prefix
		done <- struct{}{}
		return nil
	})

	// Start nodes and send message
	alice.Start()
	bob.Start()
	bob.SendString(alice.Address(), "hello", "Hi Alice!")

	// Wait for completion and cleanup
	<-done
	alice.Close()
	bob.Close()
	fmt.Println("Done!")
}

func TestHandlers(t *testing.T) {
	// Create network and nodes
	net := network.NewMockNetwork(0.0)
	alice, _ := NewNode(net, network.Address{IP: "127.0.0.1", Port: 8080})
	bob, _ := NewNode(net, network.Address{IP: "127.0.0.1", Port: 8081})
	// Channel for synchronization
	done := make(chan struct{})

	// Alice says hello when she receives a message
	aliceHandler := alice.Handle("hello", func(msg network.Message) error {
		fmt.Printf("Alice: Hello %s!\n", msg.From.IP)
		return msg.ReplyString("reply", "Nice to meet you!")
	})

	// Bob prints replies
	bob.Handle("reply", func(msg network.Message) error {
		fmt.Printf("Bob: %s\n", string(msg.Payload)[6:]) // Skip "reply:" prefix
		done <- struct{}{}
		return nil
	})

	// Remove Alice's handler before sending message
	alice.RemoveHandler("hello", aliceHandler)

	// Start nodes and send message
	alice.Start()
	bob.Start()
	bob.SendString(alice.Address(), "hello", "Hi Alice!")

	select {
	case <-done:
		t.Error("Handler was not removed, message was processed")
	case <-time.After(1 * time.Second):
		// Expected timeout since handler was removed
	}
	alice.Close()
	bob.Close()
	fmt.Println("Done!")
}
