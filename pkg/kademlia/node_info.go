package kademlia

import "github.com/linoss-7/D7024E-Project/pkg/utils"

type NodeInfo struct {
	IP   string
	Port int
	ID   utils.BitArray
}
