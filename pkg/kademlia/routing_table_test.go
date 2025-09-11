package kademlia

import (
	"log"
	"math/big"
	"math/rand"
	"sort"
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/node"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

func TestFullTable(t *testing.T) {
	net := network.NewMockNetwork()

	node, err := node.NewNode(net, network.Address{IP: "localhost", Port: 8000})

	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	nodeInfo := NodeInfo{
		IP:   "localhost",
		Port: 8000,
		ID:   utils.NewBitArray(160),
	}

	rt := NewRoutingTable(node, nodeInfo, 4)

	table := make([][]NodeInfo, 0)

	// Fill the table with nodes, each bucket should contain random nodes within the range 2^i to 2^(i+1)
	for i := 0; i < 160; i++ {
		bucket := make([]NodeInfo, 0)
		for j := 0; j < 4; j++ {
			// Generate a random ID within the range of the i-th bucket
			id := generateValidId(i)

			//log.Printf("Generated ID: %s for bucket %d, index %d\n", id.ToString(), i, j)

			nodeInfo := NodeInfo{
				IP:   "localhost",
				Port: 8000 + j,
				ID:   id,
			}
			bucket = append(bucket, nodeInfo)
		}
		table = append(table, bucket)
	}

	rt.InitBuckets(table)

	// Test finding closest nodes, generate a random i
	for i := 0; i < 10; i++ {
		iVal := rand.Intn(159)
		expectedBucket := 159 - iVal

		// Generate a random ID within the range of the i-th bucket
		id := generateValidId(iVal)

		closest := rt.FindClosest(id)

		// Ensure we got k nodes and all are within valid range
		if len(closest) != 4 {
			t.Errorf("Expected 4 closest nodes, got %d", len(closest))
		}

		for _, nodeInfo := range closest {
			// Ensure all xor'd IDs have their first differing bit at iVal
			xor := utils.NewBitArray(160)
			for b := 0; b < 160; b++ {
				xor.Set(b, rt.ownerNodeInfo.ID.Get(b) != nodeInfo.ID.Get(b)) // XOR operation
			}

			for bit := 0; bit < expectedBucket; bit++ {
				if xor.Get(bit) != false { // Bits before expectedBucket bit should be 0, typed out for clarity
					t.Errorf("Node ID %s out of expected range for bucket %d, index %d", nodeInfo.ID.ToString(), iVal, iVal%8)
				}
			}
			if xor.Get(expectedBucket) != true { // The expectedBucket bit should be 1, the rest can be anything
				t.Errorf("Node ID %s out of expected range for bucket %d, index %d", nodeInfo.ID.ToString(), iVal, iVal%8)
			}
		}
	}

}

func TestNonFullBuckets(t *testing.T) {
	net := network.NewMockNetwork()

	node, err := node.NewNode(net, network.Address{IP: "localhost", Port: 8000})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	nodeInfo := NodeInfo{
		IP:   "localhost",
		Port: 8000,
		ID:   utils.NewBitArray(160),
	}

	rt := NewRoutingTable(node, nodeInfo, 4)

	table := make([][]NodeInfo, 0)

	// Initialize empty buckets
	for i := 0; i < 160; i++ {
		bucket := make([]NodeInfo, 0)
		table = append(table, bucket)
	}

	// Add 100 nodes to their appriopriate buckets and keep track of the 4 smallest
	type candidate struct {
		info     NodeInfo
		distance *big.Int
	}

	var candidates []candidate

	for i := 0; i < 100; i++ {
		iVal := rand.Intn(159)
		id := generateValidId(iVal)

		nodeInfo := NodeInfo{
			IP:   "localhost",
			Port: 8000 + i,
			ID:   id,
		}
		// Add to appropriate bucket

		bucket := 159 - iVal
		currentBucket := table[bucket]
		// Only add if bucket not full
		if len(currentBucket) >= 4 {
			continue
		}

		currentBucket = append(currentBucket, nodeInfo)
		table[bucket] = currentBucket

		// XOR distance to zero is just the ID itself as a number
		dist := id.ToBigInt()
		log.Printf("Generated node ID %s with distance %s\n", id.ToString(), dist.String())

		candidates = append(candidates, candidate{
			info:     nodeInfo,
			distance: dist,
		})
	}

	// Sort by distance
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].distance.Cmp(candidates[j].distance) < 0
	})

	log.Printf("Generated %d candidates\n", len(candidates))
	// Smallest 4
	expected := candidates[:4]

	rt.InitBuckets(table)

	// Compare to 0 id
	id := utils.NewBitArray(160)
	closest := rt.FindClosest(id)

	// Ensure we got the 4 smallest nodes
	if len(closest) != 4 {
		t.Errorf("Expected 4 closest nodes, got %d", len(closest))
	}

	for _, nodeInfo := range closest {
		found := false
		for _, smallNode := range expected {
			if nodeInfo.ID.ToString() == smallNode.info.ID.ToString() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Node ID %s not found in smallest nodes", nodeInfo.ID.ToString())
		}
	}

	log.Printf("Smallest nodes IDs:")
	for _, smallNode := range expected {
		log.Printf(" - %s", smallNode.info.ID.ToString())
	}

	log.Printf("Closest nodes IDs:")
	for _, nodeInfo := range closest {
		log.Printf(" - %s", nodeInfo.ID.ToString())
	}

}

func TestFewerThanKNodes(t *testing.T) {
	net := network.NewMockNetwork()
	node, err := node.NewNode(net, network.Address{IP: "localhost", Port: 8000})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	nodeInfo := NodeInfo{
		IP:   "localhost",
		Port: 8000,
		ID:   utils.NewBitArray(160),
	}
	rt := NewRoutingTable(node, nodeInfo, 4)

	table := make([][]NodeInfo, 0)

	// Initialize empty buckets
	for i := 0; i < 160; i++ {
		bucket := make([]NodeInfo, 0)
		table = append(table, bucket)
	}

	nodes := make([]NodeInfo, 0)
	// Add 2 nodes to their appriopriate buckets
	for i := 0; i < 2; i++ {
		// Generate a random i
		iVal := rand.Intn(159)
		id := generateValidId(iVal)

		nodeInfo := NodeInfo{
			IP:   "localhost",
			Port: 8000 + i,
			ID:   id,
		}

		// Add to appropriate bucket

		bucket := 159 - iVal
		currentBucket := table[bucket]
		currentBucket = append(currentBucket, nodeInfo)
		table[bucket] = currentBucket

		nodes = append(nodes, nodeInfo)
	}

	rt.InitBuckets(table)

	// Compare to 0 id
	id := utils.NewBitArray(160)
	closest := rt.FindClosest(id)

	// Ensure we got the 2 nodes
	if len(closest) != 2 {
		t.Errorf("Expected 2 closest nodes, got %d", len(closest))
	}
	for _, nodeInfo := range closest {
		found := false
		for _, node := range nodes {
			if nodeInfo.ID.ToString() == node.ID.ToString() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Node ID %s not found in added nodes", nodeInfo.ID.ToString())
		}
	}
}

func generateValidId(i int) utils.BitArray {
	id := utils.NewBitArray(160)

	// Fill with random values
	for k := 0; k < 160; k++ {
		//log.Printf("Filling byte %d of ID with random value\n", k)
		bit := rand.Intn(2)
		if bit == 1 {
			id.Set(k, true)
		}

	}
	//log.Printf("Generated random ID: %s\n", id.ToString())
	// Set the i-th bit to 1
	id.Set(159-i, true)
	//log.Printf("Set bit %d to 1: %s\n", i, id.ToString())

	// Set higher bits to 0
	for k := 0; k < 160-(i+1); k++ {
		id.Set(k, false)
	}
	//log.Printf("Cleared bits above %d: %s\n", i, id.ToString())
	return id
}
