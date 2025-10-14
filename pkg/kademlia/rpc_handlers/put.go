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

	//logrus.Printf("Node %s received put request from %s to store value: %s", fmt.Sprintf("%s:%d", msg.To.IP, msg.To.Port), fmt.Sprintf("%s:%d", msg.From.IP, msg.From.Port), value)
	// Store the value using IDataStorage
	key, err := fnh.IDataStorage.StoreInNetwork(value)
	if err != nil {
		return err
	}
	//logrus.Printf("Node %s stored value for key: %s", msg.To.IP+":"+string(rune(msg.To.Port)), key.ToString())

	// Reply with the key where the value is stored. Use the same RPCId so
	// the caller waiting in SendAndAwaitResponse can match the response.
	response := common.DefaultKademliaMessage(*fnh.SenderId, key.ToBytes())
	response.RPCId = km.RPCId

	return fnh.RpcSender.SendRPC("reply", network.Address{IP: msg.From.IP, Port: msg.From.Port}, response)

}
