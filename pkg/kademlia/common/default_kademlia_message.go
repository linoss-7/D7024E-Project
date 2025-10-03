package common

import (
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

func DefaultKademliaMessage(ownId utils.BitArray, body []byte) *proto_gen.KademliaMessage {
	rpcId := utils.NewRandomBitArray(160)

	//logrus.Infof("Generated new RPC ID %x for node %s", id.ToBytes(), ownId.ToString())

	return &proto_gen.KademliaMessage{
		RPCId:    rpcId.ToBytes(),
		SenderId: ownId.ToBytes(),
		Body:     body,
	}
}
