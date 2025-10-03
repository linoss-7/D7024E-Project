package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

type GetHandler struct {
	IDataStorage common.IDataStorage
	RpcSender    common.IRPCSender
	SenderId     *utils.BitArray
}

func NewGetHandler(rpcSender common.IRPCSender, dataStorage common.IDataStorage, senderId *utils.BitArray) *GetHandler {
	return &GetHandler{
		IDataStorage: dataStorage,
		RpcSender:    rpcSender,
		SenderId:     senderId,
	}
}

func (fnh *GetHandler) Handle(msg network.Message) error {
	// Unmarshal body to kademlia message
	var km proto_gen.KademliaMessage
	if err := proto.Unmarshal(msg.Payload[4:], &km); err != nil {
		return err
	}

	// Extract the key from the body
	key := km.Body

	// Call IDataStorage to retrieve the value
	value, err := fnh.IDataStorage.FindValueInNetwork(utils.NewBitArrayFromBytes(key, 160))
	if err != nil {
		return err
	}

	// Create a response message
	response := common.DefaultKademliaMessage(*fnh.SenderId, []byte(value))

	// Send the response back to the requester
	return fnh.RpcSender.SendRPC("reply", network.Address{IP: msg.From.IP, Port: msg.From.Port}, response)
}
