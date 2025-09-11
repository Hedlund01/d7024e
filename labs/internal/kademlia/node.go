package kademlia

import (
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/pkg/network"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg *network.Message, node IKademliaNode) error

// IKademliaNode defines the interface that all Kademlia node types must implement
type IKademliaNode interface {
	// Core networking methods
	send(to network.Address, msgType string, data []byte, messageID *kademliaID.KademliaID) error
	SendPingMessage(to network.Address) error
	SendPongMessage(to network.Address, messageID *kademliaID.KademliaID) error

	Address() network.Address

	// Message handling
	Handle(msgType string, handler MessageHandler)
	Start()
	Close() error

	GetRoutingTable() *RoutingTable
	GetKademlia() *Kademlia
}

// BaseNode provides common functionality that can be embedded
type KademliaNode struct {
	address        network.Address
	network        network.Network
	connection     network.Connection
	handlers       map[string]MessageHandler
	closed         bool
	closeMu        sync.RWMutex
	routingTable   *RoutingTable
	kademlia       *Kademlia
	messageChanMap map[kademliaID.KademliaID]chan kademliaID.KademliaID
}

type KademliaNodeData struct {
	RoutingTable *RoutingTable
	Kademlia     *Kademlia
}

// NewKademliaNode creates a new Kademlia node
func NewKademliaNode(network network.Network, addr network.Address) (*KademliaNode, error) {
	conn, err := network.Listen(addr)
	if err != nil {
		return nil, err
	}

	contact := NewContact(kademliaID.NewRandomKademliaID(), addr.String())
	routingTable := NewRoutingTable(contact)

	return &KademliaNode{
		address:        addr,
		network:        network,
		connection:     conn,
		handlers:       make(map[string]MessageHandler),
		closed:         false,
		kademlia:       &Kademlia{},
		routingTable:   routingTable,
		messageChanMap: make(map[kademliaID.KademliaID]chan kademliaID.KademliaID),
	}, nil
}

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

func (kn *KademliaNode) getMessageChan(messageID *kademliaID.KademliaID) (chan kademliaID.KademliaID, bool) {
	if messageID == nil {
		return nil, false
	}
	chn, exists := kn.messageChanMap[*messageID]
	return chn, exists
}

func (kn *KademliaNode) addToMessageChanMap(messageID *kademliaID.KademliaID) {
	if messageID == nil {
		return
	}
	kn.messageChanMap[*messageID] = make(chan kademliaID.KademliaID, 1)
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
				// Check if this is a PONG message and if there's a message channel for it
				if msg.PayloadType == "PONG" {
					msgChan, chanExists := kn.getMessageChan(msg.MessageID)
					if chanExists && msg.MessageID != nil {
						// Send the message ID to the channel
						msgChan <- *msg.MessageID
						log.WithField("msgID", msg.MessageID.String()).WithField("func", "KademliaNode/Start").Debugf("PING message ID sent to channel")
						kn.GetRoutingTable().AddContact(NewContact(msg.FromID, msg.From.String()))
					} else {
						return
					}
				}

				go handler(&msg, kn)
			}
		}
	}()
}

// Address returns the node's address
func (b *KademliaNode) Address() network.Address {
	return b.address
}

// Handle registers a message handler for a specific message type
func (b *KademliaNode) Handle(msgType string, handler MessageHandler) {
	b.handlers[msgType] = handler
}

// Close shuts down the Kademlia node
func (b *KademliaNode) Close() error {
	b.closeMu.Lock()
	b.closed = true
	b.closeMu.Unlock()
	return b.connection.Close()
}

// GetConnection returns the connection for use by concrete implementations
func (b *KademliaNode) GetConnection() network.Connection {
	return b.connection
}

// GetHandlers returns the handlers map for use by concrete implementations
func (b *KademliaNode) GetHandlers() map[string]MessageHandler {
	return b.handlers
}

// IsClosed returns whether the node is closed
func (b *KademliaNode) IsClosed() bool {
	b.closeMu.RLock()
	defer b.closeMu.RUnlock()
	return b.closed
}

func checkReply(b *KademliaNode, messageID *kademliaID.KademliaID) {
	if messageID == nil {
		return
	}
	msgChan, exists := b.getMessageChan(messageID)
	if !exists {
		return
	}

	select {
	case <-msgChan:
		log.WithField("msgID", messageID.String()).WithField("func", "checkReply").Debugf("Received reply for message ID")
	case <-time.After(30 * time.Second):
		log.WithField("msgID", messageID.String()).WithField("func", "checkReply").Debugf("Timeout waiting for reply for message ID")
	}

	delete(b.messageChanMap, *messageID)
}

// Send sends a message to the target address
func (b *KademliaNode) send(to network.Address, msgType string, data []byte, messageID *kademliaID.KademliaID) error {
	connection, err := b.network.Dial(to)
	if err != nil {
		return err
	}
	defer connection.Close()

	b.addToMessageChanMap(messageID)
	go checkReply(b, messageID)

	msg := &network.Message{
		From:        b.address,
		To:          to,
		Payload:     data,
		PayloadType: msgType,
		FromID:      b.routingTable.me.ID,
		MessageID:   messageID,
	}

	return connection.Send(msg)
}

func (kn *KademliaNode) SendPingMessage(to network.Address) error {

	return kn.send(to, "PING", []byte("ping"), kademliaID.NewRandomKademliaID())
}

func (kn *KademliaNode) SendPongMessage(to network.Address, messageID *kademliaID.KademliaID) error {
	return kn.send(to, "PONG", []byte("pong"), messageID)
}
