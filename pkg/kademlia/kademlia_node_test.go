package kademlia

import (
	"bytes"
	"sync"
	"testing"
	"time"

	//"github.com/sirupsen/logrus"
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

	//logrus.Infof("Lookup returned %d nodes", len(closestNodes))

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
	k := 3 // k is less than the number of nodes in the network
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
