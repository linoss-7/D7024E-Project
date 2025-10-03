package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

type PutHandler struct {
	IDataStorage common.IDataStorage
	RpcSender    common.IRPCSender
	SenderId     *utils.BitArray
}

func NewPutHandler(rpcSender common.IRPCSender, dataStorage common.IDataStorage, senderId *utils.BitArray) *PutHandler {
	return &PutHandler{
		IDataStorage: dataStorage,
		RpcSender:    rpcSender,
		SenderId:     senderId,
	}
}

func (fnh *PutHandler) Handle(msg network.Message) error {
	// Unmarshal body to kademlia message
	var km proto_gen.KademliaMessage
	if err := proto.Unmarshal(msg.Payload[4:], &km); err != nil {
		return err
	}

	// Extract the value from the body

	value := string(km.Body)

	// Store the value using IDataStorage
	key, err := fnh.IDataStorage.StoreInNetwork(value)
	if err != nil {
		return err
	}

	// Reply with the key where the value is stored

	response := common.DefaultKademliaMessage(*fnh.SenderId, key.ToBytes())

	return fnh.RpcSender.SendRPC("reply", network.Address{IP: msg.From.IP, Port: msg.From.Port}, response)

}
