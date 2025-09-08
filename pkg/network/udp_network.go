package network

import (
	"errors"
	"log"
	"net"
	"sync"
)

type udpNetwork struct {
	mu         sync.RWMutex
	listeners  map[Address]chan Message
	partitions map[Address]bool // true if the address is partitioned
}

func NewUDPNetwork() Network {
	return &udpNetwork{
		listeners:  make(map[Address]chan Message),
		partitions: make(map[Address]bool),
	}
}

func (n *udpNetwork) Listen(addr Address) (Connection, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.listeners[addr]; exists {
		return nil, errors.New("address already in use")
	}

	// Create a UDP socket
	address := &net.UDPAddr{IP: net.ParseIP(addr.IP), Port: addr.Port}
	soc, err := net.ListenUDP("udp", address)
	if err != nil {
		return nil, errors.New("Error creating UDP socket:" + err.Error())
	}
	ch := make(chan Message, 100) // buffered channel
	n.listeners[addr] = ch
	conn := &udpConnection{addr: addr, network: n, soc: soc, recvCh: ch}
	go conn.listen()
	return conn, nil
}

func (n *udpNetwork) Dial(addr Address) (Connection, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	/*
		if _, exists := n.listeners[addr]; !exists {
			return nil, errors.New("address not found")
		}
	*/
	address := &net.UDPAddr{IP: net.ParseIP(addr.IP), Port: addr.Port}
	soc, err := net.DialUDP("udp", nil, address)
	if err != nil {
		return nil, errors.New("Error creating UDP socket:" + err.Error())
	}

	return &udpConnection{addr: addr, network: n, soc: soc}, nil
}

func (n *udpNetwork) Partition(group1, group2 []Address) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, addr := range group1 {
		n.partitions[addr] = true
	}
	for _, addr := range group2 {
		n.partitions[addr] = true
	}
}

func (n *udpNetwork) Heal() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.partitions = make(map[Address]bool)
}

type udpConnection struct {
	addr    Address
	network *udpNetwork
	recvCh  chan Message
	mu      sync.RWMutex
	closed  bool
	soc     *net.UDPConn
}

func (c *udpConnection) Send(msg Message) error {
	c.network.mu.RLock()
	defer c.network.mu.RUnlock()

	// Add network reference to message for replies
	msg.network = c.network

	if c.network.partitions[c.addr] || c.network.partitions[msg.To] {
		return errors.New("network partitioned")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", msg.To.String())

	if err != nil {
		return errors.New("Invalid address:" + err.Error())
	}

	if c.soc.RemoteAddr() != nil {
		// Pre-connected socket (DialUDP)
		_, err = c.soc.Write(msg.Payload)
	} else {
		// Not connected (ListenUDP)
		_, err = c.soc.WriteToUDP(msg.Payload, udpAddr)
	}

	//return errors.New("Sent to udp:" + udpAddr.String())

	if err != nil {
		return errors.New("Error sending:" + err.Error())
	}

	return nil
}

func (c *udpConnection) listen() {
	buf := make([]byte, 1024)
	for {

		if c.closed {
			return
		}
		log.Printf("Listening for messages at %s\n", c.addr.String())
		n, addr, err := c.soc.ReadFromUDP(buf)
		log.Printf("Read %d bytes from %s\n", n, addr.String())
		if err != nil {
			log.Printf("Error reading from UDP socket: %v", err)
			continue
		}
		var msg Message
		msg.From = Address{IP: addr.IP.String(), Port: addr.Port}
		msg.To = c.addr
		msg.Payload = make([]byte, n)
		copy(msg.Payload, buf[:n])
		msg.network = c.network

		c.mu.RLock()
		ch := c.recvCh
		c.mu.RUnlock()

		if ch != nil {
			select {
			case ch <- msg:
				log.Printf("Message received at %s from %s: %s", c.addr.String(), msg.From.String(), string(msg.Payload))
			default:
				// Drop message to prevent stalling
			}
		}

	}
}

func (c *udpConnection) Recv() (Message, error) {
	c.mu.RLock()
	if c.closed || c.recvCh == nil {
		c.mu.RUnlock()
		return Message{}, errors.New("Connection not listening")
	}
	ch := c.recvCh
	c.mu.RUnlock()

	msg, ok := <-ch
	if !ok {
		return Message{}, errors.New("Connection closed")
	}
	return msg, nil
}

func (c *udpConnection) Close() error {
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
