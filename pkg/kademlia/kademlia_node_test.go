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
	"github.com/sirupsen/logrus"
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

func TestRefresh(t *testing.T) {
	return // Uncomment when store is implemented

	// Setup 10 kademlia nodes
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

	// Store a value from the first node
	value := "Test string for refreshing"
	key, err := nodes[0].StoreInNetwork(value)

	// Start refreshing the value on the first node with a mock ticker

	ticker := NewMockTicker()
	nodes[0].StartRefresh(key, ticker)

	// Add a new node to the network that is close to the key

	newNodeId := utils.NewBitArrayFromBytes(key.ToBytes(), 160)
	newNodeId.Set(159, !newNodeId.Get(159)) // Flip last bit to make it close but not the same

	// Refresh the message by ticking the mock ticker

	newNode, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8002}, *newNodeId, k, alpha)
	if err != nil {
		t.Fatalf("Failed to create new node: %v", err)
	}

	// Join new node to the network
	err = newNode.Join(nodes[0].Node.Address())

	if err != nil {
		t.Fatalf("New node failed to join the network: %v", err)
	}

	// Check that the value is still stored in the network by retrieving it from a random node
	retrievingNode := nodes[rand.Intn(10)]

	retrievedValue, storingNodesInfo, err := retrievingNode.FindValueInNetwork(key)
	if err != nil {
		t.Fatalf("Failed to retrieve value from network: %v", err)
	}
	if retrievedValue != value {
		t.Fatalf("Retrieved value does not match stored value! Got %s, expected %s", retrievedValue, value)
	}

	// Ensure that the new node has the value stored
	found := false
	for _, n := range storingNodesInfo {
		if n.ID.Equals(newNode.ID) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("New node %s is not in the storing nodes", newNode.ID.ToString())
	}
}

func TestStoreFindValue(t *testing.T) {
	// Test local storing and getting on kademlia node
	net := network.NewMockNetwork(0.0)
	k := 4
	alpha := 3
	node, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8000}, *utils.NewBitArray(160), k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Kademlia node: %v", err)
	}

	value := "Test string for local store"
	dataObj := common.DataObject{
		Data:           value,
		ExpirationDate: time.Now().Add(1 * time.Hour),
	}

	key, err := node.Store(dataObj)
	if err != nil {
		t.Fatalf("Failed to store data object: %v", err)
	}

	// Try to find the value
	retrievedValue, err := node.FindValue(key)
	if err != nil {
		t.Fatalf("Failed to find value: %v", err)
	}

	if retrievedValue != value {
		t.Fatalf("Retrieved value does not match stored value! Got %s, expected %s", retrievedValue, value)
	}
}

func TestStoreFindExpiredValue(t *testing.T) {
	// Test local storing and getting on kademlia node
	net := network.NewMockNetwork(0.0)

	k := 4
	alpha := 3
	node, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8000}, *utils.NewBitArray(160), k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Kademlia node: %v", err)
	}
	value := "Test string for local store"
	dataObj := common.DataObject{
		Data:           value,
		ExpirationDate: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	key, err := node.Store(dataObj)
	if err != nil {
		t.Fatalf("Failed to store data object: %v", err)
	}

	// Try to find the value
	retrievedValue, err := node.FindValue(key)
	if err != nil {
		t.Fatalf("Failed to find value: %v", err)
	}

	if retrievedValue != "" {
		t.Fatalf("Expected empty string for expired value, got %s", retrievedValue)
	}
}

func TestFindNonExistentValue(t *testing.T) {
	// Test finding a non-existent value on the node
	net := network.NewMockNetwork(0.0)
	k := 4
	alpha := 3
	node, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8000}, *utils.NewBitArray(160), k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Kademlia node: %v", err)
	}
	key := utils.NewRandomBitArray(160)
	retrievedValue, err := node.FindValue(key)
	if err != nil {
		t.Fatalf("Failed to find value: %v", err)
	}
	if retrievedValue != "" {
		t.Fatalf("Expected empty string for non-existent value, got %s", retrievedValue)
	}
}

func TestLookUpFewerNodesThenAlpha(t *testing.T) {
	// Create a mock network with no message loss
	net := network.NewMockNetwork(0.0)

	// Set parameters k and alpha
	k := 20
	alpha := 12 // Alpha is greater than the number of nodes in the network

	localIP := "127.0.0.1"

	// Set up node 1 - Alice
	alice, err := NewKademliaNode(net, network.Address{IP: localIP, Port: 8000}, *utils.NewBitArray(160), k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	// Create Bob's ID
	bobId := utils.NewBitArray(160)
	bobId.Set(110, true)
	bobPort := 8001

	// Set up node 2 - Bob
	bob, err := NewKademliaNode(net, network.Address{IP: localIP, Port: bobPort}, *bobId, k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	// Create Bob's NodeInfo
	bobInfo := common.NodeInfo{
		ID:   *bobId,
		IP:   localIP,
		Port: bobPort,
	}

	// Create Charlie's ID
	charlieId := utils.NewBitArray(160)
	charlieId.Set(120, true)
	charliePort := 8002

	// Set up node 3 - Charlie
	charlie, err := NewKademliaNode(net, network.Address{IP: localIP, Port: charliePort}, *charlieId, k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	// Create Charlie's NodeInfo
	charlieInfo := common.NodeInfo{
		ID:   *charlieId,
		IP:   localIP,
		Port: charliePort,
	}

	// Register find_node handler for Alice, Bob and Charlie

	findNodeHandlerAlice := rpc_handlers.NewFindNodeHandler(alice, alice.RoutingTable)
	alice.Node.Handle("find_node", findNodeHandlerAlice.Handle)

	findNodeHandlerBob := rpc_handlers.NewFindNodeHandler(bob, bob.RoutingTable)
	bob.Node.Handle("find_node", findNodeHandlerBob.Handle)

	findNodeHandlerCharlie := rpc_handlers.NewFindNodeHandler(charlie, charlie.RoutingTable)
	charlie.Node.Handle("find_node", findNodeHandlerCharlie.Handle)

	// Create IDs

	id1 := utils.NewBitArray(160)
	id1.Set(101, true)

	id2 := utils.NewBitArray(160)
	id2.Set(112, true)

	id3 := utils.NewBitArray(160)
	id3.Set(103, true)

	id4 := utils.NewBitArray(160)
	id4.Set(114, true)

	id5 := utils.NewBitArray(160)
	id5.Set(105, true)

	// Create NodeInfos

	info1 := common.NodeInfo{
		ID:   *id1,
		IP:   localIP,
		Port: 8011,
	}

	info2 := common.NodeInfo{
		ID:   *id2,
		IP:   localIP,
		Port: 8012,
	}

	info3 := common.NodeInfo{
		ID:   *id3,
		IP:   localIP,
		Port: 8013,
	}

	info4 := common.NodeInfo{
		ID:   *id4,
		IP:   localIP,
		Port: 8014,
	}

	info5 := common.NodeInfo{
		ID:   *id5,
		IP:   localIP,
		Port: 8015,
	}

	// Add 1,3,5, Bob & Charlie to Alice's routing table
	alice.RoutingTable.NewContact(info1)
	alice.RoutingTable.NewContact(info3)
	alice.RoutingTable.NewContact(info5)
	alice.RoutingTable.NewContact(bobInfo)
	alice.RoutingTable.NewContact(charlieInfo)
	alice.RoutingTable.NewContact(charlieInfo)

	// Add 2,4 & Charlie to Bob's routing table (nodes closer to Bob than Alice)
	bob.RoutingTable.NewContact(info2)
	bob.RoutingTable.NewContact(info4)
	bob.RoutingTable.NewContact(charlieInfo)

	// Give some time for the nodes to initialize
	time.Sleep(100 * time.Millisecond)

	// Alice calls LookUp
	closestNodes, err := alice.LookUp(id4)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	logrus.Infof("Lookup returned %d nodes", len(closestNodes))

	// Check if target node is in the list of closest nodes returned by the lookup

	found := false
	for _, node := range closestNodes {
		if node.ID.Equals(*id4) {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Node not found in the lookup results")
	}

	// Check that the number of nodes returned is less then k
	if len(closestNodes) >= k {
		t.Fatalf("Expected %d nodes, got %d", k, len(closestNodes))
	}
}

func TestLookUpMoreNodesThenAlpha(t *testing.T) {
	// Create a mock network with no message loss
	net := network.NewMockNetwork(0.0)

	// Set parameters k and alpha
	k := 3     // k is less than the number of nodes in the network
	alpha := 2 // alpha is less than the number of nodes in the network

	localIP := "127.0.0.1"

	// Set up node 1 - Alice
	alice, err := NewKademliaNode(net, network.Address{IP: localIP, Port: 8000}, *utils.NewBitArray(160), k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	// Create Bob's ID
	bobId := utils.NewBitArray(160)
	bobId.Set(110, true)
	bobPort := 8001

	// Set up node 2 - Bob
	bob, err := NewKademliaNode(net, network.Address{IP: localIP, Port: bobPort}, *bobId, k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	// Create Bob's NodeInfo
	bobInfo := common.NodeInfo{
		ID:   *bobId,
		IP:   localIP,
		Port: bobPort,
	}

	// Register find_node handler for Alice and Bob

	findNodeHandlerAlice := rpc_handlers.NewFindNodeHandler(alice, alice.RoutingTable)
	alice.Node.Handle("find_node", findNodeHandlerAlice.Handle)

	findNodeHandlerBob := rpc_handlers.NewFindNodeHandler(bob, bob.RoutingTable)
	bob.Node.Handle("find_node", findNodeHandlerBob.Handle)

	// Create IDs

	id1 := utils.NewBitArray(160)
	id1.Set(101, true)

	id2 := utils.NewBitArray(160)
	id2.Set(112, true)

	id3 := utils.NewBitArray(160)
	id3.Set(103, true)

	id4 := utils.NewBitArray(160)
	id4.Set(114, true)

	id5 := utils.NewBitArray(160)
	id5.Set(105, true)

	// Create NodeInfos

	info1 := common.NodeInfo{
		ID:   *id1,
		IP:   localIP,
		Port: 8011,
	}

	info2 := common.NodeInfo{
		ID:   *id2,
		IP:   localIP,
		Port: 8012,
	}

	info3 := common.NodeInfo{
		ID:   *id3,
		IP:   localIP,
		Port: 8013,
	}

	info4 := common.NodeInfo{
		ID:   *id4,
		IP:   localIP,
		Port: 8014,
	}

	info5 := common.NodeInfo{
		ID:   *id5,
		IP:   localIP,
		Port: 8015,
	}

	// Add 1,3,5 & Bob to Alice's routing table
	alice.RoutingTable.NewContact(info1)
	alice.RoutingTable.NewContact(info3)
	alice.RoutingTable.NewContact(info5)
	alice.RoutingTable.NewContact(bobInfo)

	// Add 2,4 to Bob's routing table (nodes closer to Bob than Alice)
	bob.RoutingTable.NewContact(info2)
	bob.RoutingTable.NewContact(info4)

	// Give some time for the nodes to initialize
	time.Sleep(100 * time.Millisecond)

	// Alice calls LookUp
	closestNodes, err := alice.LookUp(id4)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	logrus.Infof("Lookup returned %d nodes", len(closestNodes))

	// Check if target node is in the list of closest nodes returned by the lookup

	found := false
	for _, node := range closestNodes {
		if node.ID.Equals(*id4) {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Node not found in the lookup results")
	}

	// Check that the number of nodes returned is k
	if len(closestNodes) != k {
		t.Fatalf("Expected %d nodes, got %d", k, len(closestNodes))
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
