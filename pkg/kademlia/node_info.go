package kademlia

import "github.com/linoss-7/D7024E-Project/pkg/utils"

type NodeInfo struct {
	IP        string
	Port      int
	ID        utils.BitArray
	LastSeen  int64 // Unix timestamp of when the node was last seen
	FirstSeen int64 // Unix timestamp of when the node was first seen
}
