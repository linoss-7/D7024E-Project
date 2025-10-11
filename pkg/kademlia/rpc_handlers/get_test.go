package rpc_handlers

import (
	"fmt"
	"testing"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

func TestGetHandler_GetValue(t *testing.T) {
	// Test get handler on mock storage
	storage := &MockStorage{data: make(map[string]common.DataObject)}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	handler := NewGetHandler(rpcSender, storage, utils.NewRandomBitArray(160))

	// Store a value in the mock storage
	value := "Test string"
	key, err := storage.StoreInNetwork(value)
	if err != nil {
		t.Fatalf("Failed to store value in mock storage: %v", err)
	}

	// Create a get request message, from an arbitrary node
	messengerId := *utils.NewRandomBitArray(160)
	getMsg := common.DefaultKademliaMessage(messengerId, key.ToBytes())

	// Wrap message in network.Message

	payload, err := proto.Marshal(getMsg)
	if err != nil {
		t.Fatalf("Failed to marshal get message: %v", err)
	}

	// Prepend "get:" to the payload
	payload = append([]byte("get:"), payload...)
	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}
	// Run the get handler
	err = handler.Handle(msg)
	if err != nil {
		t.Fatalf("Failed to handle get message: %v", err)
	}

	// Check for reply message in rpcSender
	valueBytes := rpcSender.messages["reply"]
	if valueBytes == nil {
		t.Fatalf("Expected reply message not found")
	}

	// Check that the value in the reply matches the stored value

	// Unmarshal the ValueAndNodesMessage
	var vanm proto_gen.ValueAndNodesMessage
	err = proto.Unmarshal(valueBytes, &vanm)
	if err != nil {
		t.Fatalf("Failed to unmarshal ValueAndNodesMessage: %v", err)
	}

	// Check that the value in the reply matches the stored value
	if vanm.Value != value {
		t.Fatalf("Expected value %s, got %s", value, vanm.Value)
	}
}

func TestGetHandler_InvalidMsg(t *testing.T) {
	// Test get handler on mock storage
	storage := &MockStorage{data: make(map[string]common.DataObject)}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	handler := NewGetHandler(rpcSender, storage, utils.NewRandomBitArray(160))

	// Create an invalid get request message
	payload := []byte("invalid data")

	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}

	err := handler.Handle(msg)

	if err == nil {
		t.Fatalf("Expected handler to return error for invalid request, but got nil")
	}
}

func TestGetHandler_NotFound(t *testing.T) {
	// Test get handler on mock storage
	storage := &MockStorage{data: make(map[string]common.DataObject)}
	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	handler := NewGetHandler(rpcSender, storage, utils.NewRandomBitArray(160))
	// Create a get request message for a non-existent key
	messengerId := *utils.NewRandomBitArray(160)
	nonExistentKey := utils.NewRandomBitArray(160)
	getMsg := common.DefaultKademliaMessage(messengerId, nonExistentKey.ToBytes())

	// Wrap message in network.Message

	payload, err := proto.Marshal(getMsg)
	if err != nil {
		t.Fatalf("Failed to marshal get message: %v", err)
	}
	payload = append([]byte("get:"), payload...)

	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}

	err = handler.Handle(msg)
	if err != nil {
		t.Fatalf("Failed to handle get message: %v", err)
	}

	// Check that the reply message has an empty body
	reply, exists := rpcSender.messages["reply"]
	if !exists {
		t.Fatalf("Expected reply message not found")
	}

	// Unmarshal the ValueAndNodesMessage
	var vanm proto_gen.ValueAndNodesMessage
	err = proto.Unmarshal(reply, &vanm)

	// An empty body indicates that the value was not found
	if vanm.Value != "" {
		t.Fatalf("Expected empty value for non-existent key, got %s", vanm.Value)
	}
}

type MockStorage struct {
	data map[string]common.DataObject
}

func (ms *MockStorage) Store(value common.DataObject) (*utils.BitArray, error) {
	key := utils.ComputeHash(value.Data, 160)
	ms.data[key.ToString()] = value
	return key, nil
}

func (ms *MockStorage) StoreInNetwork(value string) (*utils.BitArray, error) {
	data := common.DataObject{Data: value, ExpirationDate: time.Now()}
	return ms.Store(data)
}

func (ms *MockStorage) FindValue(key *utils.BitArray) (string, error) {
	value, exists := ms.data[key.ToString()]
	if !exists {
		return "", nil // No error, just return empty string
	}
	return value.Data, nil
}

func (ms *MockStorage) FindValueInNetwork(key *utils.BitArray) (string, []common.NodeInfo, error) {
	value, err := ms.FindValue(key)
	var nodes []common.NodeInfo
	// Generate dummy nodes
	for i := 0; i < 3; i++ {
		nodes = append(nodes, common.NodeInfo{
			ID:   *utils.NewRandomBitArray(160),
			IP:   fmt.Sprintf("node-%d", i+1),
			Port: 8000,
		})
	}
	return value, nodes, err
}
