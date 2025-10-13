package rpc_handlers

import (
	"fmt"
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

func TestExitHandler_ExitNode(t *testing.T) {
	return
	// Test exit handler on mock process
	process := &MockProcess{Closed: false}

	selfInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "127.0.0.1",
		Port: 8080,
	}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	handler := NewExitHandler(rpcSender, process, &selfInfo.ID)

	km := common.DefaultKademliaMessage(selfInfo.ID, nil)

	payload, err := proto.Marshal(km)
	if err != nil {
		t.Fatalf("Failed to marshal KademliaMessage: %v", err)
	}

	// Create exit message
	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		Payload: payload,
	}

	// Run the exit handler
	handler.Handle(msg)

	// Check for exit message in rpcSender
	_, exists := rpcSender.messages["reply"]
	if !exists {
		t.Fatalf("Expected exit message not found")
	}

	// Check if node is closed
	if !process.Closed {
		t.Fatalf("Expected process to be closed")
	}
}

type MockProcess struct {
	Closed bool
}

func (mp *MockProcess) Exit() error {
	if mp.Closed {
		return fmt.Errorf("Process already closed")
	}
	mp.Closed = true
	return nil
}
