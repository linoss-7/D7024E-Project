package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"github.com/sirupsen/logrus"
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

	logrus.Printf("Node %s received get request from %s for key: %x", msg.To.IP+":"+string(rune(msg.To.Port)), msg.From.IP+":"+string(rune(msg.From.Port)), km.Body)
	// Extract the key from the body
	key := km.Body

	// Call IDataStorage to retrieve the value
	value, nodes, err := fnh.IDataStorage.FindValueInNetwork(utils.NewBitArrayFromBytes(key, 160))
	if err != nil {
		logrus.Errorf("Node %s failed to find value for key: %x, error: %v", msg.To.IP+":"+string(rune(msg.To.Port)), key, err)
		return err
	}

	// Create a ValueAndNodesMessage
	valueAndNodes := &proto_gen.ValueAndNodesMessage{
		Value: value,
		Nodes: &proto_gen.NodeInfoMessageList{},
	}

	for _, n := range nodes {
		valueAndNodes.Nodes.Nodes = append(valueAndNodes.Nodes.Nodes, &proto_gen.NodeInfoMessage{
			ID:   n.ID.ToBytes(),
			IP:   n.IP,
			Port: int32(n.Port),
		})
	}
	// Marshal ValueAndNodesMessage to bytes
	body, err := proto.Marshal(valueAndNodes)
	if err != nil {
		return err
	}

	// Create a response message
	response := common.DefaultKademliaMessage(*fnh.SenderId, body)
	response.RPCId = km.RPCId
	logrus.Printf("Node %s replying to get request from %s with value: %s", msg.To.IP+":"+string(rune(msg.To.Port)), msg.From.IP+":"+string(rune(msg.From.Port)), value)

	// Send the response back to the requester
	return fnh.RpcSender.SendRPC("reply", network.Address{IP: msg.From.IP, Port: msg.From.Port}, response)
}
