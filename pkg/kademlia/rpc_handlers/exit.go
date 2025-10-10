package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
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

	// Reply with an acknowledgment message
	response := common.DefaultKademliaMessage(*eh.SenderId, []byte("Exiting"))

	eh.RpcSender.SendRPC("reply", network.Address{IP: msg.From.IP, Port: msg.From.Port}, response)

	// Exit the node
	return eh.Process.Exit()
}
