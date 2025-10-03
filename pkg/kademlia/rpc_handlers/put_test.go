package rpc_handlers

import (
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

func TestPutHandler_PutValue(t *testing.T) {
	// Test put handler on mock storage
	storage := &MockStorage{data: make(map[string]common.DataObject)}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	handler := NewPutHandler(rpcSender, storage, utils.NewRandomBitArray(160))

	// Store a value
	value := "Test value"

	// Create a put request message, from an arbitrary node
	messengerId := *utils.NewRandomBitArray(160)
	putMsg := common.DefaultKademliaMessage(messengerId, []byte(value))

	// Wrap message in network.Message
	payload, err := proto.Marshal(putMsg)

	if err != nil {
		t.Fatalf("Failed to marshal put message: %v", err)
	}

	// Prepend "put:" to the payload
	payload = append([]byte("put:"), payload...)
	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}

	// Run the put handler
	err = handler.Handle(msg)

	if err != nil {
		t.Fatalf("Failed to handle put message: %v", err)
	}

	// Check the key in the reply message
	replyKey := rpcSender.messages["reply"]
	if replyKey == nil {
		t.Fatalf("Expected reply message not found")
	}

	expectedKey := utils.ComputeHash(value, 160)
	// Check that the key in the reply matches the stored key
	if !utils.NewBitArrayFromBytes(replyKey, 160).Equals(*expectedKey) {
		t.Fatalf("Expected key %s, got %s", expectedKey.ToString(), utils.NewBitArrayFromBytes(replyKey, 160).ToString())
	}

	// Check that the value is stored in the mock storage
	storedValue, exists := storage.data[expectedKey.ToString()]
	if !exists {
		t.Fatalf("Expected stored value not found")
	}
	if storedValue.Data != value {
		t.Errorf("Unexpected stored value: got %q, want %q",
			storedValue.Data, value)
	}
}

func TestPutHandler_InvalidMsg(t *testing.T) {
	// Test put handler on mock storage
	storage := &MockStorage{data: make(map[string]common.DataObject)}
	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}
	handler := NewPutHandler(rpcSender, storage, utils.NewRandomBitArray(160))
	// Create an invalid put request message
	payload := []byte("invalid data")
	payload = append([]byte("put:"), payload...)
	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}

	// Run the put handler
	err := handler.Handle(msg)

	if err == nil {
		t.Fatal("Expected error for invalid message, got nil")
	}

	// Check that no value is stored in the mock storage
	if len(storage.data) != 0 {
		t.Fatalf("Expected no values in storage, got %d", len(storage.data))
	}
}
