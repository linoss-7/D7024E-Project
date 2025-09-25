package common

import (
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

func TestFullTable(t *testing.T) {

	rpcSender := &mockRPCSender{}

	nodeInfo := NodeInfo{
		IP:   "localhost",
		Port: 8000,
		ID:   *utils.NewBitArray(160),
	}

	rt := NewRoutingTable(rpcSender, nodeInfo, 4)

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
				ID:   *id,
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

		closest := rt.FindClosest(*id)

		// Ensure we got k nodes and all are within valid range
		if len(closest) != 4 {
			t.Errorf("Expected 4 closest nodes, got %d", len(closest))
		}

		for _, nodeInfo := range closest {
			// Ensure all xor'd IDs have their first differing bit at iVal
			xor := utils.NewBitArray(160)
			for b := 0; b < 160; b++ {
				xor.Set(b, rt.OwnerNodeInfo.ID.Get(b) != nodeInfo.ID.Get(b)) // XOR operation
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
	mockSender := &mockRPCSender{}
	nodeInfo := NodeInfo{
		IP:   "localhost",
		Port: 8000,
		ID:   *utils.NewBitArray(160),
	}

	// Mock an rpc sender that always gives correct response

	rt := NewRoutingTable(mockSender, nodeInfo, 4)

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
			ID:   *id,
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
		//log.Printf("Generated node ID %s with distance %s\n", id.ToString(), dist.String()

		candidates = append(candidates, candidate{
			info:     nodeInfo,
			distance: dist,
		})
	}

	// Sort by distance
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].distance.Cmp(candidates[j].distance) < 0
	})

	//log.Printf("Generated %d candidates\n", len(candidates))
	// Smallest 4
	expected := candidates[:4]

	rt.InitBuckets(table)

	// Compare to 0 id
	id := utils.NewBitArray(160)
	closest := rt.FindClosest(*id)

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

	/*
		log.Printf("Smallest nodes IDs:")
		for _, smallNode := range expected {
			log.Printf(" - %s", smallNode.info.ID.ToString())
		}

		log.Printf("Closest nodes IDs:")
		for _, nodeInfo := range closest {
			log.Printf(" - %s", nodeInfo.ID.ToString())
		}
	*/

}

func TestFewerThanKNodes(t *testing.T) {

	nodeInfo := NodeInfo{
		IP:   "localhost",
		Port: 8000,
		ID:   *utils.NewBitArray(160),
	}

	mockSender := &mockRPCSender{}
	rt := NewRoutingTable(mockSender, nodeInfo, 4)

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
			ID:   *id,
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
	closest := rt.FindClosest(*id)

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

func TestAddingSingleNode(t *testing.T) {

	for i := 0; i < 10; i++ {
		// Generate a random node ID
		nodeInfo := NodeInfo{
			IP:   "localhost",
			Port: 8000,
			ID:   *utils.NewRandomBitArray(160),
		}

		mockSender := &mockRPCSender{}

		rt := NewRoutingTable(mockSender, nodeInfo, 4)

		// Generate another random ID
		anotherNodeInfo := NodeInfo{
			IP:   "localhost",
			Port: 8001,
			ID:   *utils.NewRandomBitArray(160),
		}

		rt.NewContact(anotherNodeInfo)

		newId := utils.NewRandomBitArray(160)
		closest := rt.FindClosest(*newId)

		// Ensure we got the 1 node
		if len(closest) != 1 {
			t.Errorf("Expected 1 closest node, got %d", len(closest))
		}

		if closest[0].ID.ToString() != anotherNodeInfo.ID.ToString() {
			t.Errorf("Node ID %s not found in added node %s", closest[0].ID.ToString(), anotherNodeInfo.ID.ToString())
		}
	}
}

func TestRespondingFullNode(t *testing.T) {
	firstFour := make([]NodeInfo, 0)

	nodeInfo := NodeInfo{
		IP:   "localhost",
		Port: 8000,
		ID:   *utils.NewBitArray(160),
	}

	mockSender := &mockRPCSender{}

	rt := NewRoutingTable(mockSender, nodeInfo, 4)

	// Generate 100 nodes in the same bucket

	// Select a bucket greater than 5
	bucket := rand.Intn(154) + 5

	for i := 0; i < 100; i++ {
		nodeInfo := NodeInfo{
			IP:   "localhost",
			Port: 8000 + i,
			ID:   *generateValidId(bucket),
		}
		rt.NewContact(nodeInfo)

		if i < 4 {
			firstFour = append(firstFour, nodeInfo)
		}
	}

	// Get an arbitary id

	newId := utils.NewRandomBitArray(160)
	closest := rt.FindClosest(*newId)

	// Ensure we got the 4 nodes
	if len(closest) != 4 {
		t.Errorf("Expected 4 closest nodes, got %d", len(closest))
	}

	for _, nodeInfo := range closest {
		found := false
		for _, firstNode := range firstFour {
			if nodeInfo.ID.ToString() == firstNode.ID.ToString() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Node ID %s not found in first four nodes", nodeInfo.ID.ToString())
		}
	}
}

func TestNonRespondingFullNode(t *testing.T) {
	lastFour := make([]NodeInfo, 0)

	nodeInfo := NodeInfo{
		IP:   "localhost",
		Port: 8000,
		ID:   *utils.NewBitArray(160),
	}

	mockSender := &mockNoResponseRPCSender{}

	rt := NewRoutingTable(mockSender, nodeInfo, 4)

	// Generate 100 nodes in the same bucket

	// Select a bucket greater than 5
	bucket := rand.Intn(154) + 5
	for i := 0; i < 100; i++ {
		nodeInfo := NodeInfo{
			IP:   "localhost",
			Port: 8000 + i,
			ID:   *generateValidId(bucket),
		}
		rt.NewContact(nodeInfo)

		if i >= 96 {
			lastFour = append(lastFour, nodeInfo)
		}
	}

	// Get an arbitary id

	newId := utils.NewRandomBitArray(160)
	closest := rt.FindClosest(*newId)

	// Ensure we got the 4 nodes
	if len(closest) != 4 {
		t.Errorf("Expected 4 closest nodes, got %d", len(closest))
	}

	for _, nodeInfo := range closest {
		found := false
		for _, lastNode := range lastFour {
			if nodeInfo.ID.ToString() == lastNode.ID.ToString() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Node ID %s not found in last four nodes", nodeInfo.ID.ToString())
		}
	}
}

func generateValidId(i int) *utils.BitArray {
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

////////////////////////////////////////////////////////////////////////////////
// Mock senders
////////////////////////////////////////////////////////////////////////////////

type mockRPCSender struct{}

func (m *mockRPCSender) SendAndAwaitResponse(rpc string, address network.Address, kademliaMessage *proto_gen.KademliaMessage) (*proto_gen.KademliaMessage, error) {
	// return a reponse in the opposite direction

	// Make an arbitary rpc id and node ids
	Id := make([]byte, 160)
	SenderId := make([]byte, 160)
	copy(SenderId, address.IP)
	km := proto_gen.KademliaMessage{
		RPCId:    Id,
		SenderId: SenderId,
		Body:     []byte{},
	}

	return &km, nil

	// RPCId []byte, SenderId []byte, Body []byte
}
func (m *mockRPCSender) SendRPC(rpc string, address network.Address, kademliaMessage *proto_gen.KademliaMessage) error {
	return nil
}

type mockNoResponseRPCSender struct{}

func (m *mockNoResponseRPCSender) SendAndAwaitResponse(rpc string, address network.Address, kademliaMessage *proto_gen.KademliaMessage) (*proto_gen.KademliaMessage, error) {
	return nil, fmt.Errorf("no response")
}
func (m *mockNoResponseRPCSender) SendRPC(rpc string, address network.Address, kademliaMessage *proto_gen.KademliaMessage) error {
	return nil
}
