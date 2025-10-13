package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto" // go get google.golang.org/protobuf (install protobuf)
)

type PingHandler struct {
	RPCSender common.IRPCSender
	SelfInfo  common.NodeInfo
}

func NewPingHandler(rpcSender common.IRPCSender, selfInfo common.NodeInfo) *PingHandler {
	return &PingHandler{
		RPCSender: rpcSender,
		SelfInfo:  selfInfo,
	}
}

func (ph *PingHandler) Handle(msg network.Message) error {
	// Echo reply with the same RPCId
	//logrus.Infof("Node %s received ping from %s", ph.ID.ToString(), msg.From.String())
	var km proto_gen.KademliaMessage
	payload := msg.Payload[5:] // Exclude "ping:" prefix
	if err := proto.Unmarshal(payload, &km); err != nil {
		logrus.Infof("Failed to unmarshal ping message: %v", err)
		return err
	}

	replyMsg := NewKademliaMessage(km.RPCId, ph.SelfInfo.ID.ToBytes(), nil)

	replyMsg.RPCId = km.RPCId

	addr := msg.From

	// Send response back to the requester
	ph.RPCSender.SendRPC("reply", addr, replyMsg)
	return nil
}

// Empty responses and requests for uniformity
type PingRequest struct {
}
type PingResponse struct {
}
