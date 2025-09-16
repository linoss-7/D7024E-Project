package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/node"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto" // go get google.golang.org/protobuf (install protobuf)
)

type PingHandler struct {
	Node *node.Node
	ID   utils.BitArray
}

func NewPingHandler(node *node.Node, id utils.BitArray) *PingHandler {
	return &PingHandler{
		Node: node,
		ID:   id,
	}
}

func (ph *PingHandler) Handle(msg network.Message) error {
	// Echo reply with the same RPCId
	//logrus.Infof("Node %s received ping from %s", ph.ID.ToString(), msg.From.String())
	var km KademliaMessage
	payload := msg.Payload[5:] // Exclude "ping:" prefix
	if err := proto.Unmarshal(payload, &km); err != nil {
		return err
	}

	replyMsg := NewKademliaMessage(km.RPCId, ph.ID.ToBytes(), nil)

	body, err := proto.Marshal(replyMsg)
	if err != nil {
		return err
	}

	ph.Node.Send(msg.From, "reply", body)
	return nil
}

// Empty responses and requests for uniformity
type PingRequest struct {
}
type PingResponse struct {
}
