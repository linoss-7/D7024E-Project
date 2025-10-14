package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type ExitHandler struct {
	Process   common.IProcess
	RpcSender common.IRPCSender
	SenderId  *utils.BitArray
}

func NewExitHandler(rpcSender common.IRPCSender, process common.IProcess, senderId *utils.BitArray) *ExitHandler {
	return &ExitHandler{
		Process:   process,
		RpcSender: rpcSender,
		SenderId:  senderId,
	}
}

func (eh *ExitHandler) Handle(msg network.Message) error {

	// Unmarshal body to kademlia message (skip "exit:")
	var km proto_gen.KademliaMessage
	if err := proto.Unmarshal(msg.Payload[5:], &km); err != nil {
		return err
	}

	// Reply with an acknowledgment message
	response := common.DefaultKademliaMessage(*eh.SenderId, nil)
	response.RPCId = km.RPCId

	logrus.Infof("Node %s received exit command from %s, exiting...", eh.SenderId.ToString(), msg.From.IP+":"+string(rune(msg.From.Port)))
	eh.RpcSender.SendRPC("reply", msg.From, response)

	// Exit the node
	return eh.Process.Exit()
}
