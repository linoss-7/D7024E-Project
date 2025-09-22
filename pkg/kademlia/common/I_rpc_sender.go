package common

import (
	"github.com/linoss-7/D7024E-Project/pkg/network"
)

type IRPCSender interface {
	SendAndAwaitResponse(rpc string, address network.Address, kademliaMessage *KademliaMessage) (*KademliaMessage, error)
	SendRPC(rpc string, address network.Address, kademliaMessage *KademliaMessage) error
}
