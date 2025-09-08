package gossip

import (
	"fmt"
	mathrand "math/rand"
	"sync"
	"time"

	"github.com/ncyborgse/go-template/pkg/network"
)

// NetworkBuilder helps create a network of gossip nodes with random topology
type NetworkBuilder struct {
	network   network.Network
	nodes     []*GossipNode
	traces    []MessageTrace
	startTime time.Time
	traceMu   sync.Mutex
}

func NewNetworkBuilder(net network.Network) *NetworkBuilder {
	return &NetworkBuilder{
		network:   net,
		nodes:     make([]*GossipNode, 0),
		traces:    make([]MessageTrace, 0),
		startTime: time.Now(),
	}
}

// CreateNodes creates the specified number of gossip nodes
func (nb *NetworkBuilder) CreateNodes(count int) error {
	fmt.Printf("creating %d gossip nodes...\n", count)

	for i := 0; i < count; i++ {
		node, err := NewGossipNode(nb.network, i, 8000+i, nb)
		if err != nil {
			return fmt.Errorf("failed to create node %d: %v", i, err)
		}
		nb.nodes = append(nb.nodes, node)
	}

	return nil
}

// BuildRandomTopology creates random connections between nodes
func (nb *NetworkBuilder) BuildRandomTopology(peerspernode int) {
	fmt.Printf("building random topology (%d peers per node)...\n", peerspernode)

	for _, node := range nb.nodes {
		// randomly select peers for this node
		selectedpeers := nb.SelectRandomPeers(node.id, peerspernode)

		for _, peerid := range selectedpeers {
			peeraddr := network.Address{IP: "127.0.0.1", Port: 8000 + peerid}
			node.AddPeer(peeraddr)
		}
	}
}

func (nb *NetworkBuilder) SelectRandomPeers(nodeid int, count int) []int {
	peers := make([]int, 0)
	maxattempts := count * 3 // prevent infinite loop

	for len(peers) < count && maxattempts > 0 {
		candidate := mathrand.Intn(len(nb.nodes))
		if candidate == nodeid {
			maxattempts--
			continue // don't add ourselves
		}

		// check if already selected
		found := false
		for _, existing := range peers {
			if existing == candidate {
				found = true
				break
			}
		}

		if !found {
			peers = append(peers, candidate)
		}
		maxattempts--
	}

	return peers
}

// StartAllNodes starts all nodes in the network
func (nb *NetworkBuilder) StartAllNodes() {
	fmt.Printf("starting %d nodes...\n", len(nb.nodes))

	for _, node := range nb.nodes {
		node.Start()
	}

	// give nodes time to start
	time.Sleep(100 * time.Millisecond)
}

// InitiateGossip starts gossip from a random node
func (nb *NetworkBuilder) InitiateGossip(content string) {
	if len(nb.nodes) == 0 {
		return
	}

	// pick a random node to start the gossip
	starter := mathrand.Intn(len(nb.nodes))
	nb.nodes[starter].Gossip(content)
}

// GetNodes returns all nodes in the network
func (nb *NetworkBuilder) GetNodes() []*GossipNode {
	return nb.nodes
}

// CloseAllNodes shuts down all nodes
func (nb *NetworkBuilder) CloseAllNodes() {
	for _, node := range nb.nodes {
		node.Close()
	}
}
