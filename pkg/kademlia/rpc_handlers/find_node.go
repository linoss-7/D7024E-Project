package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

type FindNodeHandler struct {
	rpcSender common.IRPCSender
	table     *common.RoutingTable
}

func NewFindNodeHandler(rpcSender common.IRPCSender, table *common.RoutingTable) *FindNodeHandler {
	return &FindNodeHandler{
		rpcSender: rpcSender,
		table:     table,
	}
}

func (fnh *FindNodeHandler) Handle(msg network.Message) error {

	// Unmarshal message to KademliaMessage

	var km common.KademliaMessage
	payload := msg.Payload[10:] // Exclude "find_node:" prefix
	if err := proto.Unmarshal(payload, &km); err != nil {
		return err
	}

	// Unmarshal body to node info

	var nodeInfo common.NodeInfoMessage
	if err := proto.Unmarshal(km.Body, &nodeInfo); err != nil {
		return err
	}

	// Search for the closest nodes in the routing table
	id := *utils.NewBitArrayFromBytes(nodeInfo.ID, 160)

	closest := fnh.table.FindClosest(id)

	// Convert closest nodes to NodeInfoMessageList

	nodeInfoList := &common.NodeInfoMessageList{}

	for _, n := range closest {
		nodeInfoList.Nodes = append(nodeInfoList.Nodes, &common.NodeInfoMessage{
			ID:   n.ID.ToBytes(),
			IP:   n.IP,
			Port: int32(n.Port),
		})
	}

	// Marshal NodeInfoMessageList to bytes

	data, err := proto.Marshal(nodeInfoList)
	if err != nil {
		return err
	}

	// Create reply KademliaMessage
	replyMsg := &common.KademliaMessage{
		SenderId: fnh.table.OwnerNodeInfo.ID.ToBytes(),
		Body:     data,
	}

	replyMsg.RPCId = km.RPCId

	addr := msg.From

	// Send response back to the requester
	fnh.rpcSender.SendRPC("reply", addr, replyMsg)
	//logrus.Infof("Node %s handled find_node from %s, replied with %d nodes", fnh.table.OwnerNodeInfo.ID.ToString(), addr.String(), len(closest))
	return nil
}
