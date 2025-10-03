package network

import (
	"errors"
	"math/rand/v2"
	"sync"
)

type mockNetwork struct {
	mu         sync.RWMutex
	listeners  map[Address]chan Message
	partitions map[Address]bool // true if the address is partitioned
	dropRate   float64          // probability of dropping a message

}

func NewMockNetwork(dropRate float64) Network {
	return &mockNetwork{
		listeners:  make(map[Address]chan Message),
		partitions: make(map[Address]bool),
		dropRate:   dropRate,
	}
}

func (n *mockNetwork) Listen(addr Address) (Connection, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.listeners[addr]; exists {
		return nil, errors.New("address already in use")
	}
	ch := make(chan Message, 100) // buffered channel
	n.listeners[addr] = ch
	return &mockConnection{addr: addr, network: n, recvCh: ch}, nil
}

func (n *mockNetwork) Dial(addr Address) (Connection, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if _, exists := n.listeners[addr]; !exists {
		return nil, errors.New("address not found")
	}
	return &mockConnection{addr: addr, network: n}, nil
}

func (n *mockNetwork) Partition(group1, group2 []Address) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, addr := range group1 {
		n.partitions[addr] = true
	}
	for _, addr := range group2 {
		n.partitions[addr] = true
	}
}

func (n *mockNetwork) Heal() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.partitions = make(map[Address]bool)
}

func (n *mockNetwork) SetDropRate(rate float64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.dropRate = rate
}

type mockConnection struct {
	addr    Address
	network *mockNetwork
	recvCh  chan Message
	mu      sync.RWMutex
	closed  bool
}

func (c *mockConnection) Send(msg Message) error {
	c.network.mu.RLock()

	if c.network.partitions[c.addr] || c.network.partitions[msg.To] {
		c.network.mu.RUnlock()
		return errors.New("network partitioned")
	}

	ch, exists := c.network.listeners[msg.To]
	if !exists {
		c.network.mu.RUnlock()
		return errors.New("destination address not found")
	}

	// Simulate message drop
	if c.network.dropRate > 0.0 {
		// Simple random drop based on dropRate
		if randFloat := rand.Float64(); randFloat < c.network.dropRate {
			c.network.mu.RUnlock()
			// Pretend the message was sent successfully
			return nil
		}
	}

	// Add network reference to message for replies
	msg.Network = c.network

	// Keep the lock while sending to prevent the channel from being closed
	select {
	case ch <- msg:
		c.network.mu.RUnlock()
		return nil
	default:
		c.network.mu.RUnlock()
		return errors.New("message queue full")
	}
}

func (c *mockConnection) Recv() (Message, error) {
	c.mu.RLock()
	if c.closed || c.recvCh == nil {
		c.mu.RUnlock()
		return Message{}, errors.New("connection not listening")
	}
	ch := c.recvCh
	c.mu.RUnlock()

	msg, ok := <-ch
	if !ok {
		return Message{}, errors.New("connection closed")
	}
	return msg, nil
}

func (c *mockConnection) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil // Already closed
	}
	c.closed = true
	c.mu.Unlock()

	c.network.mu.Lock()
	defer c.network.mu.Unlock()

	if c.recvCh != nil {
		close(c.recvCh)
		delete(c.network.listeners, c.addr)
		c.recvCh = nil
	}
	return nil
}
