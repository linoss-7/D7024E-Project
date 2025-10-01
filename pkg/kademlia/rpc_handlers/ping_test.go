package rpc_handlers

import (
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

func TestPing_ValidRequest(t *testing.T) {
	// Set up mock sender to capture sent messages
	net := network.NewMockNetwork(0.0)

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	selfInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "localhost",
		Port: 8000,
	}

	handler := NewPingHandler(rpcSender, selfInfo)

	// Create a find_node request message,, from an arbitrary node

	messengerId := *utils.NewRandomBitArray(160)

	// Create KademliaMessage

	pingMsg := common.DefaultKademliaMessage(messengerId, nil)

	// Wrap message in network.Message

	payload, err := proto.Marshal(pingMsg)
	if err != nil {
		t.Fatalf("Failed to marshal KademliaMessage: %v", err)
	}

	payload = append([]byte("ping:"), payload...)

	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: selfInfo.IP, Port: selfInfo.Port},
		Payload: payload,
		Network: net,
	}

	err = handler.Handle(msg)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Response should be stored in rpcSender

	_, exists := rpcSender.messages["reply"]
	if !exists {
		t.Fatalf("Expected reply not found")
	}
}
