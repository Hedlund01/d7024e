package kademlia

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	kademliaBucket "d7024e/internal/kademlia/bucket"
	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/internal/kademlia/shortlist"
	"d7024e/pkg/network"

	log "github.com/sirupsen/logrus"
)

const (
	FIND_NODE_REQUEST   string = "FIND_NODE"
	FIND_NODE_RESPONSE  string = "FIND_NODE_RESPONSE"
	STORE_REQUEST       string = "STORE"
	STORE_RESPONSE      string = "STORE_RESPONSE"
	LOOKUP_REQUEST      string = "LOOKUP"
	LOOKUP_RESPONSE     string = "LOOKUP_RESPONSE"
	PING                string = "PING"
	PONG                string = "PONG"
	FIND_VALUE_REQUEST  string = "FIND_VALUE"
	FIND_VALUE_RESPONSE string = "FIND_VALUE_RESPONSE"
)

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg *network.Message, node IKademliaNode) error

type TempMessageHandler func(msg *network.Message, contactCh chan []kademliaContact.Contact, valueCh chan []byte) error

// IKademliaNode defines the interface that all Kademlia node types must implement
type IKademliaNode interface {
	// Core networking methods
	send(to network.Address, msgType string, data []byte, messageID *kademliaID.KademliaID, expectReply bool) error
	SendPingMessage(to network.Address) error
	SendPongMessage(to network.Address, messageID *kademliaID.KademliaID) error
	SendFindNode(to network.Address, messageID *kademliaID.KademliaID, id *kademliaID.KademliaID) error
	SendFindNodeResponse(to network.Address, contacts []kademliaContact.Contact, messageID *kademliaID.KademliaID) error
	FindNode(targetID *kademliaID.KademliaID) *kademliaContact.Contact
	Join(contact *kademliaContact.Contact)
	Store(value []byte) error
	StoreValue(data []byte, hash *kademliaID.KademliaID) error
	GetValue(hash *kademliaID.KademliaID) ([]byte, error)
	TempHandle(msgType string, msgId *kademliaID.KademliaID, handler TempMessageHandler, contactCh chan []kademliaContact.Contact, valueCh chan []byte)
	SendStoreResponse(to network.Address, err error, id *kademliaID.KademliaID) error
	Address() network.Address
	SendFindValueResponse(to network.Address, value []byte, messageID *kademliaID.KademliaID) error
	FindValue(id *kademliaID.KademliaID) ([]byte, error)

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
	valueCh   chan []byte
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
	storage             map[kademliaID.KademliaID][]byte
}

type KademliaNodeData struct {
	RoutingTable *RoutingTable
}

// Map for message channels with mutex for concurrent access
type messageChanMap struct {
	mu    sync.RWMutex
	chMap map[kademliaID.KademliaID]chan kademliaID.KademliaID
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
		storage:             make(map[kademliaID.KademliaID][]byte),
	}, nil
}

func (kn *KademliaNode) GetNodeData() any {
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
			if err != nil {
				if !kn.IsClosed() {
					log.Printf("KademliaNode %s failed to receive message: %v", kn.Address().String(), err)
				}
				return
			}

			kn.closeMu.RLock()
			handlers := kn.GetHandlers()
			handler, exists := handlers[msg.PayloadType]
			tempHandler, tempExists := kn.tempHandlers[tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID}]
			kn.closeMu.RUnlock()

			if tempExists && tempHandler != nil {
				msgChan, chanExists := kn.getMessageChan(msg.MessageID)
				if chanExists && msg.MessageID != nil {
					msgChan <- *msg.MessageID
					kn.GetRoutingTable().AddContact(kademliaContact.NewContact(msg.FromID, msg.From.String()))
					kn.closeMu.RLock()
					channels := kn.tempHandlerChannels[tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID}]
					kn.closeMu.RUnlock()
					go tempHandler(&msg, channels.contactCh, channels.valueCh)
					kn.closeMu.Lock()
					delete(kn.tempHandlers, tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID})
					delete(kn.tempHandlerChannels, tempHandlerKey{msgType: msg.PayloadType, msgId: msg.MessageID})
					kn.closeMu.Unlock()
				}
				continue
			}
			if !exists {
				log.WithField("msgType", msg.PayloadType).WithField("func", "KademliaNode/Start").Debugf("No handler found, continuing...")
				continue
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
						continue
					}
				}

				go handler(&msg, kn)
			}
		}
	}()
}

// Address returns the node's address
func (kn *KademliaNode) Address() network.Address {
	return kn.address
}

// Handle registers a message handler for a specific message type, always online
func (kn *KademliaNode) Handle(msgType string, handler MessageHandler) {
	kn.closeMu.Lock()
	kn.handlers[msgType] = handler
	kn.closeMu.Unlock()
}

func (kn *KademliaNode) TempHandle(msgType string, msgId *kademliaID.KademliaID, handler TempMessageHandler, contactCh chan []kademliaContact.Contact, valueCh chan []byte) {
	kn.closeMu.Lock()
	kn.tempHandlers[tempHandlerKey{msgType: msgType, msgId: msgId}] = handler
	kn.tempHandlerChannels[tempHandlerKey{msgType: msgType, msgId: msgId}] = tempHandlerChannels{contactCh: contactCh, valueCh: valueCh}
	kn.closeMu.Unlock()
}

// Close shuts down the Kademlia node
func (kn *KademliaNode) Close() error {
	kn.closeMu.Lock()
	kn.closed = true

	// Clean up all message channels to prevent goroutine leaks
	kn.messageChanMap.mu.Lock()
	for id, ch := range kn.messageChanMap.chMap {
		close(ch)
		delete(kn.messageChanMap.chMap, id)
	}
	kn.messageChanMap.mu.Unlock()

	// Clean up temp handlers
	kn.tempHandlers = make(map[tempHandlerKey]TempMessageHandler)
	kn.tempHandlerChannels = make(map[tempHandlerKey]tempHandlerChannels)

	kn.closeMu.Unlock()
	return kn.connection.Close()
}

// GetConnection returns the connection for use by concrete implementations
func (kn *KademliaNode) GetConnection() network.Connection {
	return kn.connection
}

// GetHandlers returns the handlers map for use by concrete implementations
func (kn *KademliaNode) GetHandlers() map[string]MessageHandler {
	return kn.handlers
}

// IsClosed returns whether the node is closed
func (kn *KademliaNode) IsClosed() bool {
	kn.closeMu.RLock()
	defer kn.closeMu.RUnlock()
	return kn.closed
}

// CheckReply waits for a reply on the message channel associated with the given message ID
func checkReply(kn *KademliaNode, messageID *kademliaID.KademliaID) {
	if messageID == nil {
		return
	}
	msgChan, exists := kn.getMessageChan(messageID)
	if !exists {
		return
	}

	select {
	case _, ok := <-msgChan:
		if !ok {
			// Channel was closed, node is shutting down
			return
		}
	case <-time.After(15 * time.Second):
	}
	kn.messageChanMap.mu.Lock()
	delete(kn.messageChanMap.chMap, *messageID)
	kn.messageChanMap.mu.Unlock()
}

// Send sends a message to the target address
func (kn *KademliaNode) send(to network.Address, msgType string, data []byte, messageID *kademliaID.KademliaID, expectReply bool) error {
	connection, err := kn.network.Dial(to)
	if err != nil {
		return err
	}
	defer connection.Close()

	if expectReply {
		kn.addToMessageChanMap(messageID)
		go checkReply(kn, messageID)
	}

	msg := &network.Message{
		From:        kn.address,
		To:          to,
		Payload:     data,
		PayloadType: msgType,
		FromID:      kn.GetRoutingTable().GetMe().ID,
		MessageID:   messageID,
	}

	return connection.Send(msg)
}

func (kn *KademliaNode) FindNode(targetID *kademliaID.KademliaID) *kademliaContact.Contact {
	result := kn.findNode(targetID)
	if result.Len() == 0 {
		return nil
	}
	return result.GetClosestContact()
}

func (kn *KademliaNode) Join(contact *kademliaContact.Contact) {
	kn.GetRoutingTable().AddContact(*contact)
	kn.findNode(kn.GetRoutingTable().GetMe().ID)
}

func (kn *KademliaNode) FindValue(id *kademliaID.KademliaID) ([]byte, error) {
	localVal, err := kn.GetValue(id)
	if err == nil && localVal != nil {
		return localVal, nil
	}
	return kn.findValue(id)
}

// Store : stores the given value on the kademlia network, with the hash of the data as a key
func (kn *KademliaNode) Store(value []byte) error {
	log.WithField("func", "Store").Debugf("Storing value: %s", string(value))
	id := kademliaID.NewKademliaID(string(value))
	shortlist := kn.findNode(id)
	if shortlist == nil {
		return errors.New("shortlist is nil, cannot send store value")
	}
	errorsCh := make(chan error, len(shortlist.GetAllProbedContacts()))
	log.WithField("func", "Store").Debugf("Storing value on %d nodes", len(shortlist.GetAllProbedContacts()))
	var wg sync.WaitGroup
	wg.Add(len(shortlist.GetAllProbedContacts()))
	for _, contact := range shortlist.GetAllProbedContacts() {
		if contact.ID.Equals(kn.GetRoutingTable().GetMe().ID) { // Skip self, do not send store request RPC to itself, in case we have somehow added ourselves to the shortlist
			wg.Done()
			continue
		}
		go kn.sendStore(contact.GetNetworkAddress(), id, value, &wg, errorsCh)
	}
	wg.Wait()
	close(errorsCh)

	allErrors := make([]error, 0)
	for err := range errorsCh {
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}
	return errors.Join(allErrors...)
}

func (kn *KademliaNode) SendPingMessage(to network.Address) error {
	return kn.send(to, PING, []byte("ping"), kademliaID.NewRandomKademliaID(), true)
}

func (kn *KademliaNode) SendPongMessage(to network.Address, messageID *kademliaID.KademliaID) error {
	return kn.send(to, PONG, []byte("pong"), messageID, false)
}

func (kn *KademliaNode) SendFindNode(to network.Address, messageID *kademliaID.KademliaID, id *kademliaID.KademliaID) error {
	payload, err := json.Marshal(id)
	if err != nil {
		return err
	}
	return kn.send(to, FIND_NODE_REQUEST, payload, messageID, true)
}

func (kn *KademliaNode) SendFindNodeResponse(to network.Address, contacts []kademliaContact.Contact, messageID *kademliaID.KademliaID) error {
	// Serialize contacts to bytes
	data, err := json.Marshal(contacts)
	if err != nil {
		return err
	}
	return kn.send(to, FIND_NODE_RESPONSE, data, messageID, false)
}

func (kn *KademliaNode) SendFindValue(to network.Address, messageID *kademliaID.KademliaID, id *kademliaID.KademliaID) error {
	payload, err := json.Marshal(id)
	if err != nil {
		return err
	}
	return kn.send(to, FIND_VALUE_REQUEST, payload, messageID, true)
}

func (kn *KademliaNode) SendFindValueResponse(to network.Address, value []byte, messageID *kademliaID.KademliaID) error {
	return kn.send(to, FIND_VALUE_RESPONSE, value, messageID, false)
}

func (kn *KademliaNode) SendStoreResponse(to network.Address, err error, id *kademliaID.KademliaID) error {
	if err != nil {
		return kn.send(to, STORE_RESPONSE, []byte("REJECT"), id, false)
	}
	return kn.send(to, STORE_RESPONSE, []byte("OK"), id, true)
}

// StoreValue : store the given data with the given hash in the nodes storage
func (kn *KademliaNode) StoreValue(data []byte, hash *kademliaID.KademliaID) error {
	kn.closeMu.Lock()
	defer kn.closeMu.Unlock()
	kn.storage[*hash] = data
	return nil
}

// GetValue : the value with the given hash from the nodes storage
func (kn *KademliaNode) GetValue(hash *kademliaID.KademliaID) ([]byte, error) {
	kn.closeMu.RLock()
	defer kn.closeMu.RUnlock()
	value, exists := kn.storage[*hash]
	if !exists {
		return nil, errors.New("value not found")
	}
	return value, nil
}

func (kn *KademliaNode) sendStore(to network.Address, hash *kademliaID.KademliaID, value []byte, wg *sync.WaitGroup, errorsCh chan error) {
	defer wg.Done()
	msgID := kademliaID.NewRandomKademliaID()
	log.WithField("to", to.String()).WithField("msgID", msgID.String()).WithField("func", "sendStore").Debugf("Sending STORE request to %s", to.String())

	valCh := make(chan []byte, 1)
	kn.TempHandle(STORE_RESPONSE, msgID, StoreResponseTempHandler, nil, valCh)
	data, marshalErr := json.Marshal(shortlist.StoreData{Hash: hash, Value: value})
	if marshalErr != nil {
		log.WithField("msgID", msgID.String()).WithField("func", "sendStore").Errorf("Failed to marshal store data: %v", marshalErr)
		errorsCh <- marshalErr
	}

	err := kn.send(to, STORE_REQUEST, data, msgID, true)
	if err != nil {
		log.WithField("msgID", msgID.String()).WithField("func", "sendStore").Errorf("Failed to send STORE request to %s: %v", to.String(), err)
		errorsCh <- err
	}

	select {
	case result := <-valCh:
		if string(result) != "OK" {
			errorsCh <- fmt.Errorf("STORE request rejected by %s", to.String())
		}
	case <-time.After(15 * time.Second):
		log.WithField("msgID", msgID.String()).WithField("func", "sendStore").Errorf("Timeout waiting for STORE_RESPONSE")
		errorsCh <- fmt.Errorf("timeout waiting for STORE_RESPONSE from %s", to.String())
	}
}

func (kn *KademliaNode) findNode(targetID *kademliaID.KademliaID) *shortlist.Shortlist {
	return iterativeLookup(kn, targetID, FIND_NODE_RESPONSE)
}

func (kn *KademliaNode) findValue(targetID *kademliaID.KademliaID) ([]byte, error) {
	value := iterativeLookup(kn, targetID, FIND_VALUE_RESPONSE).GetValue()
	if value.Value == nil {
		log.WithField("func", "findValue").Debugf("Value not found in network")
		return nil, errors.New("value not found in network")
	}
	return value.Value, nil
}

func iterativeLookup(kn *KademliaNode, targetID *kademliaID.KademliaID, handlerType string) *shortlist.Shortlist {
	shortlistParam := kn.GetRoutingTable().FindClosestContacts(targetID, kademliaBucket.GetBucketSize())
	if len(shortlistParam) == 0 {
		log.Warn("Shortlist is empty, returning nil")
		return nil
	}

	alpha := 3
	if val, err := strconv.Atoi(os.Getenv("ALPHA")); err == nil && val > 0 {
		alpha = val
	} else {
		log.Warn("ALPHA not set or invalid, defaulting to 3")
	}

	list := shortlist.NewShortlist(targetID, kademliaBucket.GetBucketSize(), alpha)
	list.AddContacts(shortlistParam)

	probeCh := make(chan kademliaContact.Contact, alpha)

	// Start the go rutines that will consume the shortlist and send out FIND_NODE requests
	for range alpha {
		go func() {
			for {
				contact, ok := <-probeCh
				if !ok {
					return
				}
				messageID := kademliaID.NewRandomKademliaID()

				contactCh := make(chan []kademliaContact.Contact, alpha)
				valueCh := make(chan []byte, 1)

				switch handlerType {
				case FIND_NODE_RESPONSE:
					kn.TempHandle(FIND_NODE_RESPONSE, messageID, FindNodeResponseTempHandler, contactCh, nil)
					err := kn.SendFindNode(contact.GetNetworkAddress(), messageID, targetID)
					if err != nil {
						log.Error("Failed to send FindNode in iterativeFindNode, error: ", err)
					}
					select {
					case contacts := <-contactCh:
						close(contactCh)
						close(valueCh)
						list.AddContacts(contacts)
						list.MarkSucceeded(contact)
					case <-time.After(30 * time.Second):
						list.MarkFailed(contact)
					}
				case FIND_VALUE_RESPONSE:
					kn.TempHandle(FIND_VALUE_RESPONSE, messageID, FindValueResponseTempHandler, contactCh, valueCh)
					err := kn.SendFindValue(contact.GetNetworkAddress(), messageID, targetID)
					if err != nil {
						log.Error("Failed to send FindValue in iterativeFindNode, error: ", err)
					}
					select {
					case value := <-valueCh:
						close(contactCh)
						close(valueCh)
						list.SetValue(targetID, value)
						list.MarkSucceeded(contact)
						list.SetTargetFound()
					case contacts := <-contactCh:
						close(contactCh)
						close(valueCh)
						list.AddContacts(contacts)
						list.MarkSucceeded(contact)
					case <-time.After(30 * time.Second):
						close(contactCh)
						close(valueCh)
						list.MarkFailed(contact)
					}
				}
			}
		}()
	}

	for {
		if list.TargetFound() {
			log.WithField("func", "iterativeFindNode").Debugf("Target found in shortlist")
			close(probeCh)
			return list
		}

		if list.HasUnprobed() && list.HasImproved() && list.GetProbingCount() < alpha {
			contact, error := list.GetUnprobed()
			if error != nil {
				continue
			}
			probeCh <- contact
		} else if !list.HasImproved() && list.GetProbingCount() == 0 {
			log.WithField("func", "iterativeFindNode").Debugf("No improvement and no active queries, ending search")
			close(probeCh)
			probeRemaining(list, kn, targetID, alpha, handlerType)
			return list
		}
	}
}

func probeRemaining(list *shortlist.Shortlist, kn *KademliaNode, targetID *kademliaID.KademliaID, alpha int, handlerType string) {
	contacts, err := list.GetAllUnprobed()
	if err != nil {
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(contacts))
	probeCh := make(chan kademliaContact.Contact, len(contacts))
	for range contacts {
		go func() {
			contact := <-probeCh
			messageID := kademliaID.NewRandomKademliaID()

			contactCh := make(chan []kademliaContact.Contact, alpha)
			valueCh := make(chan []byte, 1)

			switch handlerType {
			case FIND_NODE_RESPONSE:
				kn.TempHandle(FIND_NODE_RESPONSE, messageID, FindNodeResponseTempHandler, contactCh, nil)
				err := kn.SendFindNode(contact.GetNetworkAddress(), messageID, targetID)
				if err != nil {
					log.Error("Failed to send FindNode in iterativeFindNode, error: ", err)
				}
				select {
				case contacts := <-contactCh:
					close(contactCh)
					close(valueCh)
					list.AddContacts(contacts)
					list.MarkSucceeded(contact)
				case <-time.After(30 * time.Second):
					list.MarkFailed(contact)
				}
			case FIND_VALUE_RESPONSE:
				kn.TempHandle(FIND_VALUE_RESPONSE, messageID, FindValueResponseTempHandler, contactCh, valueCh)
				err := kn.SendFindValue(contact.GetNetworkAddress(), messageID, targetID)
				if err != nil {
					log.Error("Failed to send FindValue in iterativeFindNode, error: ", err)
				}
				select {
				case value := <-valueCh:
					close(contactCh)
					close(valueCh)
					list.SetValue(targetID, value)
					list.MarkSucceeded(contact)
				case contacts := <-contactCh:
					close(contactCh)
					close(valueCh)
					list.AddContacts(contacts)
					list.MarkSucceeded(contact)
				case <-time.After(30 * time.Second):
					close(contactCh)
					close(valueCh)
					list.MarkFailed(contact)
				}
			}
			wg.Done()
		}()
	}
	for _, contact := range contacts {
		probeCh <- contact
	}
	wg.Wait()
	close(probeCh)
}
