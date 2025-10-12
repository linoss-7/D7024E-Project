package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

type FindValueHandler struct {
	rpcSender   common.IRPCSender
	table       *common.RoutingTable
	dataStorage common.IDataStorage
}

func NewFindValueHandler(rpcSender common.IRPCSender, table *common.RoutingTable, dataStorage common.IDataStorage) *FindValueHandler {
	return &FindValueHandler{
		rpcSender:   rpcSender,
		table:       table,
		dataStorage: dataStorage,
	}
}
func (fvh *FindValueHandler) Handle(msg network.Message) error {
	// 1. Unmarshal KademliaMessage
	var km proto_gen.KademliaMessage
	payload := msg.Payload[11:] // e.g., exclude "find_value:" prefix
	if err := proto.Unmarshal(payload, &km); err != nil {
		return err
	}

	// 2. Extract the key being searched for
	key := utils.NewBitArrayFromBytes(km.Body, 160)

	// 3. Check local storage
	if value, err := fvh.dataStorage.FindValue(key); value != "" {
		// Node has the value
		body := []byte(value)
		if err != nil {
			return err
		}

		reply := &proto_gen.KademliaMessage{
			RPCId:    km.RPCId,
			SenderId: fvh.table.OwnerNodeInfo.ID.ToBytes(),
			Body:     body,
		}

		fvh.rpcSender.SendRPC("reply", msg.From, reply)
		return nil
	}

	// 4. Value not found → send closest nodes
	closest := fvh.table.FindClosest(*key)

	//logrus.Infof("Closest nodes to %v: %v", key, closest)

	nodeList := &proto_gen.NodeInfoMessageList{}
	for _, n := range closest {
		nodeList.Nodes = append(nodeList.Nodes, &proto_gen.NodeInfoMessage{
			ID:   n.ID.ToBytes(),
			IP:   n.IP,
			Port: int32(n.Port),
		})
	}

	body, err := proto.Marshal(nodeList)
	if err != nil {
		return err
	}

	reply := &proto_gen.KademliaMessage{
		RPCId:    km.RPCId,
		SenderId: fvh.table.OwnerNodeInfo.ID.ToBytes(),
		Body:     body,
	}

	fvh.rpcSender.SendRPC("reply", msg.From, reply)
	return nil
}
