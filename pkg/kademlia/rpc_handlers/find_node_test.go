package rpc_handlers

import (
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

func TestValidFindNodeRequest(t *testing.T) {

	net := network.NewMockNetwork(0.0)

	selfInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "localhost",
		Port: 8000,
	}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	// Setup routing table and mock rpc sender

	table := common.NewRoutingTable(rpcSender, selfInfo, 4)

	// Insert 4 random nodes in the routing table

	insertedNodes := make(map[string]bool)
	for i := 0; i < 4; i++ {
		id := *utils.NewRandomBitArray(160)
		address := network.Address{IP: "localhost", Port: 8001 + i}
		nodeInfo := common.NodeInfo{ID: id, IP: address.IP, Port: address.Port}
		table.NewContact(nodeInfo)
		insertedNodes[id.ToString()] = true
	}

	handler := NewFindNodeHandler(rpcSender, table)

	// Create a find_node request message,, from an arbitrary node

	targetId := *utils.NewRandomBitArray(160)
	messengerId := *utils.NewRandomBitArray(160)

	// Marshal targetId into NodeInfoMessage

	nodeInfo := &proto_gen.NodeInfoMessage{
		ID:   targetId.ToBytes(),
		IP:   "localhost",
		Port: 9000,
	}

	nodeInfoMessage, err := proto.Marshal(nodeInfo)
	if err != nil {
		t.Fatalf("Failed to marshal NodeInfoMessage: %v", err)
	}

	// Create KademliaMessage

	findNodeMsg := common.DefaultKademliaMessage(messengerId, nodeInfoMessage)

	// Wrap message in network.Message

	payload, err := proto.Marshal(findNodeMsg)
	if err != nil {
		t.Fatalf("Failed to marshal KademliaMessage: %v", err)
	}

	payload = append([]byte("find_node:"), payload...)

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

	responseData, exists := rpcSender.messages["reply"]
	if !exists {
		t.Fatalf("Expected reply not found")
	}

	// Unmarshal the response and ensure it contains the 4 nodes in our routing table

	var nodeInfoList proto_gen.NodeInfoMessageList
	if err := proto.Unmarshal(responseData, &nodeInfoList); err != nil {
		t.Fatalf("Failed to unmarshal NodeInfoMessageList: %v", err)
	}

	if len(nodeInfoList.Nodes) != 4 {
		t.Fatalf("Expected 4 nodes in response, got %d", len(nodeInfoList.Nodes))
	}

	for _, n := range nodeInfoList.Nodes {
		if !insertedNodes[utils.NewBitArrayFromBytes(n.ID, 160).ToString()] {
			t.Fatalf("Received unexpected node ID %s", utils.NewBitArrayFromBytes(n.ID, 160).ToString())
		}
	}
}

func TestInvalidRequest(t *testing.T) {

	net := network.NewMockNetwork(0.0)

	selfInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "localhost",
		Port: 8000,
	}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	// Setup routing table and mock rpc sender

	table := common.NewRoutingTable(rpcSender, selfInfo, 4)

	// Insert 4 random nodes in the routing table

	insertedNodes := make(map[string]bool)
	for i := 0; i < 4; i++ {
		id := *utils.NewRandomBitArray(160)
		address := network.Address{IP: "localhost", Port: 8001 + i}
		nodeInfo := common.NodeInfo{ID: id, IP: address.IP, Port: address.Port}
		table.NewContact(nodeInfo)
		insertedNodes[id.ToString()] = true
	}

	handler := NewFindNodeHandler(rpcSender, table)

	// Create invalid kademlia message
	payload := []byte("invalid data")
	payload = append([]byte("find_node:"), payload...)

	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: selfInfo.IP, Port: selfInfo.Port},
		Payload: payload,
		Network: net,
	}

	err := handler.Handle(msg)

	if err == nil {
		t.Fatalf("Expected handler to return error for invalid request, but got nil")
	}
}

func TestEmptyRoutingTable(t *testing.T) {

	net := network.NewMockNetwork(0.0)

	selfInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "localhost",
		Port: 8000,
	}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	// Setup routing table and mock rpc sender

	table := common.NewRoutingTable(rpcSender, selfInfo, 4)

	handler := NewFindNodeHandler(rpcSender, table)

	// Create a find_node request message,, from an arbitrary node

	targetId := *utils.NewRandomBitArray(160)
	messengerId := *utils.NewRandomBitArray(160)

	// Marshal targetId into NodeInfoMessage

	nodeInfo := &proto_gen.NodeInfoMessage{
		ID:   targetId.ToBytes(),
		IP:   "localhost",
		Port: 9000,
	}

	nodeInfoMessage, err := proto.Marshal(nodeInfo)
	if err != nil {
		t.Fatalf("Failed to marshal NodeInfoMessage: %v", err)
	}

	// Create KademliaMessage

	findNodeMsg := common.DefaultKademliaMessage(messengerId, nodeInfoMessage)

	// Wrap message in network.Message

	payload, err := proto.Marshal(findNodeMsg)
	if err != nil {
		t.Fatalf("Failed to marshal KademliaMessage: %v", err)
	}

	payload = append([]byte("find_node:"), payload...)

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

	// Response should be stored in rpcSender, ensure it's empty

	responseData, exists := rpcSender.messages["reply"]
	if !exists {
		t.Fatalf("Expected reply not found")
	}

	// Unmarshal the response and ensure it contains 0 nodes

	var nodeInfoList proto_gen.NodeInfoMessageList
	if err := proto.Unmarshal(responseData, &nodeInfoList); err != nil {
		t.Fatalf("Failed to unmarshal NodeInfoMessageList: %v", err)
	}

	if len(nodeInfoList.Nodes) != 0 {
		t.Fatalf("Expected 0 nodes in response, got %d", len(nodeInfoList.Nodes))
	}
}

func TestInvalidNodeInfo(t *testing.T) {

	net := network.NewMockNetwork(0.0)

	selfInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "localhost",
		Port: 8000,
	}

	rpcSender := &StoringRPCSender{messages: make(map[string][]byte)}

	// Setup routing table and mock rpc sender

	table := common.NewRoutingTable(rpcSender, selfInfo, 4)

	// Insert 4 random nodes in the routing table

	insertedNodes := make(map[string]bool)
	for i := 0; i < 4; i++ {
		id := *utils.NewRandomBitArray(160)
		address := network.Address{IP: "localhost", Port: 8001 + i}
		nodeInfo := common.NodeInfo{ID: id, IP: address.IP, Port: address.Port}
		table.NewContact(nodeInfo)
		insertedNodes[id.ToString()] = true
	}

	handler := NewFindNodeHandler(rpcSender, table)

	// Create a find_node request message,, from an arbitrary node

	messengerId := *utils.NewRandomBitArray(160)

	// Marshal targetId into NodeInfoMessage

	nodeInfoMessage := []byte("invalid data")

	// Create KademliaMessage

	findNodeMsg := common.DefaultKademliaMessage(messengerId, nodeInfoMessage)

	// Wrap message in network.Message

	payload, err := proto.Marshal(findNodeMsg)
	if err != nil {
		t.Fatalf("Failed to marshal KademliaMessage: %v", err)
	}

	payload = append([]byte("find_node:"), payload...)

	msg := network.Message{
		From:    network.Address{IP: "localhost", Port: 9000},
		To:      network.Address{IP: selfInfo.IP, Port: selfInfo.Port},
		Payload: payload,
		Network: net,
	}

	err = handler.Handle(msg)
	if err == nil {
		t.Fatalf("Expected handler to return error for invalid NodeInfoMessage, but got nil")
	}
}

// Mock RPC sender that stores sent messages for inspection in tests

type StoringRPCSender struct {
	messages map[string][]byte
}

func (s *StoringRPCSender) SendRPC(rpc string, address network.Address, kademliaMessage *proto_gen.KademliaMessage) error {
	s.messages[rpc] = kademliaMessage.Body
	return nil
}

func (s *StoringRPCSender) SendAndAwaitResponse(rpc string, address network.Address, kademliaMessage *proto_gen.KademliaMessage, timeout float32) (*proto_gen.KademliaMessage, error) {
	s.messages[rpc] = kademliaMessage.Body
	return &proto_gen.KademliaMessage{}, nil
}
