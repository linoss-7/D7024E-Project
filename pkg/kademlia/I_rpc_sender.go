package kademlia

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/rpc_handlers"
	"github.com/linoss-7/D7024E-Project/pkg/network"
)

type IRPCSender interface {
	SendAndAwaitResponse(rpc string, address network.Address, body []byte) (*rpc_handlers.KademliaMessage, error)
	Send(rpc string, address network.Address, body []byte) error
}
