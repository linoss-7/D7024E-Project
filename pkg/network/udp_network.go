package network

import (
	"errors"
	"log"
	"net"
	"sync"

	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"google.golang.org/protobuf/proto"
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
	raddr, err := net.ResolveUDPAddr(("udp"), addr.String())
	if err != nil {
		return nil, errors.New("Error resolving address:" + err.Error())
	}
	soc, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, errors.New("Error creating UDP socket:" + err.Error())
	}

	// If there are any listeners, add the first one as a reply address

	if len(n.listeners) > 0 {
		var replyAddr Address
		for k := range n.listeners {
			replyAddr = k
			break
		}
		return &udpConnection{replyAddr: replyAddr, addr: addr, network: n, soc: soc}, nil
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
	replyAddr Address
	addr      Address
	network   *udpNetwork
	recvCh    chan Message
	mu        sync.RWMutex
	closed    bool
	soc       *net.UDPConn
}

func (c *udpConnection) Send(msg Message) error {
	c.network.mu.RLock()
	defer c.network.mu.RUnlock()

	// Add network reference to message for replies
	msg.Network = c.network

	if c.network.partitions[c.addr] || c.network.partitions[msg.To] {
		return errors.New("network partitioned")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", msg.To.String())

	if err != nil {
		return errors.New("Invalid address:" + err.Error())
	}

	udpMessage := proto_gen.UDPMessage{
		FromIP:   c.replyAddr.IP,
		FromPort: int32(c.replyAddr.Port),
		ToIP:     msg.To.IP,
		ToPort:   int32(msg.To.Port),
		Payload:  msg.Payload,
	}

	payload, err := proto.Marshal(&udpMessage)

	if err != nil {
		return errors.New("Error marshalling UDP message:" + err.Error())
	}

	if c.soc.RemoteAddr() != nil {
		// Pre-connected socket (DialUDP)
		_, err = c.soc.Write(payload)
	} else {
		// Not connected (ListenUDP)
		_, err = c.soc.WriteToUDP(payload, udpAddr)
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

		n, _, err := c.soc.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP socket: %v", err)
			continue
		}
		var UDPMsg proto_gen.UDPMessage
		if err := proto.Unmarshal(buf[:n], &UDPMsg); err != nil {
			log.Printf("Error unmarshalling UDP message: %v", err)
			continue
		}

		var msg = Message{
			From: Address{
				IP:   UDPMsg.FromIP,
				Port: int(UDPMsg.FromPort),
			},
			To: Address{
				IP:   UDPMsg.ToIP,
				Port: int(UDPMsg.ToPort),
			},
			Payload: make([]byte, len(UDPMsg.Payload)),
		}

		copy(msg.Payload, UDPMsg.Payload)

		c.mu.Lock()
		closed := c.closed
		ch := c.recvCh
		c.mu.Unlock()
		if closed {
			// Close channel and return
			if ch != nil {
				close(ch)
			}
			return
		}

		if ch != nil {
			select {
			case ch <- msg:
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

	// Check if network is partitioned
	c.network.mu.RLock()
	if c.network.partitions[c.addr] {
		c.network.mu.RUnlock()
		return Message{}, errors.New("network partitioned")
	}
	c.network.mu.RUnlock()

	msg, ok := <-ch
	if !ok {
		return Message{}, errors.New("Connection closed")
	}
	return msg, nil
}

func (c *udpConnection) Close() error {

	// Just set closed parameter and let listen() exit gracefully

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil // Already closed
	}
	c.closed = true
	c.mu.Unlock()

	return nil
}
