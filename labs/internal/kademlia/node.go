package kademlia

import (
	kademliaBucket "d7024e/internal/kademlia/bucket"
	kademliaContact "d7024e/internal/kademlia/contact"
	"d7024e/internal/kademlia/handlers/tempHandlers"
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

type TempMessageHandler func(msg *network.Message, contactCh chan []kademliaContact.Contact, valueCh chan string) error

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

// Type for temporary handler keys
type tempHandlerKey struct {
	msgType string
	msgId   *kademliaID.KademliaID
}

type tempHandlerChannels struct {
	contactCh chan []kademliaContact.Contact
	valueCh   chan string
}

// Type for value with ID
type valueWithId struct {
	value string
	msgId *kademliaID.KademliaID
}

// BaseNode provides common functionality that can be embedded
type KademliaNode struct {
	address             network.Address
	network             network.Network
	connection          network.Connection
	handlers            map[string]MessageHandler
	tempHandlers        map[tempHandlerKey]TempMessageHandler
	tempHandlerChannels map[tempHandlerKey]tempHandlerChannels
	closed              bool
	closeMu             sync.RWMutex
	routingTable        *RoutingTable
	messageChanMap      messageChanMap
}

type KademliaNodeData struct {
	RoutingTable *RoutingTable
}

// Map for message channels with mutex for concurrent access
type messageChanMap struct {
	mu    sync.RWMutex
	chMap map[kademliaID.KademliaID]chan kademliaID.KademliaID
}

// Type for currentClosest node with mutex for concurrent access
type currentClosest struct {
	mu      sync.RWMutex
	contact kademliaContact.Contact
}

type shortlist struct {
	mu    sync.RWMutex
	queue []kademliaContact.Contact
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
		address:             addr,
		network:             network,
		connection:          conn,
		handlers:            make(map[string]MessageHandler),
		tempHandlers:        make(map[tempHandlerKey]TempMessageHandler),
		tempHandlerChannels: make(map[tempHandlerKey]tempHandlerChannels),
		closed:              false,
		routingTable:        routingTable,
		messageChanMap:      messageChanMap{chMap: make(map[kademliaID.KademliaID]chan kademliaID.KademliaID)},
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
	kn.messageChanMap.mu.RLock()
	defer kn.messageChanMap.mu.RUnlock()
	if messageID == nil {
		return nil, false
	}
	chn, exists := kn.messageChanMap.chMap[*messageID]
	return chn, exists
}

func (kn *KademliaNode) addToMessageChanMap(messageID *kademliaID.KademliaID) {
	kn.messageChanMap.mu.Lock()
	defer kn.messageChanMap.mu.Unlock()
	if messageID == nil {
		return
	}
	kn.messageChanMap.chMap[*messageID] = make(chan kademliaID.KademliaID, 1)
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
				msgChan, chanExists := kn.getMessageChan(msg.MessageID)
				if chanExists && msg.MessageID != nil {
					msgChan <- *msg.MessageID
					log.WithField("msgID", msg.MessageID.String()).WithField("func", "KademliaNode/Start").Debugf(`%s message ID sent to channel`, msg.PayloadType)
				}
				kn.GetRoutingTable().AddContact(kademliaContact.NewContact(msg.FromID, msg.From.String()))
				channels := kn.tempHandlerChannels[tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID}]
				go tempHandler(&msg, channels.contactCh, channels.valueCh)
				delete(kn.tempHandlers, tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID})
				delete(kn.tempHandlerChannels, tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID})

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

func (kn *KademliaNode) TempHandle(msgType string, msgId *kademliaID.KademliaID, handler TempMessageHandler, contactCh chan []kademliaContact.Contact, valueCh chan string) {
	println("TempHandle called with msgType:", msgType, " msgId:", msgId)
	kn.tempHandlers[tempHandlerKey{msgType: msgType, msgId: msgId}] = handler
	kn.tempHandlerChannels[tempHandlerKey{msgType: msgType, msgId: msgId}] = tempHandlerChannels{contactCh: contactCh, valueCh: valueCh}
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

// CheckReply waits for a reply on the message channel associated with the given message ID
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
	b.messageChanMap.mu.Lock()
	delete(b.messageChanMap.chMap, *messageID)
	b.messageChanMap.mu.Unlock()
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
	list := kn.routingTable.FindClosestContacts(targetID, kademliaBucket.GetBucketSize())

	return iterativeFindNodev2(kn, targetID, &list)
}

func iterativeFindNodev2(kn *KademliaNode, targetID *kademliaID.KademliaID, shortlistParam *[]kademliaContact.Contact) *kademliaContact.Contact {
	if len(*shortlistParam) == 0 {
		println("Shortlist is empty, returning nil")
		return nil
	}

	alpha := 3
	if val, err := strconv.Atoi(os.Getenv("ALPHA")); err == nil && val > 0 {
		alpha = val
	} else {
		log.Warn("ALPHA not set or invalid, defaulting to 3")
	}

	// Keep track of the active queries, should be only alpha at a time
	activeQueries := 0

	// Make our own shortlist copy as a FIFO queue
	list := shortlist{queue: make([]kademliaContact.Contact, 0, len(*shortlistParam))}
	list.mu.Lock()
	list.queue = append(list.queue, (*shortlistParam)...)
	list.mu.Unlock()

	closest := currentClosest{contact: list.queue[0]}

	var wg sync.WaitGroup
	wg.Add(alpha)

	listCh := make(chan kademliaContact.Contact, alpha)

	// Start the go rutines that will consume the shortlist and send out FIND_NODE requests
	for range alpha {
		go func() {
			for {
				contact, ok := <-listCh
				if !ok {
					wg.Done()
					return
				}
				activeQueries++
				messageID := kademliaID.NewRandomKademliaID()
				contactCh := make(chan []kademliaContact.Contact, alpha)
				defer close(contactCh)

				kn.TempHandle(FIND_NODE_RESPONSE, messageID, tempHandlers.FindNodeResponseTempHandler, contactCh, nil)

				err := kn.SendFindNode(contact.GetNetworkAddress(), messageID)
				if err != nil {
					log.Error("Failed to send FindNode in iterativeFindNode, error: ", err)
				}

				select {
				case contacts := <-contactCh:
					closest.mu.Lock()
					closer := checkCloser(closest.contact, contacts, targetID)
					if closer.ID.Less(closest.contact.ID) || closer.ID.Equals(closest.contact.ID) {
						closest.contact = closer
						println("New closest node found: ", closer.ID.String())
						if !closer.ID.Equals(targetID) {
							placeInShortlist(&list, contacts)
						}
					}
					closest.mu.Unlock()
				case <-time.After(30 * time.Second):
				}
				activeQueries--
			}
		}()
	}

	for {
		closest.mu.RLock()
		if closest.contact.ID.Equals(targetID) {
			return &closest.contact
		}
		closest.mu.RUnlock()

		if activeQueries < alpha && len(list.queue) > 0 {
			list.mu.Lock()
			listCh <- list.queue[0]
			list.queue = list.queue[1:]
			list.mu.Unlock()
		}

	}
}

func placeInShortlist(sl *shortlist, contacts []kademliaContact.Contact) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	for _, contact := range contacts {
		if len(sl.queue) == 0 {
			sl.queue = append(sl.queue, contact)
		} else {
			for i, existing := range sl.queue {
				if contact.ID.Equals(existing.ID) || contact.ID.Less(existing.ID) {
					sl.queue = append(sl.queue[:i+1], sl.queue[i:]...) // Create space by shifting right
					sl.queue[i] = contact
					break
				}
			}
		}
	}
}

func checkCloser(currentClosest kademliaContact.Contact, newContacts []kademliaContact.Contact, targetID *kademliaID.KademliaID) kademliaContact.Contact {
	best := currentClosest
	for index := range newContacts {
		contact := newContacts[index]
		distance := contact.ID.CalcDistance(targetID)
		if distance.Less(best.ID.CalcDistance(targetID)) {
			best = contact
		}
	}
	return best
}
