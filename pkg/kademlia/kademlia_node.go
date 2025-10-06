package kademlia

import (
	"bytes"
	"fmt"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
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
	Value        map[*utils.BitArray][]common.DataObject
}

func NewKademliaNode(net network.Network, addr network.Address, id utils.BitArray, k int, alpha int) (*KademliaNode, error) {
	// Update to account for k and alpha

	node, err := node.NewNode(net, addr)

	if err != nil {
		return nil, err
	}

	node.Start()

	return &KademliaNode{
		Node: node,
		ID:   id,
	}, nil
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

func (kn *KademliaNode) FindValueInNetwork(key *utils.BitArray) (string, error) {

	// Perform a lookup on the key

	nodes, err := kn.LookUp(key)

	if err != nil {
		return "", err
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
		return "", fmt.Errorf("value not found in network")
	}

	return finalValue, nil
}

func (kn *KademliaNode) Join(address network.Address) error {
	// Dummy implementation, always returns not implemented
	return fmt.Errorf("not implemented")
}

func (kn *KademliaNode) Store(value common.DataObject) (*utils.BitArray, error) {
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

func (kn *KademliaNode) Lookup(targetID *utils.BitArray) ([]*common.NodeInfo, error) {
	alpha := kn.alpha
	k := kn.k

	// Configurable timeouts
	const (
		nodeTimeout   = 2 * time.Second  // Per-node query timeout
		lookupTimeout = 30 * time.Second // Overall lookup timeout
	)

	// Create context with overall timeout
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	var closestNode *common.NodeInfo
	kClosestNodes := make([]*common.NodeInfo, 0, k)
	unprobedNodes := make([]*common.NodeInfo, 0)

	// Track probed nodes to avoid re-querying
	probed := make(map[string]bool)

	// Step 1: choose alpha nodes from the routing table
	initialNodes := kn.routingTable.FindClosest(*targetID)
	if len(initialNodes) > alpha {
		unprobedNodes = append(unprobedNodes, initialNodes[:alpha]...)
	} else {
		unprobedNodes = append(unprobedNodes, initialNodes...)
	}
	kClosestNodes = append(kClosestNodes, unprobedNodes...)

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
		probed[node.ID.String()] = true
		inflight++
		wg.Add(1)
		go func(n *common.NodeInfo) {
			defer wg.Done()

			// Create per-node timeout
			nodeCtx, nodeCancel := context.WithTimeout(ctx, nodeTimeout)
			defer nodeCancel()

			// Channel for the actual query
			type nodeResult struct {
				nodes []*common.NodeInfo
				err   error
			}
			nodeCh := make(chan nodeResult, 1)

			// Run query in goroutine
			go func() {
				res, err := n.FindNode(targetID)
				nodeCh <- nodeResult{nodes: res, err: err}
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
				continue // failed or timed out node
			}

			for _, n := range result.nodes {
				if n.ID.Equals(targetID) {
					cancel() // Cancel remaining operations
					wg.Wait()
					close(resultsCh)
					return []*common.NodeInfo{n}, nil
				}

				// Track closest node seen so far
				if closestNode == nil || n.ID.CloserTo(targetID, closestNode.ID) {
					closestNode = n
				}

				// Update kClosestNodes
				if !containsNodePtr(kClosestNodes, n) {
					if len(kClosestNodes) < k {
						kClosestNodes = append(kClosestNodes, n)
					} else {
						farthestIdx := findFarthestNodeIndexPtr(kClosestNodes, targetID)
						if n.ID.CloserTo(targetID, kClosestNodes[farthestIdx].ID) {
							kClosestNodes[farthestIdx] = n
						}
					}
				}

				// Schedule new queries if not probed
				if !probed[n.ID.String()] {
					unprobedNodes = append(unprobedNodes, n)
				}
			}

			// Launch more queries (keep α parallelism)
			for inflight < alpha && len(unprobedNodes) > 0 {
				next := unprobedNodes[0]
				unprobedNodes = unprobedNodes[1:]
				if !probed[next.ID.String()] {
					launchQuery(next)
				}
			}
		}
	}

	wg.Wait()
	close(resultsCh)

	return kClosestNodes, nil
}

/* pseudocode
function Lookup(targetID):
	closestNode = nil
	kClosestNodes = []
	unprobedNodes = []

	alpha = self.alpha
	k = self.k

	// choose alpha nodes from routing table
	unprobedNodes.add(self.routingTable.getClosestNodes(targetID, alpha))

	kClosestNodes.add(unprobedNodes)

	for node in unprobedNodes:
		result = node.FindNode(targetID)

		if result is not nil:
			unprobedNodes.add(result)
			mark node as probed

			for n in result:
				if n contains targetID:
					return n
				if closestNode is nil or n is closer to targetID than closestNode:
					closestNode = n
				if n not in kClosestNodes:
					if kClosestNodes.size < k:
						kClosestNodes.add(n)
					else if n is closer to targetID than the farthest node in kClosestNodes:
						replace farthest node in kClosestNodes with n

		else:
			remove node from unprobedNodes
*/
/*
func (kn *KademliaNode) Lookup(targetID *utils.BitArray) ([]*common.NodeInfo, error) {
	alpha := kn.alpha
	k := kn.k

	var closestNode *common.NodeInfo
	kClosestNodes := make([]*common.NodeInfo, 0, k)
	tempList := make([]*common.NodeInfo, 0)

	// Track probed nodes to avoid re-querying
	probed := make(map[string]bool)

	// Step 1: choose alpha nodes from the routing table
	initialNodes := kn.routingTable.FindClosest(*targetID)
	if len(initialNodes) > alpha {
		tempList = append(tempList, initialNodes[:alpha]...)
	} else {
		tempList = append(tempList, initialNodes...)
	}
	kClosestNodes = append(kClosestNodes, tempList...)

	// Step 2: iterative probing
	for len(tempList) > 0 {
		node := tempList[0]
		tempList = tempList[1:]

		nodeIDStr := node.ID.String()
		if probed[nodeIDStr] {
			continue // already probed
		}
		probed[nodeIDStr] = true

		// Perform a FindNode RPC
		result, err := node.FindNode(targetID)
		if err != nil || result == nil {
			continue // unreachable node
		}

		// Process returned nodes
		for _, n := range result {
			// If direct match, return immediately
			if n.ID.Equals(targetID) {
				return []*common.NodeInfo{n}, nil
			}

			// Track closest
			if closestNode == nil || n.ID.CloserTo(targetID, closestNode.ID) {
				closestNode = n
			}

			// Update kClosestNodes set
			if !containsNodePtr(kClosestNodes, n) {
				if len(kClosestNodes) < k {
					kClosestNodes = append(kClosestNodes, n)
				} else {
					farthestIdx := findFarthestNodeIndexPtr(kClosestNodes, targetID)
					if n.ID.CloserTo(targetID, kClosestNodes[farthestIdx].ID) {
						kClosestNodes[farthestIdx] = n
					}
				}
			}

			// Add to tempList if not probed
			if !probed[n.ID.String()] && !containsNodePtr(tempList, n) {
				tempList = append(tempList, n)
			}
		}
	}

	return kClosestNodes, nil
}


// Helpers for pointer slices
func containsNodePtr(nodes []*common.NodeInfo, candidate *common.NodeInfo) bool {
	for _, n := range nodes {
		if n.ID.Equals(candidate.ID) {
			return true
		}
	}
	return false
}

func findFarthestNodeIndexPtr(nodes []*common.NodeInfo, targetID *utils.BitArray) int {
	if len(nodes) == 0 {
		return -1
	}
	farthestIdx := 0
	for i := 1; i < len(nodes); i++ {
		if nodes[i].ID.Distance(targetID).Cmp(nodes[farthestIdx].ID.Distance(targetID)) > 0 {
			farthestIdx = i
		}
	}
	return farthestIdx
}



func (kn *KademliaNode) Lookup(targetID *utils.BitArray) ([]*common.NodeInfo, error) {
	alpha := kn.alpha
	k := kn.k

	var closestNode *common.NodeInfo
	kClosestNodes := make([]*common.NodeInfo, 0, k)
	tempList := kn.routingTable.FindClosest(*targetID)

	// Track probed nodes
	probed := make(map[string]bool)

	// Channel to receive async results
	type queryResult struct {
		from   *common.NodeInfo
		nodes  []*common.NodeInfo
		err    error
	}
	resultsCh := make(chan queryResult, 100)

	// Concurrency control
	var wg sync.WaitGroup
	inflight := 0

	// Launch up to alpha queries in parallel
	launchQuery := func(node *common.NodeInfo) {
		probed[node.ID.String()] = true
		inflight++
		wg.Add(1)
		go func(n *common.NodeInfo) {
			defer wg.Done()
			res, err := n.FindNode(targetID)
			resultsCh <- queryResult{from: n, nodes: res, err: err}
		}(node)
	}

	// Kick off initial α queries
	for i := 0; i < alpha && i < len(tempList); i++ {
		launchQuery(tempList[i])
	}
	tempList = tempList[min(alpha, len(tempList)):] // remove those launched

	// Main loop
	for inflight > 0 {
		result := <-resultsCh
		inflight--

		if result.err != nil || result.nodes == nil {
			continue // failed node
		}

		for _, n := range result.nodes {
			if n.ID.Equals(targetID) {
				close(resultsCh)
				wg.Wait()
				return []*common.NodeInfo{n}, nil
			}

			// Track closest node seen so far
			if closestNode == nil || n.ID.CloserTo(targetID, closestNode.ID) {
				closestNode = n
			}

			// Update kClosestNodes
			if !containsNodePtr(kClosestNodes, n) {
				if len(kClosestNodes) < k {
					kClosestNodes = append(kClosestNodes, n)
				} else {
					farthestIdx := findFarthestNodeIndexPtr(kClosestNodes, targetID)
					if n.ID.CloserTo(targetID, kClosestNodes[farthestIdx].ID) {
						kClosestNodes[farthestIdx] = n
					}
				}
			}

			// Schedule new queries if not probed
			if !probed[n.ID.String()] {
				tempList = append(tempList, n)
			}
		}

		// Launch more queries (keep α parallelism)
		for inflight < alpha && len(tempList) > 0 {
			next := tempList[0]
			tempList = tempList[1:]
			if !probed[next.ID.String()] {
				launchQuery(next)
			}
		}
	}

	wg.Wait()
	close(resultsCh)

	return kClosestNodes, nil
}
*/
