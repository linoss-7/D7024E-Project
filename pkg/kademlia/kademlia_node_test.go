package kademlia

import (
	"bytes"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/rpc_handlers"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

func TestPingAndResponse(t *testing.T) {
	net := network.NewMockNetwork(0.0)

	k := 4
	alpha := 3
	// Set up two nodes

	alice, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8000}, *utils.NewBitArray(160), k, alpha)

	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	bobId := utils.NewBitArray(160)

	// Set a bit to differentiate from Alice Id
	bobId.Set(100, true)

	bob, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8001}, *bobId, k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	bobInfo := common.NodeInfo{
		ID:   *bobId,
		IP:   "127.0.0.1",
		Port: 8001,
	}
	// Define ping handler
	pingHandler := rpc_handlers.NewPingHandler(bob, bobInfo)

	// Register ping handler to node
	bob.Node.Handle("ping", pingHandler.Handle)

	// Alice sends a ping to Bob
	aliceMsg := common.DefaultKademliaMessage(alice.ID, nil)

	response, err := alice.SendAndAwaitResponse("ping", bob.Node.Address(), aliceMsg)
	if err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	if !bytes.Equal(aliceMsg.RPCId, response.RPCId) {
		t.Fatalf("RPC IDs do not match! Sent %x, got %x", aliceMsg.RPCId, response.RPCId)
	}
}

func TestMultiplePings(t *testing.T) {

	k := 4
	alpha := 3
	net := network.NewMockNetwork(0.0)

	// Set up two nodes

	alice, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8000}, *utils.NewBitArray(160), k, alpha)

	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	bobId := utils.NewBitArray(160)

	// Set a bit to differentiate from Alice Id
	bobId.Set(100, true)

	bob, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8001}, *bobId, k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	bobInfo := common.NodeInfo{
		ID:   *bobId,
		IP:   "127.0.0.1",
		Port: 8001,
	}

	// Define ping handler
	pingHandler := rpc_handlers.NewPingHandler(bob, bobInfo)

	// Register ping handler to node
	bob.Node.Handle("ping", pingHandler.Handle)

	respCh := make(chan *proto_gen.KademliaMessage, 10)
	errCh := make(chan error)

	var wg sync.WaitGroup

	// Alice sends two pings to Bob

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			aliceMsg := common.DefaultKademliaMessage(alice.ID, nil)
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
	var responses []*proto_gen.KademliaMessage
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

// Test republishing

func TestRepublishing(t *testing.T) {

	return // Uncomment when store is implemented

	// This test will:
	// Test republishing of a value, begin by storing a value in the network, then add a new node that should store the value when republishing occurs
	// Trigger republishing manually by ticking the republish ticker and check that the new node has the value stored

	// Setup 10 nodes
	k := 4
	alpha := 3
	net := network.NewMockNetwork(0.0)

	nodes := make([]*KademliaNode, 10)

	for i := 0; i < 10; i++ {
		port := 8000 + i
		id := utils.NewRandomBitArray(160)
		node, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: port}, *id, k, alpha)
		if err != nil {
			t.Fatalf("Failed to create Node: %v", err)
		}
		nodes[i] = node
	}

	// Join nodes to the network, each node joing a random node already in the network
	for i := 1; i < 10; i++ {
		// Pick a random node to join
		joinNode := nodes[rand.Intn(i)]
		err := nodes[i].Join(joinNode.Node.Address()) // Uncomment when implemented, should have been done with interfaces
		if err != nil {
			t.Fatalf("Node %d failed to join the network: %v", i, err)
		}
	}

	// Store a value in the network from node 0
	value := "Test string for republishing"
	key, err := nodes[0].StoreInNetwork(value)

	if err != nil {
		t.Fatalf("Failed to store value in network: %v", err)
	}

	// Find the nodes where the value should be stored
	nodesForKey, err := nodes[0].LookUp(key)

	if err != nil {
		t.Fatalf("Failed to lookup nodes for key: %v", err)
	}

	// Find the nodes from the lookup in our list of nodes
	storingNodes := []*KademliaNode{}
	for _, expectedNode := range nodesForKey {
		for _, node := range nodes {
			if node.ID.Equals(expectedNode.ID) {
				storingNodes = append(storingNodes, node)
			}
		}
	}

	tickers := []*MockTicker{}
	// Manually trigger republishing for the stored value
	for _, node := range storingNodes {
		ticker := NewMockTicker()
		node.StartRepublish(key, ticker)
		tickers = append(tickers, ticker)
	}

	// Add a new node to the network with ID close to the key

	// Inefficient copy
	newNodeId := utils.NewBitArrayFromBytes(key.ToBytes(), 160)
	newNodeId.Set(159, !newNodeId.Get(159)) // Flip last bit to make it close but not the same

	newNode, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8002}, *newNodeId, k, alpha)
	if err != nil {
		t.Fatalf("Failed to create new node: %v", err)
	}

	// Join new node to the network
	err = newNode.Join(nodes[0].Node.Address())

	if err != nil {
		t.Fatalf("New node failed to join the network: %v", err)
	}

	// Trigger republishing manually by ticking the republish ticker on the first storing node
	tickers[0].Tick(time.Now())

	// Check that the new node has the value stored, with a mock network this should be instant
	retrievedValue, storingNodeInfo, err := newNode.FindValueInNetwork(key)

	// Check that the retrieved value matches the stored value
	if err != nil {
		t.Fatalf("Failed to retrieve value from network: %v", err)
	}

	if retrievedValue != value {
		t.Fatalf("Retrieved value does not match stored value! Got %s, expected %s", retrievedValue, value)
	}

	// Check that the storing nodes contains the new node
	found := false
	for _, n := range storingNodeInfo {
		if n.ID.Equals(newNode.ID) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("New node %s is not in the storing nodes", newNode.ID.ToString())
	}

}

// Mock ticker for testing

type MockTicker struct {
	Ch     chan time.Time
	mu     sync.Mutex
	closed bool
}

func NewMockTicker() *MockTicker {
	return &MockTicker{Ch: make(chan time.Time)}
}

func (mt *MockTicker) Stop() {
	mt.mu.Lock()
	if !mt.closed {
		close(mt.Ch)
		mt.closed = true
	}
	mt.mu.Unlock()
}
func (mt *MockTicker) Reset() {}
func (mt *MockTicker) C() <-chan time.Time {
	return mt.Ch
}
func (mt *MockTicker) Tick(tm time.Time) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	if mt.closed {
		return
	}
	select {
	case mt.Ch <- tm:
		return
	default:
		return
	}
}
