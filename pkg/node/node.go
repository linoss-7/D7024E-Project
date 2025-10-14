package node

import (
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid" // go get github.com/google/uuid

	"github.com/linoss-7/D7024E-Project/pkg/network"
)

// Node provides a unified abstraction for both sending and receiving messages
type Node struct {
	addr          network.Address
	network       network.Network
	connection    network.Connection
	handlers      map[string][]HandlerEntry
	firstHandlers map[string][]HandlerEntry
	mu            sync.RWMutex
	closed        bool
	closeMu       sync.RWMutex
}

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg network.Message) error

type HandlerEntry struct {
	ID      string
	Handler MessageHandler
}

// NewNode creates a new node that can both send and receive messages
func NewNode(network network.Network, addr network.Address) (*Node, error) {
	connection, err := network.Listen(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %v", err)
	}

	return &Node{
		addr:          addr,
		network:       network,
		connection:    connection,
		handlers:      make(map[string][]HandlerEntry),
		firstHandlers: make(map[string][]HandlerEntry),
	}, nil
}

// Handle registers a message handler for a specific message type
func (n *Node) Handle(msgType string, handler MessageHandler) string {
	n.mu.Lock()
	defer n.mu.Unlock()
	id := uuid.New().String()
	n.handlers[msgType] = append(n.handlers[msgType], HandlerEntry{
		ID:      id,
		Handler: handler,
	})
	return id
}

func (n *Node) RemoveHandler(msgType string, id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	handlers, exists := n.handlers[msgType]
	if !exists {
		return
	}
	for i, h := range handlers {
		if h.ID == id {
			n.handlers[msgType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

func (n *Node) HandleFirst(msgType string, handler MessageHandler) string {
	n.mu.Lock()
	defer n.mu.Unlock()
	id := uuid.New().String()
	n.firstHandlers[msgType] = append(n.firstHandlers[msgType], HandlerEntry{
		ID:      id,
		Handler: handler,
	})
	return id
}

// Start begins listening for incoming messages
func (n *Node) Start() {
	go func() {
		for {
			n.closeMu.RLock()
			if n.closed {
				n.closeMu.RUnlock()
				log.Print("Node ", n.addr.String(), " is closed, stopping listener")
				return
			}
			n.closeMu.RUnlock()

			msg, err := n.connection.Recv()
			if err != nil {
				n.closeMu.RLock()
				if !n.closed {
					log.Printf("Node %s failed to receive message: %v", n.addr.String(), err)
				}
				n.closeMu.RUnlock()
				return
			}

			// Extract message type from payload (first part before ':')
			msgType := "default"
			payload := string(msg.Payload)
			if len(payload) > 0 {
				for i, char := range payload {
					if char == ':' {
						msgType = payload[:i]
						break
					}
				}
			}

			// Copy the handlers slice for this msgType while holding the lock
			// to avoid races if another goroutine removes or modifies handlers.
			n.mu.RLock()
			origHandlers, exists := n.handlers[msgType]
			handlersCopy := make([]HandlerEntry, len(origHandlers))
			copy(handlersCopy, origHandlers)

			// Copy first-handlers and default handlers under the same lock so we
			// capture a consistent snapshot for this message.
			origFirstHandlers := n.firstHandlers[msgType]
			firstHandlersCopy := make([]HandlerEntry, len(origFirstHandlers))
			copy(firstHandlersCopy, origFirstHandlers)

			origDefaultHandlers := n.handlers["default"]
			defaultHandlersCopy := make([]HandlerEntry, len(origDefaultHandlers))
			copy(defaultHandlersCopy, origDefaultHandlers)
			n.mu.RUnlock()

			// Dispatch: start a single goroutine per message that runs first-handlers
			// and waits for them to finish, then starts the other handlers. This
			// keeps the listener loop non-blocking while preserving the ordering
			// guarantee (first-handlers run-before other handlers).
			go func(m network.Message, fh []HandlerEntry, dh []HandlerEntry, oth []HandlerEntry, hasSpecific bool) {
				// Run first-handlers concurrently then wait for completion.
				if len(fh) > 0 {
					var wg sync.WaitGroup
					for _, h := range fh {
						wg.Add(1)
						go func(h HandlerEntry) {
							defer wg.Done()
							// Run handler; ignore returned error here. Handlers
							// should handle their own errors/logging.
							_ = h.Handler(m)
						}(h)
					}
					wg.Wait()
				}

				// After first-handlers have completed, launch default handlers
				// and any specific handlers without waiting for them. They are
				// free to run concurrently and won't block the listener.
				for _, h := range dh {
					go h.Handler(m)
				}
				if hasSpecific {
					for _, h := range oth {
						go h.Handler(m)
					}
				}
			}(msg, firstHandlersCopy, defaultHandlersCopy, handlersCopy, exists)
		}
	}()
}

// Send sends a message to the target address
func (n *Node) Send(to network.Address, msgType string, data []byte) error {
	connection, err := n.network.Dial(to)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", to.String(), err)
	}
	defer connection.Close()

	// Format payload as "msgType:data"
	var payload []byte
	if msgType != "" {
		payload = append([]byte(msgType+":"), data...)
	} else {
		payload = data
	}

	msg := network.Message{
		From:    n.addr,
		To:      to,
		Payload: payload,
	}

	//logrus.Infof("Node %s sending message of type '%s' to %s", n.addr.String(), msgType, to.String())
	return connection.Send(msg)
}

// SendString is a convenience method for sending string messages
func (n *Node) SendString(to network.Address, msgType, data string) error {
	return n.Send(to, msgType, []byte(data))
}

// Close shuts down the node
func (n *Node) Close() error {
	n.closeMu.Lock()
	n.closed = true
	n.closeMu.Unlock()
	return n.connection.Close()
}

// Address returns the node's address
func (n *Node) Address() network.Address {
	return n.addr
}
