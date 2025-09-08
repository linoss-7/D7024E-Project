package gossip

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ncyborgse/go-template/pkg/network"
	"github.com/ncyborgse/go-template/pkg/node"
)

// GossipMessage represents a piece of information spreading through the network
type GossipMessage struct {
	ID        string    `json:"id"`        // unique message identifier
	Content   string    `json:"content"`   // the actual information
	Sender    int       `json:"sender"`    // original sender node id
	Timestamp time.Time `json:"timestamp"` // when message was created
	TTL       int       `json:"ttl"`       // time-to-live (hops remaining)
}

// GossipNode represents a node in the gossip network
type GossipNode struct {
	id           int
	addr         network.Address
	peers        []network.Address // known peer addresses
	node         *node.Node
	seenMessages map[string]bool // prevent message loops
	receivedMsgs []GossipMessage // messages this node has received
	mu           sync.RWMutex

	// visualization tracking
	builder *NetworkBuilder // reference to builder for trace logging

	// statistics
	messagesSent     int
	messagesReceived int
}

// NewGossipNode creates a new gossip node
func NewGossipNode(net network.Network, id int, port int, builder *NetworkBuilder) (*GossipNode, error) {
	addr := network.Address{IP: "127.0.0.1", Port: port}
	node, err := node.NewNode(net, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create gossip node %d: %v", id, err)
	}

	gossipnode := &GossipNode{
		id:           id,
		addr:         addr,
		peers:        make([]network.Address, 0),
		node:         node,
		seenMessages: make(map[string]bool),
		receivedMsgs: make([]GossipMessage, 0),
		builder:      builder,
	}

	// set up message handlers
	gossipnode.SetupHandlers()

	return gossipnode, nil
}

func (gn *GossipNode) SetupHandlers() {
	// handle gossip messages
	gn.node.Handle("gossip", func(msg network.Message) error {
		var gossipmsg GossipMessage
		if err := json.Unmarshal(msg.Payload[7:], &gossipmsg); err != nil { // skip "gossip:" prefix
			return fmt.Errorf("failed to unmarshal gossip message: %v", err)
		}

		// Extract the immediate sender's node ID from the port
		immediateForwarder := msg.From.Port - 8000
		return gn.HandleGossipMessage(gossipmsg, immediateForwarder)
	})

	// handle peer discovery
	gn.node.Handle("discover", func(msg network.Message) error {
		// send back our peer list
		peerdata, _ := json.Marshal(gn.peers)
		return gn.node.Send(msg.From, "peers", peerdata)
	})
}

// AddPeer adds a peer to this node's peer list
func (gn *GossipNode) AddPeer(peeraddr network.Address) {
	gn.mu.Lock()
	defer gn.mu.Unlock()

	// don't add ourselves or duplicates
	if peeraddr.Port == gn.addr.Port {
		return
	}

	for _, existing := range gn.peers {
		if existing.Port == peeraddr.Port {
			return // already exists
		}
	}

	gn.peers = append(gn.peers, peeraddr)
}

// Start begins the node's operation
func (gn *GossipNode) Start() {
	gn.node.Start()
}

// Gossip initiates spreading of a new message
func (gn *GossipNode) Gossip(content string) error {
	// create unique message id
	msgid := gn.GenerateMessageID()

	gossipmsg := GossipMessage{
		ID:        msgid,
		Content:   content,
		Sender:    gn.id,
		Timestamp: time.Now(),
		TTL:       20, // maximum 20 hops
	}

	fmt.Printf("node %d starting gossip: '%s'\n", gn.id, content)

	return gn.SpreadGossip(gossipmsg)
}

func (gn *GossipNode) HandleGossipMessage(msg GossipMessage, immediateForwarder int) error {
	gn.mu.Lock()

	// check if we've seen this message before
	if gn.seenMessages[msg.ID] {
		gn.mu.Unlock()
		return nil // already processed
	}

	// mark as seen
	gn.seenMessages[msg.ID] = true
	gn.receivedMsgs = append(gn.receivedMsgs, msg)
	gn.messagesReceived++

	gn.mu.Unlock()

	// Log message trace for visualization
	if gn.builder != nil {
		trace := MessageTrace{
			Timestamp:          time.Now(),
			MessageID:          msg.ID,
			OriginalSender:     msg.Sender,
			ImmediateForwarder: immediateForwarder,
			Receiver:           gn.id,
			Content:            msg.Content,
			TTL:                msg.TTL,
			IsDirect:           msg.Sender == immediateForwarder,
		}
		gn.builder.traceMu.Lock()
		gn.builder.traces = append(gn.builder.traces, trace)
		gn.builder.traceMu.Unlock()
	}

	if msg.Sender == immediateForwarder {
		// Direct from original sender
		fmt.Printf("node %d received gossip from node %d: '%s'\n", gn.id, msg.Sender, msg.Content)
	} else {
		// Forwarded by intermediate node
		fmt.Printf("node %d received gossip from node %d (via node %d): '%s'\n", gn.id, msg.Sender, immediateForwarder, msg.Content)
	}

	// decrease ttl and forward if still valid
	if msg.TTL > 0 {
		msg.TTL--
		go gn.SpreadGossip(msg)
	}

	return nil
}

func (gn *GossipNode) SpreadGossip(msg GossipMessage) error {
	gn.mu.RLock()
	peers := make([]network.Address, len(gn.peers))
	copy(peers, gn.peers)
	gn.mu.RUnlock()

	// send to all peers
	for _, peeraddr := range peers {
		go func(addr network.Address) {
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("failed to marshal gossip message: %v", err)
				return
			}

			if err := gn.node.Send(addr, "gossip", data); err != nil {
				// peer might be down or partitioned - that's ok in gossip protocols
				return
			}

			gn.mu.Lock()
			gn.messagesSent++
			gn.mu.Unlock()
		}(peeraddr)
	}

	return nil
}

func (gn *GossipNode) GenerateMessageID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetID returns the node's ID
func (gn *GossipNode) GetID() int {
	return gn.id
}

// GetStats returns node statistics
func (gn *GossipNode) GetStats() (int, int, int, int) {
	gn.mu.RLock()
	defer gn.mu.RUnlock()

	return len(gn.peers), len(gn.receivedMsgs), gn.messagesSent, gn.messagesReceived
}

// GetReceivedMessages returns all messages this node has received
func (gn *GossipNode) GetReceivedMessages() []GossipMessage {
	gn.mu.RLock()
	defer gn.mu.RUnlock()

	messages := make([]GossipMessage, len(gn.receivedMsgs))
	copy(messages, gn.receivedMsgs)
	return messages
}

// Close shuts down the node
func (gn *GossipNode) Close() error {
	return gn.node.Close()
}
