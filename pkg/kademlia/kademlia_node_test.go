package kademlia

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/rpc_handlers"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

func TestPingAndResponse(t *testing.T) {
	net := network.NewMockNetwork()

	// Set up two nodes

	alice, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8000}, utils.NewBitArray(160))

	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	bobId := utils.NewBitArray(160)

	// Set a bit to differentiate from Alice Id
	bobId.Set(100, true)

	bob, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8001}, bobId)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	// Define ping handler
	pingHandler := rpc_handlers.NewPingHandler(alice.Node, alice.ID)

	// Register ping handler to node
	bob.Node.Handle("ping", pingHandler.Handle)

	// Alice sends a ping to Bob
	aliceMsg := alice.DefaultKademliaMessage(nil)

	response, err := alice.SendAndAwaitResponse("ping", bob.Node.Address(), aliceMsg)
	if err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	if !bytes.Equal(aliceMsg.RPCId, response.RPCId) {
		t.Fatalf("RPC IDs do not match! Sent %x, got %x", aliceMsg.RPCId, response.RPCId)
	}
}

func TestMultiplePings(t *testing.T) {
	net := network.NewMockNetwork()

	// Set up two nodes

	alice, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8000}, utils.NewBitArray(160))

	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	bobId := utils.NewBitArray(160)

	// Set a bit to differentiate from Alice Id
	bobId.Set(100, true)

	bob, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8001}, bobId)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	// Define ping handler
	pingHandler := rpc_handlers.NewPingHandler(bob.Node, bob.ID)

	// Register ping handler to node
	bob.Node.Handle("ping", pingHandler.Handle)

	respCh := make(chan *rpc_handlers.KademliaMessage, 10)
	errCh := make(chan error)

	var wg sync.WaitGroup

	// Alice sends two pings to Bob

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			aliceMsg := alice.DefaultKademliaMessage(nil)
			resp, err := alice.SendAndAwaitResponse("ping", bob.Node.Address(), aliceMsg)
			if err != nil {
				t.Errorf("Failed to send ping: %v", err)
				errCh <- err
			}
			respCh <- resp

		}()
	}

	waitCh := make(chan struct{})

	// Wait until all goroutines are done
	go func() {
		wg.Wait()
		close(waitCh)
		close(respCh)
	}()

	select {
	case <-waitCh:
		// All goroutines completed
	case <-errCh:
		t.Fatalf("An error occurred during pinging")
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout waiting for pings to complete")
	}

	// Store the responses from goroutines
	var responses []*rpc_handlers.KademliaMessage
	for i := 0; i < 10; i++ {
		select {
		case resp := <-respCh:
			responses = append(responses, resp)
		case err := <-errCh:
			t.Errorf("Error occurred: %v", err)
		}
	}

	// Check that all responses have unique RPC IDs and length is 10
	rpcIdMap := make(map[string]bool)
	if len(responses) != 10 {
		t.Fatalf("Expected 10 responses, got %d", len(responses))
	}
	for _, resp := range responses {
		rpcIdStr := string(resp.RPCId)
		if rpcIdMap[rpcIdStr] {
			t.Errorf("Duplicate RPC ID found: %x", resp.RPCId)
		}
		rpcIdMap[rpcIdStr] = true
	}
}
