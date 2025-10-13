package network

import (
	kademliaID "d7024e/internal/kademlia/id"
	"fmt"
)

type Address struct {
	IP   string
	Port int // 1-65535
}

// String returns the string representation of the Address.
func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

type Network interface {
	Listen(addr Address) (Connection, error)
	Dial(addr Address) (Connection, error)

	// Network partition simulation
	Partition(group1, group2 []Address)
	Heal()
	EnableDropRate(enable bool)
}

type Connection interface {
	Send(msg *Message) error
	Recv() (Message, error)
	Close() error
}

type Message struct {
	From        Address
	FromID      *kademliaID.KademliaID
	MessageID   *kademliaID.KademliaID
	To          Address
	Payload     []byte
	PayloadType string
}
