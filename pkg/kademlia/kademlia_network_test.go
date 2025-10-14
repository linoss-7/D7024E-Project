package kademlia

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

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

		node, err := NewKademliaNode(net, addr, *utils.NewRandomBitArray(160), k, alpha, 10.0)
		if err != nil {
			return fmt.Errorf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
	}

	logrus.Infof("Created %d nodes", numNodes)

	// Setup waitgroup for joining
	var wg sync.WaitGroup
	// Use channels to avoid blocking
	errCh := make(chan error, numNodes-1)

	// Join nodes to the network, join random node in network
	for i := 1; i < numNodes; i++ {
		// Pick a random node to join
		joinNode := nodes[rand.Intn(i)]
		joinNodeInfo := common.NodeInfo{
			IP:   joinNode.Node.Address().IP,
			Port: joinNode.Node.Address().Port,
			ID:   joinNode.ID,
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errCh <- nodes[i].Join(joinNodeInfo)
			logrus.Infof("Node %d joined the network via node with ID %d", i, joinNodeInfo.ID.ToBigInt())
		}(i)
		//logrus.Infof("Node %d joined the network via node with ID %d", i, joinNodeInfo.ID.ToBigInt())
	}

	// Wait for all joins to complete
	wg.Wait()
	close(errCh)

	// Drain errors, if any occur, propagate
	for err := range errCh {
		if err != nil {
			return fmt.Errorf("Error during join: %v", err)
		}
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
	storeWg := sync.WaitGroup{}
	storeErrCh := make(chan error)
	storedValues := make(chan string, 100)

	// Add 100 values to the network from random nodes
	for i := 0; i < 100; i++ {
		storeWg.Add(1)
		go func() {
			defer storeWg.Done()
			// Generate random string of length 30
			value := RandomString(30)

			storingNode := nodes[rand.Intn(numNodes)]
			_, err := storingNode.StoreInNetwork(value)
			if err != nil {
				storeErrCh <- fmt.Errorf("Failed to store value in network: %v", err)
				return
			}
			storedValues <- value
		}()
	}

	storeWg.Wait()
	close(storeErrCh)
	close(storedValues)

	// Drain store errors, if any occur, propagate
	for err := range storeErrCh {
		if err != nil {
			return fmt.Errorf("Error during store: %v", err)
		}
	}

	logrus.Infof("All values stored in the network")
	for value := range storedValues {
		addedValues = append(addedValues, value)
	}

	// Wait for a bit to allow values to propagate
	time.Sleep(2 * time.Second)

	// Log the number of added values
	logrus.Infof("Added %d values to the network", len(addedValues))

	// Have another random node retrieve each value and ensure it matches, concurrent gets
	getWg := sync.WaitGroup{}
	getErrCh := make(chan error)

	for _, value := range addedValues {
		getWg.Add(1)
		go func(value string) {
			defer getWg.Done()

			retrievingNode := nodes[rand.Intn(numNodes)]

			key := utils.ComputeHash(value, 160)

			retrievedValue, _, err := retrievingNode.FindValueInNetwork(key)

			if err != nil {
				getErrCh <- fmt.Errorf("Failed to retrieve value from network: %v", err)
				return
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
				getErrCh <- fmt.Errorf("Value not found in network, closest nodes are: %s", strings.Join(nodesStrings, ", "))
				return

			}

			if retrievedValue != value {
				getErrCh <- fmt.Errorf("Retrieved value does not match stored value. Got %s, expected %s", retrievedValue, value)
			}
		}(value)
	}

	getWg.Wait()
	close(getErrCh)

	// Drain get errors, if any occur, propagate
	for err := range getErrCh {
		if err != nil {
			return fmt.Errorf("Error during get: %v", err)
		}
	}
	return nil
}

func TestStoreGetInNetwork(t *testing.T) {
	// Create a mock network with no message loss
	net := network.NewMockNetwork(0.0)
	// Set parameters k and alpha
	k := 5
	alpha := 3

	// Create 50 nodes

	numNodes := 50
	nodes := make([]*KademliaNode, numNodes)
	for i := 0; i < numNodes; i++ {
		// Create address for node
		addr := network.Address{
			IP:   fmt.Sprintf("node-%d", i+1),
			Port: 8000,
		}
		node, err := NewKademliaNode(net, addr, *utils.NewRandomBitArray(160), k, alpha, 3600.0)
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node

	} // Join nodes to the network, each node joing a random node already in the network
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
			t.Fatalf("Node %d failed to join the network: %v", i, err)
		}
	}

	// Pick a random node to store a value
	storingNode := nodes[rand.Intn(numNodes)]

	value := "Hello, world!"
	key, err := storingNode.StoreInNetwork(value)
	if err != nil {
		t.Fatalf("Failed to store value in network: %v", err)
	}

	// Wait a bit for the value to propagate
	time.Sleep(2 * time.Second)

	// Check which nodes now have a value stored by using the safe API
	found := false
	storingNodes := []*KademliaNode{}
	for _, node := range nodes {
		// Look into storage
		/*
			node.valueMutex.RLock()
			//realVal, exists := node.Values[key.ToString()]

				if exists {
					logrus.Infof("Node %s has value %s for key %s in local storage", node.Node.Address().IP, realVal, key.ToString())
				}

			node.valueMutex.RUnlock()
		*/

		val, err := node.FindValue(key)
		if err != nil {
			t.Fatalf("Failed to find value on node: %v", err)
		}
		if val != "" {
			// Log and verify
			//logrus.Infof("Node %s has value for key %s", node.Node.Address().IP, key.ToString())
			if val != value {
				t.Fatalf("Value mismatch on node: expected %s, got %s", value, val)
			}
			found = true
			storingNodes = append(storingNodes, node)
		}
	}
	if !found {
		t.Fatalf("Value not found on any node")
	}
	// Log the key we are looking for
	//logrus.Infof("Looking for key: %s", key.ToString())

	// Perform a lookup for the key and ensure that the same nodes are returned as storing nodes
	lookupNode := nodes[rand.Intn(numNodes)]
	closestNodes, err := lookupNode.LookUp(key)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	// Check that we found the expected number of nodes
	if len(closestNodes) != k {
		t.Fatalf("Expected %d closest nodes, got %d", k, len(closestNodes))
	}

	// Check that the storing nodes are the same as the closest nodes
	if len(storingNodes) != len(closestNodes) {
		t.Fatalf("Number of storing nodes (%d) does not match number of closest nodes (%d)", len(storingNodes), len(closestNodes))
	}

	for _, sn := range storingNodes {
		found := false
		for _, cn := range closestNodes {
			if sn.ID.ToString() == cn.ID.ToString() {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Storing node %s is not among the closest nodes", sn.ID.ToString())
		}
	}
}

func TestFindValueInNetwork(t *testing.T) {
	net := network.NewMockNetwork(0.0)
	// Set parameters k and alpha
	k := 5
	alpha := 3

	// Create 20 nodes
	numNodes := 20
	nodes := make([]*KademliaNode, numNodes)
	for i := 0; i < numNodes; i++ {
		// Create address for node
		addr := network.Address{
			IP:   fmt.Sprintf("node-%d", i+1),
			Port: 8000,
		}
		node, err := NewKademliaNode(net, addr, *utils.NewRandomBitArray(160), k, alpha, 3600.0)
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
	} // Join nodes to the network, each node joing a random node already in the network
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
			t.Fatalf("Node %d failed to join the network: %v", i, err)
		}
	}

	// Pick a random node to store a value
	storingNode := nodes[rand.Intn(numNodes)]
	value := "Bingus"
	key, err := storingNode.StoreInNetwork(value)
	if err != nil {
		t.Fatalf("Failed to store value in network: %v", err)
	}

	// Pick another random node to get the value
	retrievingNode := nodes[rand.Intn(numNodes)]
	retrValue, retrNodesInfo, err := retrievingNode.FindValueInNetwork(key)
	if err != nil {
		t.Fatalf("Failed to find value in network: %v", err)
	}

	// Ensure the request returned k nodes
	if len(retrNodesInfo) != k {
		t.Fatalf("Expected %d nodes, got %d", k, len(retrNodesInfo))
	}

	if retrValue != value {
		t.Fatalf("Retrieved value does not match stored value. Got %s, expected %s", retrValue, value)
	}

	if len(retrNodesInfo) == 0 {
		t.Fatalf("No nodes returned for key %s", key.ToString())
	}

	retrNodes := []*KademliaNode{}
	// Retrieve the actual nodes from the node info
	for _, n := range retrNodesInfo {
		for _, sn := range nodes {
			if n.ID.ToString() == sn.ID.ToString() {
				retrNodes = append(retrNodes, sn)
				break
			}
		}
	}

	// Ensure that the nodes returned do indeed have the value stored
	for _, n := range retrNodes {
		val, err := n.FindValue(key)
		if err != nil {
			t.Fatalf("Failed to find value in node %s: %v", n.ID.ToString(), err)
		}
		if val != value {
			t.Fatalf("Retrieved value from node %s does not match stored value. Got %s, expected %s", n.ID.ToString(), val, value)
		}
	}
}

func TestLookUpEnsureClosestNodes(t *testing.T) {
	// Check that the lookup function returns the k closest nodes to the target ID
	// Create a mock network with no message loss
	net := network.NewMockNetwork(0.0)
	// Set parameters k and alpha
	k := 5
	alpha := 3

	// Create 50 nodes

	numNodes := 50
	nodes := make([]*KademliaNode, numNodes)
	for i := 0; i < numNodes; i++ {
		// Create address for node
		addr := network.Address{
			IP:   fmt.Sprintf("node-%d", i+1),
			Port: 8000,
		}
		node, err := NewKademliaNode(net, addr, *utils.NewRandomBitArray(160), k, alpha, 10.0)
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node

	} // Join nodes to the network, each node joing a random node already in the network
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
			t.Fatalf("Node %d failed to join the network: %v", i, err)
		}
	}

	// Pick a random node to perform the lookup
	lookupNode := nodes[rand.Intn(numNodes)]

	// Perform the lookup
	targetID := *utils.NewRandomBitArray(160)
	closestNodes, err := lookupNode.LookUp(&targetID)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	// Check that we found the expected number of nodes
	if len(closestNodes) != k {
		t.Fatalf("Expected %d closest nodes, got %d", k, len(closestNodes))
	}

	// Check that the nodes are indeed the closest
	// Compute distances from targetID to all nodes in the network
	type nodeDistance struct {
		node     common.NodeInfo
		distance *utils.BitArray
	}
	allDistances := []nodeDistance{}
	for _, node := range nodes {
		distance := targetID.Xor(node.ID)
		allDistances = append(allDistances, nodeDistance{node: common.NodeInfo{ID: node.ID, IP: node.Node.Address().IP, Port: node.Node.Address().Port}, distance: distance})
	}

	// Sort distances
	for i := 0; i < len(allDistances)-1; i++ {
		for j := i + 1; j < len(allDistances); j++ {
			if allDistances[i].distance.ToBigInt().Cmp(allDistances[j].distance.ToBigInt()) > 0 {
				allDistances[i], allDistances[j] = allDistances[j], allDistances[i]
			}
		}
	}

	// Check that the closestNodes are the k closest in allDistances, ensure each node in closestNodes is in the first k of allDistances
	for _, cn := range closestNodes {
		found := false
		for i := 0; i < k; i++ {
			if cn.ID.ToString() == allDistances[i].node.ID.ToString() {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Node %s is not among the k closest nodes, longest acceptable distance is %s", cn.ID.ToString(), ToSciNot(*allDistances[k-1].distance.ToBigInt()))
		}
	}
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
	return                                             //Disable large test for now
	err := LargeKademliaNetworkTest(1000, 0.02, 10, 5) // 100 nodes, 20% drop rate, k=10, alpha=5
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

func ToSciNot(n big.Int) string {
	s := n.String()
	exponent := len(s) - 1
	if exponent < 4 {
		return s
	}
	if len(s) > 3 {
		return fmt.Sprintf("%s %s", s[:3], "e"+fmt.Sprint(exponent))
	}
	return s
}
