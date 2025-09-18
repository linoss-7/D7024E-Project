package kademlia

import (
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

type NodeInfo struct {
	IP        string
	Port      int
	ID        utils.BitArray
	LastSeen  time.Time // Unix timestamp of when the node was last seen
	FirstSeen time.Time // Unix timestamp of when the node was first seen
}
