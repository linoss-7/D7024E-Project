package common

import (
	"math/rand"

	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

func DefaultKademliaMessage(ownId utils.BitArray, body []byte) *KademliaMessage {
	rpcId := generateId()

	//logrus.Infof("Generated new RPC ID %x for node %s", id.ToBytes(), ownId.ToString())

	return &KademliaMessage{
		RPCId:    rpcId.ToBytes(),
		SenderId: ownId.ToBytes(),
		Body:     body,
	}
}

func generateId() *utils.BitArray {
	id := utils.NewBitArray(160)
	// Generate a random 160-bit ID
	for i := 0; i < 160; i++ {
		if rand.Intn(2) == 1 {
			id.Set(i, true)
		}
	}
	return id
}
