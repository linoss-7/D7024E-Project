package rpc_handlers

import (
	"testing"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

func TestFindValueHandler_ValidRequest(t *testing.T) {

	k := 4
	// Test find value handler on mock storage and routing table
	storage := &MockStorage{data: make(map[string]common.DataObject)}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	// Make a separate rpc sender for the routing table to avoid message conflicts
	tableRpcSender := &StoringRPCSender{messages: make(map[string][]byte)}
	table := common.NewRoutingTable(tableRpcSender, common.NodeInfo{
		IP:   "localhost",
		Port: 8000,
		ID:   *utils.NewRandomBitArray(160),
	}, k)

	// Add info about 4 nodes to the routing table
	for i := 0; i < k; i++ {
		node := common.NodeInfo{
			IP:   "localhost",
			Port: 8000 + i,
			ID:   *utils.NewRandomBitArray(160),
		}
		table.NewContact(node)
	}

	//logrus.Infof("Closest nodes: %v", table.FindClosest(*utils.NewRandomBitArray(160)))

	handler := NewFindValueHandler(rpcSender, table, storage)
	key := utils.NewRandomBitArray(160)

	// Create a kademlia message looking for the key

	payload, err := proto.Marshal(common.DefaultKademliaMessage(table.OwnerNodeInfo.ID, key.ToBytes()))
	if err != nil {
		t.Fatalf("Failed to marshal find value message: %v", err)
	}
	payload = append([]byte("find_value:"), payload...)

	// Create a mock network message looking for a key
	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}

	// Call the handler
	err = handler.Handle(msg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that a reply message was sent
	if rpcSender.messages["reply"] == nil {
		t.Fatalf("Expected a reply message to be sent")
	}

	// Check that the reply message contains a NodeInfoMessageList
	// (since the storage is empty, it should return closest nodes)
	var nodeList proto_gen.NodeInfoMessageList
	err = proto.Unmarshal(rpcSender.messages["reply"], &nodeList)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that we got k nodes in the reply
	if len(nodeList.Nodes) != k {
		t.Fatalf("Expected %d nodes, got %d", k, len(nodeList.Nodes))
	}

	// Add a value to the storage
	value := "test value"
	data := common.DataObject{
		Data:           value,
		ExpirationDate: time.Now().Add(1 * time.Hour),
	}

	key, err = storage.Store(data)

	if err != nil {
		t.Fatalf("Failed to store value: %v", err)
	}

	// Create a new find value message for the stored key
	payload, err = proto.Marshal(common.DefaultKademliaMessage(table.OwnerNodeInfo.ID, key.ToBytes()))
	if err != nil {
		t.Fatalf("Failed to marshal find value message: %v", err)
	}

	payload = append([]byte("find_value:"), payload...)

	msg = network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: "localhost", Port: 8000},
		Payload: payload,
	}

	// Call the handler again
	err = handler.Handle(msg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that a reply message was sent
	if rpcSender.messages["reply"] == nil {
		t.Fatalf("Expected a reply message to be sent")
	}

	// Check that the reply message contains the value
	if string(rpcSender.messages["reply"]) != value {
		t.Fatalf("Expected value %q, got %q", value, string(rpcSender.messages["reply"]))
	}
}
