package kademlia

import (
	"bytes"
	"fmt"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/node"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

type KademliaNode struct {
	Node         *node.Node
	ID           utils.BitArray
	RoutingTable *common.RoutingTable
	Value        map[*utils.BitArray][]common.DataObject
}

func NewKademliaNode(net network.Network, addr network.Address, id utils.BitArray, k int, alpha int) (*KademliaNode, error) {
	// Update to account for k and alpha

	node, err := node.NewNode(net, addr)

	if err != nil {
		return nil, err
	}

	node.Start()

	return &KademliaNode{
		Node: node,
		ID:   id,
	}, nil
}

func (kn *KademliaNode) SendAndAwaitResponse(rpc string, address network.Address, kademliaMessage *common.KademliaMessage) (*common.KademliaMessage, error) {
	// Send a message and block until a message with the same RPCId is received
	responseCh := make(chan *common.KademliaMessage)
	errCh := make(chan error)

	// Create a new message handler for the response
	handlerFunc := func(msg network.Message) error {
		var respMsg common.KademliaMessage
		payload := msg.Payload[6:] // Exclude "reply:" prefix
		if err := proto.Unmarshal(payload, &respMsg); err != nil {
			errCh <- fmt.Errorf("failed to unmarshal response: %v", err)
			return err
		}

		// Check if the RPC IDs match
		if bytes.Equal(respMsg.RPCId, kademliaMessage.RPCId) {
			responseCh <- &respMsg
		}

		return nil
	}

	kn.Node.Handle("reply", handlerFunc)

	// Send message
	kn.SendRPC(rpc, address, kademliaMessage)

	//logrus.Infof("Node %s sent %s to %s, waiting for response...", kn.ID.ToString(), rpc, address.String())
	select {
	case resp := <-responseCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

func (kn *KademliaNode) FindValueInNetwork(key *utils.BitArray) ([]common.DataObject, error) {
	// Dummy implementation, always returns not implemented
	return nil, fmt.Errorf("not implemented")
}

func (kn *KademliaNode) Join(address network.Address) error {
	// Dummy implementation, always returns not implemented
	return fmt.Errorf("not implemented")
}

func (kn *KademliaNode) Store(value common.DataObject) error {
	// Dummy implementation, always returns not implemented
	return fmt.Errorf("not implemented")
}

func (kn *KademliaNode) StoreInNetwork(value string) ([]common.NodeInfo, error) {
	// Dummy implementation, always returns not implemented
	return nil, fmt.Errorf("not implemented")
}

func (kn *KademliaNode) SendRPC(rpc string, addr network.Address, kademliaMessage *common.KademliaMessage) error {
	kademliaMessage.SenderId = kn.ID.ToBytes()

	marshalledMsg, err := proto.Marshal(kademliaMessage)
	if err != nil {
		// Handle error
		return err
	}

	// Send the marshalled message
	//logrus.Infof("Sending message with RPC ID %x to %s", kademliaMessage.RPCId, addr.String())
	kn.Node.Send(addr, rpc, marshalledMsg)
	return nil
}
