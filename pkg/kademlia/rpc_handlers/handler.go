package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
)

type Handler interface {
	Handle(msg *network.Message) error
}

func NewKademliaMessage(RPCId []byte, SenderId []byte, Body []byte) *proto_gen.KademliaMessage {
	//logrus.Infof("Creating KademliaMessage with RPC ID %x from sender ID %x", RPCId, SenderId)
	return &proto_gen.KademliaMessage{
		RPCId:    RPCId,
		SenderId: SenderId,
		Body:     Body,
	}
}
