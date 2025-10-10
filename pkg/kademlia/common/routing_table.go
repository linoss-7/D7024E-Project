package common

import (
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

type RoutingTable struct {
	rpcSender     IRPCSender
	OwnerNodeInfo NodeInfo
	buckets       [][]NodeInfo
	bucketLock    sync.RWMutex
	k             int // max nodes per bucket
}

func NewRoutingTable(rpc_sender IRPCSender, selfInfo NodeInfo, k int) *RoutingTable {
	rt := &RoutingTable{
		rpcSender:     rpc_sender,
		OwnerNodeInfo: selfInfo,
		buckets:       make([][]NodeInfo, 0),
		k:             k,
	}

	// Initialize 160 empty buckets
	for i := 0; i < 160; i++ {
		rt.buckets = append(rt.buckets, make([]NodeInfo, 0))
	}
	return rt

}

func (rt *RoutingTable) InitBuckets(buckets [][]NodeInfo) {
	rt.bucketLock.Lock()
	defer rt.bucketLock.Unlock()
	rt.buckets = buckets
}

func (rt *RoutingTable) NewContact(nodeInfo NodeInfo) {
	// Find the appropriate bucket for the new contact
	diffBit := 0

	for i := 0; i < 160; i++ {
		if rt.OwnerNodeInfo.ID.Get(i) != nodeInfo.ID.Get(i) {
			diffBit = i
			break
		}
	}

	bucketIndex := 159 - diffBit

	// Check if bucket is full
	rt.bucketLock.Lock()
	bucket := rt.buckets[bucketIndex]
	rt.bucketLock.Unlock()
	if len(bucket) < rt.k {
		// Otherwise add node to end of bucket
		rt.buckets[bucketIndex] = append(bucket, nodeInfo)
		return
	}

	// Check if node already exists in bucket
	for i, n := range bucket {
		if n.ID.ToString() == nodeInfo.ID.ToString() {
			// Node already exists, update its LastSeen and move it to the end of the bucket
			bucket[i].LastSeen = time.Now()
			newBucket := make([]NodeInfo, 0)
			for _, n := range bucket {
				if n.ID.ToString() != bucket[i].ID.ToString() {
					newBucket = append(newBucket, n)
				}
			}
			newBucket = append(newBucket, bucket[i])
			rt.bucketLock.Lock()
			rt.buckets[bucketIndex] = newBucket
			rt.bucketLock.Unlock()
			return
		}
	}

	// If bucket is full, ping the least recently seen node
	leastRecent := bucket[0]
	for _, nodeInfo := range bucket {
		if nodeInfo.LastSeen.Unix() < leastRecent.LastSeen.Unix() {
			leastRecent = nodeInfo
		}
	}

	// Ping the least recently seen node

	address := &network.Address{
		IP:   leastRecent.IP,
		Port: leastRecent.Port,
	}

	kademliaMessage := &proto_gen.KademliaMessage{}

	_, err := rt.rpcSender.SendAndAwaitResponse("ping", *address, kademliaMessage)

	if err != nil {
		// Node did note respond, remove the node and add the new node at the end of the bucket
		newBucket := make([]NodeInfo, 0)
		for _, nodeInfo := range bucket {
			if nodeInfo.ID.ToString() != leastRecent.ID.ToString() {
				newBucket = append(newBucket, nodeInfo)
			}
		}

		newBucket = append(newBucket, nodeInfo)
		rt.bucketLock.Lock()
		rt.buckets[bucketIndex] = newBucket
		rt.bucketLock.Unlock()
		return
	}

	// Node responded, update its LastSeen and move it to the end of the bucket
	leastRecent.LastSeen = time.Now()
	newBucket := make([]NodeInfo, 0)
	for _, nodeInfo := range bucket {
		if nodeInfo.ID.ToString() != leastRecent.ID.ToString() {
			newBucket = append(newBucket, nodeInfo)
		}
	}
	newBucket = append(newBucket, leastRecent)
	rt.bucketLock.Lock()
	rt.buckets[bucketIndex] = newBucket
	rt.bucketLock.Unlock()
}

func (rt *RoutingTable) FindClosest(targetID utils.BitArray) []*NodeInfo {
	var closest []*NodeInfo

	// Find the first differing bit, this gives us the closest bucket where we should start

	diffBit := 0

	for i := 0; i < 160; i++ {
		if rt.OwnerNodeInfo.ID.Get(i) != targetID.Get(i) {
			diffBit = i
			break
		}
	}

	bucketIndex := 159 - diffBit

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
