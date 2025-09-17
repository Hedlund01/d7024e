package kademlia

import (
	kademliaBucket "d7024e/internal/kademlia/bucket"
	kademliaContact "d7024e/internal/kademlia/contact"
	"d7024e/internal/kademlia/handlers/tempHandlers"
	tempHandlersPkg "d7024e/internal/kademlia/handlers/tempHandlers"
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/pkg/network"
	"encoding/json"
	"os"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	FIND_NODE_REQUEST  string = "FIND_NODE"
	FIND_NODE_RESPONSE string = "FIND_NODE_RESPONSE"
	STORE_REQUEST      string = "STORE"
	STORE_RESPONSE     string = "STORE_RESPONSE"
	LOOKUP_REQUEST     string = "LOOKUP"
	LOOKUP_RESPONSE    string = "LOOKUP_RESPONSE"
	PING               string = "PING"
	PONG               string = "PONG"
)

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg *network.Message, node IKademliaNode) error

type TempMessageHandler func(msg *network.Message, contactCh chan tempHandlersPkg.ContactsWithId, valueCh chan string) error

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
}

type tempHandlerKey struct {
	msgType   string
	msgId     *kademliaID.KademliaID
	contactCh chan tempHandlersPkg.ContactsWithId
	valueCh   chan string
}

type valueWithId struct {
	value string
	msgId *kademliaID.KademliaID
}

// BaseNode provides common functionality that can be embedded
type KademliaNode struct {
	address        network.Address
	network        network.Network
	connection     network.Connection
	handlers       map[string]MessageHandler
	tempHandlers   map[tempHandlerKey]TempMessageHandler
	closed         bool
	closeMu        sync.RWMutex
	routingTable   *RoutingTable
	messageChanMap map[kademliaID.KademliaID]chan kademliaID.KademliaID
}

type KademliaNodeData struct {
	RoutingTable *RoutingTable
}

// NewKademliaNode creates a new Kademlia node
func NewKademliaNode(network network.Network, addr network.Address) (*KademliaNode, error) {
	conn, err := network.Listen(addr)
	if err != nil {
		return nil, err
	}

	contact := kademliaContact.NewContact(kademliaID.NewRandomKademliaID(), addr.String())
	routingTable := NewRoutingTable(contact)

	return &KademliaNode{
		address:        addr,
		network:        network,
		connection:     conn,
		handlers:       make(map[string]MessageHandler),
		tempHandlers:   make(map[tempHandlerKey]TempMessageHandler),
		closed:         false,
		routingTable:   routingTable,
		messageChanMap: make(map[kademliaID.KademliaID]chan kademliaID.KademliaID),
	}, nil
}

func (kn *KademliaNode) GetNodeData() interface{} {
	return &KademliaNodeData{
		RoutingTable: kn.routingTable,
	}
}

// GetRoutingTable provides direct access to the routing table
func (kn *KademliaNode) GetRoutingTable() *RoutingTable {
	return kn.routingTable
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
			tempHandler, tempExists := kn.tempHandlers[tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID}]
			println("PayloadType: ", msg.PayloadType, " MessageID: ", msg.MessageID, " FromID: ", msg.FromID, " TempExists", tempExists, "TempHandler", tempHandler, " Port", kn.Address().String())

			if tempExists && tempHandler != nil {
				go tempHandler(&msg, tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID}.contactCh, tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID}.valueCh)
				delete(kn.tempHandlers, tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID})
				kn.GetRoutingTable().AddContact(kademliaContact.NewContact(msg.FromID, msg.From.String()))
				return
			}
			if !exists {
				log.WithField("msgType", msg.PayloadType).WithField("func", "KademliaNode/Start").Debugf("No handler found, using default")
				handler, exists = handlers["DEFAULT"]
			}

			if exists && handler != nil {
				// Check if this is a PONG message and if there's a message channel for it
				if msg.PayloadType == PONG {
					msgChan, chanExists := kn.getMessageChan(msg.MessageID)
					if chanExists && msg.MessageID != nil {
						// Send the message ID to the channel
						msgChan <- *msg.MessageID
						log.WithField("msgID", msg.MessageID.String()).WithField("func", "KademliaNode/Start").Debugf("PING message ID sent to channel")
						kn.GetRoutingTable().AddContact(kademliaContact.NewContact(msg.FromID, msg.From.String()))
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

// Handle registers a message handler for a specific message type, always online
func (kn *KademliaNode) Handle(msgType string, handler MessageHandler) {
	kn.handlers[msgType] = handler
}

func (kn *KademliaNode) TempHandle(msgType string, msgId *kademliaID.KademliaID, handler TempMessageHandler, contactCh chan tempHandlers.ContactsWithId, valueCh chan string) {
	println("TempHandle called with msgType:", msgType, " msgId:", msgId)
	kn.tempHandlers[tempHandlerKey{msgType: msgType, msgId: msgId, contactCh: contactCh, valueCh: valueCh}] = handler
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

	return kn.send(to, PING, []byte("ping"), kademliaID.NewRandomKademliaID())
}

func (kn *KademliaNode) SendPongMessage(to network.Address, messageID *kademliaID.KademliaID) error {
	return kn.send(to, PONG, []byte("pong"), messageID)
}

func (kn *KademliaNode) LookupContact(targetID *kademliaID.KademliaID) *kademliaContact.Contact {
	res := kn.lookupContact(targetID)
	return res
}

func (kn *KademliaNode) SendFindNode(to network.Address, messageID *kademliaID.KademliaID) error {
	return kn.send(to, FIND_NODE_REQUEST, []byte("find_node"), messageID)
}

func (kn *KademliaNode) SendFindNodeResponse(to network.Address, contacts []kademliaContact.Contact, messageID *kademliaID.KademliaID) error {
	// Serialize contacts to bytes
	data, err := json.Marshal(contacts)
	if err != nil {
		return err
	}
	return kn.send(to, FIND_NODE_RESPONSE, data, messageID)
}

func (kn *KademliaNode) lookupContact(targetID *kademliaID.KademliaID) *kademliaContact.Contact {
	shortlist := kn.routingTable.FindClosestContacts(targetID, kademliaBucket.GetBucketSize())

	return iterativeFindNode(kn, targetID, &shortlist)
}

func iterativeFindNode(kn *KademliaNode, targetID *kademliaID.KademliaID, shortlist *[]kademliaContact.Contact) *kademliaContact.Contact {
	if len(*shortlist) == 0 {
		println("Shortlist is empty, returning nil")
		return nil
	}
	closestNode := (*shortlist)[0]
	for {
		// Send FIND_NODE to the closest node
		// Update shortlist with new contacts
		// Check if closest node has changed, if not fetch 3 more nodes from shortlist
		// Stop if we have found the node or we have queried all nodes in the shortlist

		//Get alpha from env variables and parse as a positive int
		//If not set, default to 3
		//If set but not a positive int, log a warning and default to 3
		alpha := 3
		if val, err := strconv.Atoi(os.Getenv("ALPHA")); err == nil && val > 0 {
			alpha = val
		} else {
			log.Warn("ALPHA not set or invalid, defaulting to 3")
		}

		messageIDs := []*kademliaID.KademliaID{kademliaID.NewRandomKademliaID(), kademliaID.NewRandomKademliaID(), kademliaID.NewRandomKademliaID()}

		contactChs := make([]chan tempHandlers.ContactsWithId, alpha)
		for i := range contactChs {
			contactChs[i] = make(chan tempHandlers.ContactsWithId, 1)
		}

		for i, id := range messageIDs {

			kn.TempHandle(FIND_NODE_RESPONSE, id, tempHandlers.FindNodeResponseTempHandler, contactChs[i], nil)
		}

		// Send FIND_NODE to the three closest nodes concurrently

		contactsToQuery := []kademliaContact.Contact{}
		for i := 0; i < alpha && i < len(*shortlist); i++ {
			contactsToQuery = append(contactsToQuery, (*shortlist)[i])
		}
		var wg sync.WaitGroup
		wg.Add(len(contactsToQuery))

		for i, contact := range contactsToQuery {
			go func() {
				defer wg.Done()
				kn.SendFindNode(contact.GetNetworkAddress(), messageIDs[i]) // Use appropriate message ID
			}()
		}
		wg.Wait()

		// Check channels for new contacts
		for i, ch := range contactChs {
			go func() {
				select {
				case data := <-ch:
					// Delete from messageIDs the messageID that was done
					for j, id := range messageIDs {
						if *id == *data.MsgId {
							messageIDs = append(messageIDs[:j], messageIDs[j+1:]...)
							break
						}
					}
					if i == 3 {
						break
					}
				case <-time.After(30 * time.Second):
					for j, id := range messageIDs {
						if *id == *messageIDs[i] {
							close(ch)
							messageIDs = append(messageIDs[:j], messageIDs[j+1:]...)
							break
						}
					}

				}
				close(ch)
			}()

		}
		break
	}

	return &closestNode
}
