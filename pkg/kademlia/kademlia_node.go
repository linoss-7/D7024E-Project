package kademlia

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/rpc_handlers"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/node"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type KademliaNode struct {
	Node         *node.Node
	ID           utils.BitArray
	RoutingTable *common.RoutingTable
	Values       map[*utils.BitArray][]common.DataObject
	valueMutex   sync.RWMutex
	republishers map[*utils.BitArray]chan bool
	repubMutex   sync.RWMutex
	refreshers   map[*utils.BitArray]chan bool
	refreshMutex sync.RWMutex
	k            int
	alpha        int
}

func NewKademliaNode(net network.Network, addr network.Address, id utils.BitArray, k int, alpha int) (*KademliaNode, error) {
	// Create node (not a Kademlia node)
	node, err := node.NewNode(net, addr)
	if err != nil {
		return nil, err
	}

	// Create Kademlia node
	kn, err := &KademliaNode{
		Node:  node,
		ID:    id,
		k:     k,
		alpha: alpha,
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
	// Check only local storage
	kn.valueMutex.RLock()
	defer kn.valueMutex.RUnlock()
	if values, exists := kn.Values[key]; exists && len(values) > 0 {
		// Ensure the value is not expired
		if values[0].ExpirationDate.After(time.Now()) {
			return values[0].Data, nil
		}
		// Otherwise, remove expired value
		delete(kn.Values, key)
	}
	return "", nil
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
		go func(n *common.NodeInfo) {
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
			storingNodes = append(storingNodes, *nodes[i])
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
	key := utils.ComputeHash(value.Data, 160)

	// Check if the key already exists, if so restart the republish timer

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
	kn.valueMutex.Lock()
	kn.Values[key] = append(kn.Values[key], value)
	kn.valueMutex.Unlock()

	// Start republisher

	kn.StartRepublish(key, utils.NewRealTimeTicker(300*time.Second))

	return key, nil
}

func (kn *KademliaNode) Forget(id *utils.BitArray) error {
	// Dummy implementation, always returns not implemented
	return fmt.Errorf("not implemented")
}

func (kn *KademliaNode) StoreInNetwork(value string) (*utils.BitArray, error) {
	// Hash the value to get the key

	key := utils.ComputeHash(value, 160)

	// Check that theres no already a refresher for the key
	kn.refreshMutex.RLock()
	if kn.refreshers[key] != nil {
		kn.refreshMutex.RUnlock()
		return nil, fmt.Errorf("a refresher already exists for key %s", key.ToString())
	}
	kn.refreshMutex.RUnlock()

	// Start a refresher for the key

	kn.StartRefresh(key, utils.NewRealTimeTicker(300*time.Second))

	// Perform a lookup on the key

	nodes, err := kn.LookUp(key)

	if err != nil {
		return nil, err
	}

	// Send store RPCs to all those nodes

	for i := 0; i < len(nodes); i++ {
		go func(n *common.NodeInfo) {
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

func (kn *KademliaNode) StartRefresh(key *utils.BitArray, ticker utils.Ticker) chan error {
	exitCh := make(chan bool)
	errCh := make(chan error, 1)

	// Add the exit channel to the refreshers map
	kn.refreshMutex.Lock()
	if kn.refreshers == nil {
		kn.refreshers = make(map[*utils.BitArray]chan bool)
	}
	kn.refreshers[key] = exitCh
	kn.refreshMutex.Unlock()

	go func() {
		defer ticker.Stop()
		// Similar to republisher but cant be restarted, only tick and exit
		for {
			select {
			case <-ticker.C():
				// Refresh when ticker ticks
				val, err := kn.FindValue(key)
				if val == "" {
					// Value not found locally, exit refresher
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
			case <-exitCh:
				return
			}
		}
	}()
	return errCh
}

func (kn *KademliaNode) StopRefresh(key *utils.BitArray) error {
	kn.refreshMutex.Lock()
	ch, exists := kn.refreshers[key]
	if !exists {
		// This is not an error, just means theres no refresher for the key
		return nil
	}
	// Remove the entry from the map
	delete(kn.refreshers, key)
	kn.refreshMutex.Unlock()

	close(ch)
	return nil
}

func (kn *KademliaNode) LookUp(targetID *utils.BitArray) ([]*common.NodeInfo, error) {
	// Configurable timeouts
	const (
		nodeTimeout   = 2 * time.Second  // Per-node query timeout
		lookupTimeout = 30 * time.Second // Overall lookup timeout
	)

	// Create context with overall timeout
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	// k and alpha from kn
	alpha := kn.alpha
	k := kn.k

	// Variables to track state
	var closestNode *common.NodeInfo
	kClosestNodes := make([]*common.NodeInfo, 0, k)
	unprobedNodes := make([]*common.NodeInfo, 0)

	// Track probed nodes to avoid re-querying
	probed := make(map[string]bool)

	// Choose alpha nodes from the routing table
	initialNodes := kn.RoutingTable.FindClosest(*targetID)
	if len(initialNodes) > alpha {
		unprobedNodes = append(unprobedNodes, initialNodes[:alpha]...)
	} else {
		unprobedNodes = append(unprobedNodes, initialNodes...)
	}

	// Add unprobed nodes to kClosestNodes initially (alpha < k)
	for _, n := range unprobedNodes {
		if !containsNode(kClosestNodes, n) {
			kClosestNodes = append(kClosestNodes, n)
		}
	}

	// Channel to receive async results
	type queryResult struct {
		from  *common.NodeInfo
		nodes []*common.NodeInfo
		err   error
	}
	resultsCh := make(chan queryResult, 100)

	// Concurrency control
	var wg sync.WaitGroup
	inflight := 0

	// Launch up to alpha queries in parallel with timeout
	launchQuery := func(node *common.NodeInfo) {
		probed[node.ID.ToString()] = true
		inflight++
		wg.Add(1)
		go func(n *common.NodeInfo) {
			defer wg.Done()

			// Create per-node timeout
			nodeCtx, nodeCancel := context.WithTimeout(ctx, nodeTimeout)
			defer nodeCancel()

			// Create nodeinfo message for the request
			nodeInfoMsg := &proto_gen.NodeInfoMessage{
				ID:   n.ID.ToBytes(),
				IP:   "",
				Port: 0,
			}

			// Marshal nodeinfo message
			data, err := proto.Marshal(nodeInfoMsg)
			if err != nil {
				resultsCh <- queryResult{from: n, nodes: nil, err: fmt.Errorf("failed to marshal NodeInfoMessage: %v", err)}
				return
			}

			// Send find_node RPC
			kMsg, err := kn.SendAndAwaitResponse("find_node", network.Address{IP: n.IP, Port: n.Port}, common.DefaultKademliaMessage(kn.ID, data))
			if err != nil {
				resultsCh <- queryResult{from: n, nodes: nil, err: err}
				return
			}

			// Extract body from KademliaMessage
			kMsgBody := kMsg.GetBody()
			if kMsgBody == nil {
				resultsCh <- queryResult{from: n, nodes: nil, err: fmt.Errorf("nil body in Kademlia message")}
				return
			}

			// Unmarshal body to get nodes
			var body proto_gen.NodeInfoMessageList
			if err := proto.Unmarshal(kMsgBody, &body); err != nil {
				resultsCh <- queryResult{from: n, nodes: nil, err: fmt.Errorf("failed to unmarshal FindNodeResponse: %v", err)}
				return
			}

			// Get nodes from NodeInfoMessageList
			nodes := make([]*common.NodeInfo, 0, len(body.Nodes))
			for _, nim := range body.Nodes {
				id := utils.NewBitArrayFromBytes(nim.ID, 160)
				node := &common.NodeInfo{
					ID:   *id,
					IP:   nim.GetIP(),
					Port: int(nim.GetPort()),
				}
				nodes = append(nodes, node)
			}

			// Channel to receive result or timeout
			nodeCh := make(chan queryResult, 1)

			// Send result to nodeCh
			go func() {
				nodeCh <- queryResult{from: n, nodes: nodes, err: nil}
			}()

			// Wait for result or timeout
			select {
			case nr := <-nodeCh:
				resultsCh <- queryResult{from: n, nodes: nr.nodes, err: nr.err}
			case <-nodeCtx.Done():
				// Node timed out
				resultsCh <- queryResult{from: n, nodes: nil, err: nodeCtx.Err()}
			}
		}(node)
	}

	// Kick off initial α queries
	for i := 0; i < alpha && i < len(unprobedNodes); i++ {
		launchQuery(unprobedNodes[i])
	}
	unprobedNodes = unprobedNodes[min(alpha, len(unprobedNodes)):] // remove those launched

	// Main loop with overall timeout check
	for inflight > 0 {
		select {
		case <-ctx.Done():
			// Overall timeout reached - return what we have
			wg.Wait()
			close(resultsCh)
			if len(kClosestNodes) == 0 {
				return nil, fmt.Errorf("lookup timeout: %w", ctx.Err())
			}
			return kClosestNodes, nil

		case result := <-resultsCh:
			inflight--

			if result.err != nil || result.nodes == nil {
				//logrus.Errorf("Error: %v", result.err)
				continue // failed or timed out node
			}

			for _, n := range result.nodes {
				// Track closest node seen so far
				if closestNode == nil || n.ID.CloserTo(*targetID, closestNode.ID) {
					closestNode = n
				}

				// Update kClosestNodes
				if !containsNode(kClosestNodes, n) {
					if len(kClosestNodes) < k {
						kClosestNodes = append(kClosestNodes, n)
					} else {
						farthestIdx := findFarthestNodeIndex(kClosestNodes, targetID)
						if n.ID.CloserTo(*targetID, kClosestNodes[farthestIdx].ID) {
							kClosestNodes[farthestIdx] = n
						}
					}
				}

				// Schedule new queries if not probed
				if !probed[n.ID.ToString()] {
					unprobedNodes = append(unprobedNodes, n)
				}
			}

			// Launch more queries (keep α parallelism)
			for inflight < alpha && len(unprobedNodes) > 0 {
				next := unprobedNodes[0]
				unprobedNodes = unprobedNodes[1:]
				if !probed[next.ID.ToString()] {
					launchQuery(next)
				}
			}
		}
	}

	wg.Wait()
	close(resultsCh)

	logrus.Infof("%d nodes were probed nodes", len(probed))

	return kClosestNodes, nil
}

// Helper functions

func containsNode(s []*common.NodeInfo, v *common.NodeInfo) bool {
	for _, x := range s {
		if x.ID.ToString() == v.ID.ToString() {
			return true
		}
	}
	return false
}

func findFarthestNodeIndex(nodes []*common.NodeInfo, targetID *utils.BitArray) int {
	farthestIdx := 0
	for i := 1; i < len(nodes); i++ {
		if !nodes[i].ID.CloserTo(*targetID, nodes[farthestIdx].ID) {
			farthestIdx = i
		}
	}
	return farthestIdx
}
