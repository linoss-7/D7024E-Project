package rpc_handlers

import (
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

type ForgetHandler struct {
	rpcSender     common.IRPCSender
	ownerInfo     common.NodeInfo
	dataRefresher common.IDataRefresher
}

func NewForgetHandler(rpcSender common.IRPCSender, ownerInfo common.NodeInfo, dataRefresher common.IDataRefresher) *ForgetHandler {
	return &ForgetHandler{
		rpcSender:     rpcSender,
		ownerInfo:     ownerInfo,
		dataRefresher: dataRefresher,
	}
}

func (fnh *ForgetHandler) Handle(msg network.Message) error {

	// Unmarshal message to KademliaMessage

	var km proto_gen.KademliaMessage
	payload := msg.Payload[7:] // Exclude "forget:" prefix
	if err := proto.Unmarshal(payload, &km); err != nil {
		return err
	}

	// Unmarshal body to key

	key := utils.NewBitArrayFromBytes(km.Body, 160)

	// Stop refreshing the key
	if err := fnh.dataRefresher.StopRefresh(key); err != nil {
		return err
	}

	// Create reply KademliaMessage with empty body
	replyMsg := common.DefaultKademliaMessage(fnh.ownerInfo.ID, []byte{})

	replyMsg.RPCId = km.RPCId

	addr := msg.From

	// Send response back to the requester
	fnh.rpcSender.SendRPC("reply", addr, replyMsg)
	//logrus.Infof("Node %s handled find_node from %s, replied with %d nodes", fnh.table.OwnerNodeInfo.ID.ToString(), addr.String(), len(closest))
	return nil
}
