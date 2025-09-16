package kademlia

import (
	"math/big"
	"sort"
	"sync"

	"github.com/linoss-7/D7024E-Project/pkg/node"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

type RoutingTable struct {
	ownerNode     *node.Node
	ownerNodeInfo NodeInfo
	buckets       [][]NodeInfo
	bucketLock    sync.RWMutex
	k             int // max nodes per bucket
	PendingPings  map[int]NodeInfo
}

func NewRoutingTable(self *node.Node, selfInfo NodeInfo, k int) *RoutingTable {
	return &RoutingTable{
		ownerNode:     self,
		ownerNodeInfo: selfInfo,
		buckets:       make([][]NodeInfo, 0),
		k:             k,
	}
}

func (rt *RoutingTable) InitBuckets(buckets [][]NodeInfo) {
	rt.bucketLock.Lock()
	defer rt.bucketLock.Unlock()
	rt.buckets = buckets
}

func (rt *RoutingTable) FindClosest(targetID utils.BitArray) []*NodeInfo {
	var closest []*NodeInfo

	// Find the first differing bit, this gives us the closest bucket where we should start

	diffBit := 0

	for i := 0; i < 160; i++ {
		if rt.ownerNodeInfo.ID.Get(i) != targetID.Get(i) {
			diffBit = i
			break
		}
	}

	bucketIndex := 159 - diffBit

	//log.Printf("Finding closest nodes to %s, starting at bucket %d\n", targetID.ToString(), bucketIndex)

	// Add all nodes from the closest bucket
	rt.bucketLock.RLock()
	if bucketIndex < len(rt.buckets) {
		for _, nodeInfo := range rt.buckets[bucketIndex] {
			closest = append(closest, &nodeInfo)
		}
	}
	rt.bucketLock.RUnlock()

	// If we don't have enough nodes, expand to other buckets

	remaining := rt.k - len(closest)
	otherBucketNodes := make([]*NodeInfo, 0)
	bucketOffset := 1

	for len(otherBucketNodes) < remaining && (bucketIndex-bucketOffset >= 0 || bucketIndex+bucketOffset < len(rt.buckets)) {
		// Check lower bucket
		if bucketIndex-bucketOffset >= 0 {
			rt.bucketLock.RLock()
			for _, nodeInfo := range rt.buckets[bucketIndex-bucketOffset] {
				otherBucketNodes = append(otherBucketNodes, &nodeInfo)
				if len(closest) >= rt.k {
					break
				}
			}
			rt.bucketLock.RUnlock()
		}
		// Check higher bucket
		if bucketIndex+bucketOffset < len(rt.buckets) {
			rt.bucketLock.RLock()
			for _, nodeInfo := range rt.buckets[bucketIndex+bucketOffset] {
				otherBucketNodes = append(otherBucketNodes, &nodeInfo)
				if len(closest) >= rt.k {
					break
				}
			}
			rt.bucketLock.RUnlock()
		}
		bucketOffset++
	}

	// Sort otherBucketNodes by distance to targetID and reduce to remaining
	type candidate struct {
		info     *NodeInfo
		distance *big.Int
	}

	sorted := make([]candidate, 0)
	for _, nodeInfo := range otherBucketNodes {
		xor := nodeInfo.ID.Xor(targetID)
		dist := xor.ToBigInt()
		sorted = append(sorted, candidate{
			info:     nodeInfo,
			distance: dist,
		})
	}

	// Simple bubble sort since k is small

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].distance.Cmp(sorted[j].distance) < 0
	})

	// Convert sorted list to nodeInfo pointers
	otherBucketNodes = make([]*NodeInfo, 0)
	for _, c := range sorted {
		otherBucketNodes = append(otherBucketNodes, c.info)
	}

	if len(sorted) > remaining {
		closest = append(closest, otherBucketNodes[:remaining]...)
	} else {
		closest = append(closest, otherBucketNodes...)
	}

	return closest
}

/*
func leadingZeroBits(b byte) int {
	for i := 0; i < 8; i++ {
		if (b & (1 << (7 - i))) != 0 {
			return i
		}
	}
	return 8
}

/*
func idToBytes(id int) []byte {
	b := make([]byte, 160)
	binary.BigEndian.PutUint64(b, uint64(id))
	return b
}

/*
func (rt *RoutingTable) pingNode(nodeInfo NodeInfo) {
	// Send ping
	addr := network.Address{IP: nodeInfo.IP, Port: nodeInfo.Port}

	rt.ownerNode.Send(addr, "ping", []byte{})
}
*/
