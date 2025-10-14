package common

import (
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/network"
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
	// Defensive: ignore contacts with empty IDs. This can happen when callers
	// construct a NodeInfo from config/CLI that doesn't include the remote ID.
	// Calling Get() on an empty BitArray will panic, so bail out early.
	if nodeInfo.ID.Size() == 0 {
		//logrus.Warnf("NewContact called with empty ID for %s:%d, ignoring", nodeInfo.IP, nodeInfo.Port)
		return
	}
	// Find the appropriate bucket for the new contact
	diffBit := 0

	for i := 0; i < 160; i++ {
		if rt.OwnerNodeInfo.ID.Get(i) != nodeInfo.ID.Get(i) {
			diffBit = i
			break
		}
	}

	bucketIndex := 159 - diffBit

	// Check if node already exists in bucket
	rt.bucketLock.Lock()
	bucket := rt.buckets[bucketIndex]
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
			rt.buckets[bucketIndex] = newBucket
			rt.bucketLock.Unlock()
			return
		}
	}

	// Check if bucket is full
	if len(bucket) < rt.k {
		// Otherwise add node to end of bucket
		rt.buckets[bucketIndex] = append(bucket, nodeInfo)
		rt.bucketLock.Unlock()
		return
	}

	// Make a snapshot copy of the bucket while still holding the lock so we can
	// inspect it after releasing the lock without touching the same backing
	// array (which other goroutines may mutate).
	bucketSnapshot := make([]NodeInfo, len(bucket))
	copy(bucketSnapshot, bucket)
	rt.bucketLock.Unlock()

	// If bucket is full, ping the least recently seen node (work on the snapshot)
	leastRecent := bucketSnapshot[0]
	for _, bn := range bucketSnapshot {
		if bn.LastSeen.Unix() < leastRecent.LastSeen.Unix() {
			leastRecent = bn
		}
	}

	// Ping the least recently seen node
	address := &network.Address{
		IP:   leastRecent.IP,
		Port: leastRecent.Port,
	}

	kademliaMessage := DefaultKademliaMessage(rt.OwnerNodeInfo.ID, nil)
	_, err := rt.rpcSender.SendAndAwaitResponse("ping", *address, kademliaMessage, 2.0)

	if err != nil {
		//logrus.Infof("Node %s did not respond to ping, replacing with new node", leastRecent.IP)
		rt.bucketLock.Lock()
		// Re-read the live bucket under lock and remove leastRecent by id if still present
		live := rt.buckets[bucketIndex]
		idx := -1
		for i, n := range live {
			if n.ID.ToString() == leastRecent.ID.ToString() {
				idx = i
				break
			}
		}
		if idx != -1 {
			newBucket := append(live[:idx], live[idx+1:]...)
			newBucket = append(newBucket, nodeInfo) // add incoming node
			rt.buckets[bucketIndex] = newBucket
		}
		rt.bucketLock.Unlock()
		return
	}

	// Node responded, update its LastSeen and move it to the end of the live bucket.
	rt.bucketLock.Lock()
	live := rt.buckets[bucketIndex]
	// Update leastRecent's LastSeen in the live slice and move it to the end
	// Build a new bucket excluding the old leastRecent entry.
	newBucket := make([]NodeInfo, 0, len(live))
	for _, n := range live {
		if n.ID.ToString() != leastRecent.ID.ToString() {
			newBucket = append(newBucket, n)
		}
	}
	// Append updated leastRecent
	leastRecent.LastSeen = time.Now()
	newBucket = append(newBucket, leastRecent)
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
		for i := range rt.buckets[bucketIndex] {
			closest = append(closest, &rt.buckets[bucketIndex][i])
		}
	}
	rt.bucketLock.RUnlock()

	// Ensure we never try to slice with a negative bound: if we already
	// have >= k nodes from the closest bucket, trim and return early.
	if len(closest) >= rt.k {
		return closest[:rt.k]
	}

	// If we don't have enough nodes, expand to other buckets
	remaining := rt.k - len(closest)
	otherBucketNodes := make([]*NodeInfo, 0)
	bucketOffset := 1

	for len(otherBucketNodes) < remaining && (bucketIndex-bucketOffset >= 0 || bucketIndex+bucketOffset < len(rt.buckets)) {
		// Check lower bucket
		if bucketIndex-bucketOffset >= 0 {
			idx := bucketIndex - bucketOffset
			rt.bucketLock.RLock()
			for i := range rt.buckets[idx] {
				// take address of the slice element directly (avoid loop-var address)
				otherBucketNodes = append(otherBucketNodes, &rt.buckets[idx][i])
				if len(otherBucketNodes) >= remaining {
					break
				}
			}
			rt.bucketLock.RUnlock()
		}
		// Check higher bucket
		if bucketIndex+bucketOffset < len(rt.buckets) {
			idx := bucketIndex + bucketOffset
			rt.bucketLock.RLock()
			for i := range rt.buckets[idx] {
				otherBucketNodes = append(otherBucketNodes, &rt.buckets[idx][i])
				if len(otherBucketNodes) >= remaining {
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
