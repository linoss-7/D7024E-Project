package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
)

type Handler interface {
	Handle(msg *network.Message) error
}

func NewKademliaMessage(RPCId []byte, SenderId []byte, Body []byte) *common.KademliaMessage {
	//logrus.Infof("Creating KademliaMessage with RPC ID %x from sender ID %x", RPCId, SenderId)
	return &common.KademliaMessage{
		RPCId:    RPCId,
		SenderId: SenderId,
		Body:     Body,
	}
}
