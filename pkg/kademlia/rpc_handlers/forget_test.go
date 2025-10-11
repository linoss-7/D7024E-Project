package rpc_handlers

import (
	"sync"
	"testing"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

func TestForget_ValidRequest(t *testing.T) {
	// Test forget handler on mock refresher
	refresher := &MockRefresher{refreshedKeys: make(map[string]bool)}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	ownerInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "localhost",
		Port: 8000,
	}

	handler := NewForgetHandler(rpcSender, ownerInfo, refresher)

	// Start refreshing a key
	value := "Test string"
	key := utils.ComputeHash(value, 160)
	if err := refresher.StartRefresh(key, utils.NewRealTimeTicker(100*time.Millisecond)); err != nil {
		t.Fatalf("Failed to start refresh: %v", err)
	}

	// Create a forget request message, from an arbitrary node
	messengerId := *utils.NewRandomBitArray(160)
	forgetMsg := common.DefaultKademliaMessage(messengerId, key.ToBytes())

	// Wrap message in network.Message
	payload, err := proto.Marshal(forgetMsg)
	if err != nil {
		t.Fatalf("Failed to marshal forget message: %v", err)
	}

	// Prepend "forget:" to the payload
	payload = append([]byte("forget:"), payload...)
	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}

	// Run the forget handler
	err = handler.Handle(msg)
	if err != nil {
		t.Fatalf("Failed to handle forget message: %v", err)
	}

	// Check for reply message in rpcSender
	reply := rpcSender.messages["reply"]
	if reply == nil {
		t.Fatalf("Expected reply message not found")
	}

	// Check that the key is no longer being refreshed
	if refresher.IsRefreshing(key) {
		t.Fatalf("Expected key to no longer be refreshed")
	}
}

func TestForget_InvalidMsg(t *testing.T) {
	// Test forget handler on mock refresher
	refresher := &MockRefresher{refreshedKeys: make(map[string]bool)}

	// Create mock refresher
	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	ownerInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "localhost",
		Port: 8000,
	}

	handler := NewForgetHandler(rpcSender, ownerInfo, refresher)

	// Create an invalid forget request message
	payload := []byte("invalid data")
	payload = append([]byte("forget:"), payload...)
	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}

	// Run the forget handler
	err := handler.Handle(msg)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

type MockRefresher struct {
	refreshedKeys  map[string]bool
	refreshedMutex sync.RWMutex
}

func (m *MockRefresher) StartRefresh(key *utils.BitArray, utils utils.Ticker) error {
	m.refreshedMutex.Lock()
	defer m.refreshedMutex.Unlock()
	m.refreshedKeys[key.ToString()] = true
	return nil
}

func (m *MockRefresher) StopRefresh(key *utils.BitArray) error {
	m.refreshedMutex.Lock()
	defer m.refreshedMutex.Unlock()
	m.refreshedKeys[key.ToString()] = false
	return nil
}

func (m *MockRefresher) IsRefreshing(key *utils.BitArray) bool {
	m.refreshedMutex.RLock()
	defer m.refreshedMutex.RUnlock()
	return m.refreshedKeys[key.ToString()]
}
