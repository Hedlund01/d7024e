package kademlia

import (
	"d7024e/pkg/network"
	"d7024e/pkg/node"
	"fmt"
	log "github.com/sirupsen/logrus"
)

// KademliaNode extends the base node with Kademlia-specific functionality
type KademliaNode struct {
	*node.BaseNode
	routingTable *RoutingTable
	kademlia     *Kademlia
}

// KademliaNodeData contains Kademlia-specific data that handlers might need
type KademliaNodeData struct {
	RoutingTable *RoutingTable
	Kademlia     *Kademlia
}

// NewKademliaNode creates a new Kademlia node
func NewKademliaNode(network network.Network, addr network.Address) (*KademliaNode, error) {
	baseNode, err := node.NewBaseNode(network, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create base node: %v", err)
	}

	// Create Kademlia-specific components
	contact := NewContact(NewRandomKademliaID(), addr.String())
	routingTable := NewRoutingTable(contact)
	kademliaInstance := &Kademlia{}

	kademliaNode := &KademliaNode{
		BaseNode:     baseNode,
		routingTable: routingTable,
		kademlia:     kademliaInstance,
	}

	return kademliaNode, nil
}

// GetNodeData returns Kademlia-specific data
func (kn *KademliaNode) GetNodeData() interface{} {
	return &KademliaNodeData{
		RoutingTable: kn.routingTable,
		Kademlia:     kn.kademlia,
	}
}

// GetRoutingTable provides direct access to the routing table
func (kn *KademliaNode) GetRoutingTable() *RoutingTable {
	return kn.routingTable
}

// GetKademlia provides direct access to the Kademlia instance
func (kn *KademliaNode) GetKademlia() *Kademlia {
	return kn.kademlia
}

// Start begins listening for incoming messages with Kademlia-specific handling
func (kn *KademliaNode) Start() {
	go func() {
		for {
			if kn.IsClosed() {
				return
			}

			msg, err := kn.GetConnection().Recv()
			log.WithField("func", "KademliaNode/Start").Debugf("Received message: %v", msg)
			if err != nil {
				if !kn.IsClosed() {
					log.Printf("KademliaNode %s failed to receive message: %v", kn.Address().String(), err)
				}
				return
			}

			handlers := kn.GetHandlers()
			handler, exists := handlers[msg.PayloadType]
			if !exists {
				log.WithField("msgType", msg.PayloadType).WithField("func", "KademliaNode/Start").Debugf("No handler found, using default")
				handler, exists = handlers["DEFAULT"]
			}

			if exists && handler != nil {
				if err := handler(msg, kn); err != nil {
					log.Printf("Handler error: %v", err)
				}
			}
		}
	}()
}

// AddContact adds a contact to the routing table
func (kn *KademliaNode) AddContact(contact Contact) {
	kn.routingTable.AddContact(contact)
}

// FindClosestContacts finds the k closest contacts to a target
func (kn *KademliaNode) FindClosestContacts(target *KademliaID, count int) []Contact {
	return kn.routingTable.FindClosestContacts(target, count)
}