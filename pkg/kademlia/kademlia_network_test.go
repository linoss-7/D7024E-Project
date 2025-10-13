package kademlia

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"strings"
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
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

	logrus.Infof("Created %d nodes", numNodes)

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
		logrus.Infof("Node %d joined the network via node with ID %d", i, joinNodeInfo.ID.ToBigInt())
	}

	logrus.Infof("All nodes joined the network")

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

	logrus.Infof("All values stored in the network")

	// Have another random node retrieve each value and ensure it matches
	for _, value := range addedValues {

		retrievingNode := nodes[rand.Intn(numNodes)]

		key := utils.ComputeHash(value, 160)

		retrievedValue, _, err := retrievingNode.FindValueInNetwork(key)

		if err != nil {
			return fmt.Errorf("Failed to retrieve value: %v", err)
		}

		// Check if response can be marshalled to ValueAndNodesMessage
		vnm := &proto_gen.ValueAndNodesMessage{}
		err = proto.Unmarshal([]byte(retrievedValue), vnm)
		if err == nil {
			// Successfully unmarshalled, this means value was not found
			nodesStrings := []string{}
			for _, n := range vnm.Nodes.Nodes {
				nodesStrings = append(nodesStrings, fmt.Sprintf("%s:%d", n.IP, n.Port))
			}
			return fmt.Errorf("Value not found in network, closest nodes are: %s", strings.Join(nodesStrings, ", "))

		}

		if retrievedValue != value {
			return fmt.Errorf("Retrieved value does not match stored value. Got %s, expected %s", retrievedValue, value)
		}

	}

	return nil
}

func TestProfileLargeKademliaNetwork(t *testing.T) {
	return // Disable profiling for now
	// Create CPU profile file
	f, err := os.Create("cpu.prof")
	if err != nil {
		t.Fatalf("could not create CPU profile: %v", err)
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatalf("could not start CPU profile: %v", err)
	}
	defer pprof.StopCPUProfile()

	// Run the workload you want to profile — reduce sizes so it runs quickly.
	// Example: 30 nodes, low drop rate (so not dominated by retries)
	if err := LargeKademliaNetworkTest(30, 0.0, 5, 3); err != nil {
		t.Fatalf("workload failed: %v", err)
	}
}
func TestLargeKademliaNetwork(t *testing.T) {
	err := LargeKademliaNetworkTest(100, 0.0, 10, 5) // 100 nodes, 0% drop rate, k=10, alpha=5
	if err != nil {
		t.Fatalf("LargeKademliaNetworkTest failed: %v", err)
	}
}

func RandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
