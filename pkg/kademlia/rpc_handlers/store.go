package rpc_handlers

import (
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

type StoreHandler struct {
	IDataStorage common.IDataStorage
	RpcSender    common.IRPCSender
	SenderId     *utils.BitArray
	Ttl          float32
}

func NewStoreHandler(rpcSender common.IRPCSender, dataStorage common.IDataStorage, senderId *utils.BitArray, ttl float32) *StoreHandler {
	return &StoreHandler{
		IDataStorage: dataStorage,
		RpcSender:    rpcSender,
		SenderId:     senderId,
		Ttl:          ttl,
	}
}

func (fnh *StoreHandler) Handle(msg network.Message) error {
	// Unmarshal body to kademlia message
	var km proto_gen.KademliaMessage
	if err := proto.Unmarshal(msg.Payload[6:], &km); err != nil {
		return err
	}

	// Extract the value from the body
	value := string(km.Body)

	// Store the value in the data storage
	data := common.DataObject{Data: value, ExpirationDate: time.Now().Add(time.Duration(fnh.Ttl * float32(time.Second)))}
	key, err := fnh.IDataStorage.Store(data)
	if err != nil {
		return err
	}

	// Create a response message and send the key back to the requester
	reply := &proto_gen.KademliaMessage{
		RPCId:    km.RPCId,
		SenderId: fnh.SenderId.ToBytes(),
		Body:     key.ToBytes(),
	}

	fnh.RpcSender.SendRPC("reply", msg.From, reply)
	return nil
}
