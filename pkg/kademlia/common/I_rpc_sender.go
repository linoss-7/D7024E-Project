package common

import (
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
)

type IRPCSender interface {
	SendAndAwaitResponse(rpc string, address network.Address, kademliaMessage *proto_gen.KademliaMessage) (*proto_gen.KademliaMessage, error)
	SendRPC(rpc string, address network.Address, kademliaMessage *proto_gen.KademliaMessage) error
}
