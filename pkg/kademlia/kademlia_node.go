package kademlia

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/rpc_handlers"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/node"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"google.golang.org/protobuf/proto"
)

type KademliaNode struct {
	Node         *node.Node
	ID           utils.BitArray
	RoutingTable *common.RoutingTable
	Values       map[*utils.BitArray][]common.DataObject
	republishers map[*utils.BitArray]chan bool
	repubMutex   sync.RWMutex
	k            int
	alpha        int
}

func NewKademliaNode(net network.Network, addr network.Address, id utils.BitArray, k int, alpha int) (*KademliaNode, error) {
	// Update to account for k and alpha

	node, err := node.NewNode(net, addr)

	if err != nil {
		return nil, err
	}

	// Create kademlia node

	kn, err := &KademliaNode{
		Node: node,
		ID:   id,
	}, nil

	// Register handlers for the node

	knInfo := common.NodeInfo{
		ID:   id,
		IP:   addr.IP,
		Port: addr.Port,
	}

	rt := common.NewRoutingTable(kn, knInfo, k)

	kn.RoutingTable = rt
	kn.Values = make(map[*utils.BitArray][]common.DataObject)
	kn.k = k
	kn.alpha = alpha

	pingHandler := rpc_handlers.NewPingHandler(kn, knInfo)
	exitHandler := rpc_handlers.NewExitHandler(kn, kn, &knInfo.ID)
	//storeHandler := rpc_handlers.NewStoreHandler(kn, kn, &knInfo.ID)
	getHandler := rpc_handlers.NewGetHandler(kn, kn, &knInfo.ID)
	findNodeHandler := rpc_handlers.NewFindNodeHandler(kn, rt)
	putHandler := rpc_handlers.NewPutHandler(kn, kn, &knInfo.ID)

	kn.Node.Handle("ping", pingHandler.Handle)
	kn.Node.Handle("exit", exitHandler.Handle)
	//kn.Node.Handle("store", storeHandler.Handle)
	kn.Node.Handle("get", getHandler.Handle)
	kn.Node.Handle("find_node", findNodeHandler.Handle)
	kn.Node.Handle("put", putHandler.Handle)

	node.Start()

	return kn, nil
}

func (kn *KademliaNode) SendAndAwaitResponse(rpc string, address network.Address, kademliaMessage *proto_gen.KademliaMessage) (*proto_gen.KademliaMessage, error) {
	// Send a message and block until a message with the same RPCId is received
	responseCh := make(chan *proto_gen.KademliaMessage)
	errCh := make(chan error)

	// Create a new message handler for the response
	handlerFunc := func(msg network.Message) error {
		var respMsg proto_gen.KademliaMessage
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

func (kn *KademliaNode) FindValue(key *utils.BitArray) (string, error) {
	// Not implemented
	return "", fmt.Errorf("not implemented")
}

func (kn *KademliaNode) FindValueInNetwork(key *utils.BitArray) (string, []common.NodeInfo, error) {

	// Perform a lookup on the key

	nodes, err := kn.LookUp(key)

	if err != nil {
		return "", nil, err
	}

	// Send find_value RPCs to all those nodes

	resCh := make(chan string, len(nodes))
	for i := 0; i < len(nodes); i++ {
		go func(n common.NodeInfo) {
			// Create find_value message
			findValueMsg := common.DefaultKademliaMessage(kn.ID, key.ToBytes())
			resp, err := kn.SendAndAwaitResponse("find_value", network.Address{IP: n.IP, Port: n.Port}, findValueMsg)
			if err != nil {
				//logrus.Errorf("Error sending find_value to %s: %v", n.ID.ToString(), err)
				resCh <- ""
				return
			}

			// Convert byte body to string
			value := string(resp.Body)
			resCh <- value
		}(nodes[i])
	}

	// Collect results
	var results []string
	for i := 0; i < len(nodes); i++ {
		result := <-resCh
		results = append(results, result)
	}

	// Return the non-empty result with the highest frequency
	frequency := make(map[string]int)
	for _, v := range results {
		if v != "" {
			frequency[v]++
		}
	}

	var finalValue string
	maxFreq := 0
	for k, v := range frequency {
		if v > maxFreq {
			maxFreq = v
			finalValue = k
		}
	}

	if finalValue == "" {
		return "", nil, fmt.Errorf("value not found in network")
	}

	// Return the nodes that had the value
	var storingNodes []common.NodeInfo
	for i, v := range results {
		if v == finalValue {
			storingNodes = append(storingNodes, nodes[i])
		}
	}

	return finalValue, storingNodes, nil
}

func (kn *KademliaNode) Join(address network.Address) error {
	// Dummy implementation, always returns not implemented
	return fmt.Errorf("not implemented")
}

func (kn *KademliaNode) Store(value common.DataObject) (*utils.BitArray, error) {
	// Store the value locally

	// Check if the key already exists, if so restart the republish timer
	key := utils.ComputeHash(value.Data, 160)
	storedValue, err := kn.FindValue(key)

	if err != nil {
		return nil, err
	}

	if storedValue != "" {
		kn.repubMutex.RLock()
		kn.republishers[key] <- true
		kn.repubMutex.RUnlock()
		return key, nil
	}

	// Otherwise, add the value to local storage and start a republisher for it

	// Add value to local storage
	kn.StartRepublish(key, utils.NewRealTimeTicker(300*time.Second))

	// Add value to local storage

	// Dummy implementation, always returns not implemented
	return nil, fmt.Errorf("not implemented")
}

func (kn *KademliaNode) StoreInNetwork(value string) (*utils.BitArray, error) {
	// Hash the value to get the key

	key := utils.ComputeHash(value, 160)

	// Perform a lookup on the key

	nodes, err := kn.LookUp(key)

	if err != nil {
		return nil, err
	}

	// Send store RPCs to all those nodes

	for i := 0; i < len(nodes); i++ {
		go func(n common.NodeInfo) {
			// Create store message
			storeMsg := common.DefaultKademliaMessage(kn.ID, key.ToBytes())
			storeMsg.Body = []byte(value)
			kn.SendRPC("store", network.Address{IP: n.IP, Port: n.Port}, storeMsg)
		}(nodes[i])
	}

	return key, nil
}

func (kn *KademliaNode) Refresh(key *utils.BitArray, value string) error {

	// Check the key in the routing table

	nodes := kn.RoutingTable.FindClosest(*key)

	// Send store RPCs to all those nodes

	for i := 0; i < len(nodes); i++ {
		go func(n common.NodeInfo) {
			// Create store message
			storeMsg := common.DefaultKademliaMessage(kn.ID, key.ToBytes())
			storeMsg.Body = []byte(value)
			kn.SendRPC("store", network.Address{IP: n.IP, Port: n.Port}, storeMsg)
		}(*nodes[i])
	}

	return nil
}

func (kn *KademliaNode) SendRPC(rpc string, addr network.Address, kademliaMessage *proto_gen.KademliaMessage) error {
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

func (kn *KademliaNode) LookUp(id *utils.BitArray) ([]common.NodeInfo, error) {
	// Dummy implementation, does nothing
	return nil, fmt.Errorf("not implemented")
}

func (kn *KademliaNode) Exit() error {
	// Exit the node
	return kn.Node.Close()
}

// StartRepublish starts a goroutine that republishes the value for key periodically
// using the provided Ticker. restartCh triggers an immediate republish when a value is sent to it.
func (kn *KademliaNode) StartRepublish(key *utils.BitArray, ticker utils.Ticker) (chan bool, chan error) {
	restartCh := make(chan bool)
	errCh := make(chan error, 1)

	// Add the restart channel to the republishers map
	kn.repubMutex.Lock()
	if kn.republishers == nil {
		kn.republishers = make(map[*utils.BitArray]chan bool)
	}
	kn.republishers[key] = restartCh
	kn.repubMutex.Unlock()

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C():
				// Republish when ticker ticks
				val, err := kn.FindValue(key)
				if val == "" {
					// Value not found locally, exit republisher
					return
				}
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					continue
				}
				if err := kn.Refresh(key, val); err != nil {
					select {
					case errCh <- err:
					default:
					}
					continue
				}
			case <-restartCh:
				// Reset ticker, extending duration until next tick
				ticker.Reset()
			}
		}
	}()
	return restartCh, errCh
}
