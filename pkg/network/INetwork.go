package network

import (
	"fmt"
)

type Address struct {
	IP   string
	Port int // 1-65535
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

type Network interface {
	Listen(addr Address) (Connection, error)
	Dial(addr Address) (Connection, error)

	// Network partition simulation
	Partition(group1, group2 []Address)
	Heal()
}

type Connection interface {
	Send(msg Message) error
	Recv() (Message, error)
	Close() error
}

type Message struct {
	From    Address
	To      Address
	Payload []byte
	network Network // Reference to network for replies
}

func (m Message) ReplyString(prefix string, message string) error {
	payload := []byte(prefix + ":" + message)

	reply := Message{
		From:    m.To,
		To:      m.From,
		Payload: payload,
		network: m.network,
	}
	conn, err := m.network.Dial(m.From)
	if err != nil {
		return err
	}
	return conn.Send(reply)

}
