package kademlia

import (
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/rpc_handlers"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

func TestExist(t *testing.T) {
	net := network.NewMockNetwork(0.0)

	k := 20
	alpha := 3

	node, err := NewKademliaNode(net, network.Address{IP: "127.0.0.1", Port: 8000}, *utils.NewBitArray(160), k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Node: %v", err)
	}

	node.Exit()
}

func TestStoreInNetwork(t *testing.T) {
	net := network.NewMockNetwork(0.0)
	localIP := "127.0.0.1"

	k := 20
	alpha := 3
	valueToStore := "Hello, World!"

	// Create a Kademlia node (Alice)
	alice, err := NewKademliaNode(net, network.Address{IP: localIP, Port: 8000}, *utils.NewRandomBitArray(160), k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Alice: %v", err)
	}

	// Create unique Kademlia node for Bob
	bobId := utils.NewBitArray(160)
	bobId.Set(110, true)
	bobPort := 8001

	// Create another Kademlia node (Bob)
	bob, err := NewKademliaNode(net, network.Address{IP: localIP, Port: bobPort}, *bobId, k, alpha)
	if err != nil {
		t.Fatalf("Failed to create Bob: %v", err)
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

	// Add Bob to Alice's routing table
	alice.RoutingTable.NewContact(bobInfo)

	// Store value in the network using Alice
	alice.StoreInNetwork(valueToStore)

	/*
	// Convert value to its corresponding key
	valueKey := utils.ComputeHash(valueToStore, 160)

	// Check if Bob has the value stored
	value, err := bob.FindValue(valueKey)
	if err != nil {
		t.Fatalf("Bob failed to find the value: %v", err)
	}

	// Verify that Bob found the correct value
	if value != valueToStore {
		t.Fatalf("Bob found incorrect value: got %s, want %s", value, valueToStore)
	}
	*/
}
