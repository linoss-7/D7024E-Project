package rpc_handlers

import (
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

func TestStoreHandler_ValidRequest(t *testing.T) {
	// Test store handler on mock storage
	storage := &MockStorage{data: make(map[string]common.DataObject)}
	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}
	senderId := utils.NewBitArray(160)

	handler := NewStoreHandler(rpcSender, storage, senderId)

	payload, err := proto.Marshal(common.DefaultKademliaMessage(*senderId, []byte("test payload")))
	if err != nil {
		t.Fatalf("Failed to marshal store message: %v", err)
	}
	// Prepend "store:" to the payload
	payload = append([]byte("store:"), payload...)
	// Create a test message
	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}

	// Call the handler
	if err := handler.Handle(msg); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check that a reply was recieved
	reply := rpcSender.messages["reply"]
	if reply == nil {
		t.Fatalf("expected a reply message, got nil")
	}

	// Check that the value was stored in the mock storage
	if len(storage.data) != 1 {
		t.Fatalf("expected 1 item in storage, got %d", len(storage.data))
	}

	// Check that the key in the reply matches the stored key
	expectedKey := utils.ComputeHash("test payload", 160)
	if !utils.NewBitArrayFromBytes(reply, 160).Equals(*expectedKey) {
		t.Fatalf("expected key %s, got %s", expectedKey.ToString(), utils.NewBitArrayFromBytes(reply, 160).ToString())
	}

	// Check that the value in the storage matches the sent value
	storedValue := storage.data[utils.ComputeHash("test payload", 160).ToString()]
	if storedValue.Data != "test payload" {
		t.Fatalf("expected value %q, got %q", "test payload", storedValue.Data)
	}
}
