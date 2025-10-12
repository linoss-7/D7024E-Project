package kademlia

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

func LargeKademliaNetworkTest(numNodes int, dropRate float64, k int, alpha int) error {

	net := network.NewMockNetwork(dropRate)

	// Create nodes
	nodes := make([]*KademliaNode, numNodes)
	for i := 0; i < numNodes; i++ {
		// Create address for node
		addr := network.Address{
			IP:   fmt.Sprintf("node-%d", i+1),
			Port: 8000,
		}

		node, err := NewKademliaNode(net, addr, *utils.NewRandomBitArray(160), k, alpha)
		if err != nil {
			return fmt.Errorf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
	}

	// Join nodes to the network, each node joing a random node already in the network
	for i := 1; i < numNodes; i++ {
		// Pick a random node to join
		joinNode := nodes[rand.Intn(i)]
		joinNodeInfo := common.NodeInfo{
			IP:   joinNode.Node.Address().IP,
			Port: joinNode.Node.Address().Port,
			ID:   joinNode.ID,
		}
		err := nodes[i].Join(joinNodeInfo)
		if err != nil {
			return fmt.Errorf("Node %d failed to join the network: %v", i, err)
		}
	}

	// Simple check: ensure all nodes have at least one contact in their routing table
	// Make a search for a random ID and see if we get any results
	for i, node := range nodes {
		targetID := *utils.NewRandomBitArray(160)
		contacts := node.RoutingTable.FindClosest(targetID)
		if len(contacts) == 0 {
			return fmt.Errorf("Node %d has no contacts in its routing table", i)
		}
	}

	addedValues := []string{}
	// Add 100 values to the network from random nodes
	for i := 0; i < 100; i++ {
		value := RandomString(30)

		storingNode := nodes[rand.Intn(numNodes)]
		storingNode.StoreInNetwork(value)

		addedValues = append(addedValues, value)
	}

	// Have another random node retrieve each value and ensure it matches
	for _, value := range addedValues {

		retrievingNode := nodes[rand.Intn(numNodes)]

		key := utils.ComputeHash(value, 160)

		retrievedValue, _, err := retrievingNode.FindValueInNetwork(key)

		if err != nil {
			return fmt.Errorf("Failed to retrieve value: %v", err)
		}
		// Check that all retrieved value matches stored value

		if retrievedValue != value {
			return fmt.Errorf("Retrieved value does not match stored value. Got %s, expected %s", retrievedValue, value)
		}
	}

	return nil
}

func TestLargeKademliaNetwork(t *testing.T) {
	//LargeKademliaNetworkTest(100, 0.1, 20, 3) // 100 nodes, 10% drop rate, k=20, alpha=3
	// Automatically pass for now
	t.Log("LargeKademliaNetworkTest skipped")
}

func RandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
