package kademlia

import (
	"encoding/json"
	"testing"
	"time"

	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/internal/mock"
	"d7024e/pkg/network"

	"github.com/stretchr/testify/assert"
)

func TestFindValueResponseTempHandlerCorrectData(t *testing.T) {
	// Create a test message with payload
	testPayload := []byte("test_data")
	msg := &network.Message{
		PayloadType: FIND_VALUE_RESPONSE,
		From:        network.Address{IP: "127.0.0.1", Port: 8080},
		To:          network.Address{IP: "127.0.0.1", Port: 8081},
		MessageID:   kademliaID.NewRandomKademliaID(),
		Payload:     testPayload,
	}

	valueCh := make(chan []byte, 1)
	err := FindValueResponseTempHandler(msg, nil, valueCh)

	assert.NoError(t, err)
	assert.Equal(t, testPayload, <-valueCh)
}

func TestFindValueResponseTempHandlerContactsData(t *testing.T) {
	// Create test contacts
	id1 := kademliaID.NewRandomKademliaID()
	id2 := kademliaID.NewRandomKademliaID()
	contacts := []kademliaContact.Contact{
		kademliaContact.NewContact(id1, "127.0.0.1:8080"),
		kademliaContact.NewContact(id2, "127.0.0.1:8081"),
	}

	contactsPayload, err := json.Marshal(contacts)
	assert.NoError(t, err)

	msg := &network.Message{
		PayloadType: FIND_VALUE_RESPONSE,
		From:        network.Address{IP: "127.0.0.1", Port: 8080},
		To:          network.Address{IP: "127.0.0.1", Port: 8081},
		MessageID:   kademliaID.NewRandomKademliaID(),
		Payload:     contactsPayload,
	}

	contactCh := make(chan []kademliaContact.Contact, 1)
	valueCh := make(chan []byte, 1)

	err = FindValueResponseTempHandler(msg, contactCh, valueCh)
	assert.NoError(t, err)

	// Should receive contacts, not raw value
	select {
	case receivedContacts := <-contactCh:
		assert.Len(t, receivedContacts, 2)
		assert.Equal(t, contacts[0].ID.String(), receivedContacts[0].ID.String())
		assert.Equal(t, contacts[1].ID.String(), receivedContacts[1].ID.String())
	case <-time.After(1 * time.Second):
		t.Fatal("Expected to receive contacts")
	}

	// Should not receive anything on value channel
	select {
	case <-valueCh:
		t.Fatal("Should not receive value when contacts are sent")
	default:
		// Expected behavior
	}
}

func TestFindValueRequestHandlerValueFound(t *testing.T) {
	mockNet := mock.NewMockNetwork()
	nodeA, err := NewKademliaNode(mockNet, network.Address{IP: "127.0.0.1", Port: 8080}, *kademliaID.NewRandomKademliaID())
	assert.NoError(t, err)
	nodeB, err := NewKademliaNode(mockNet, network.Address{IP: "127.0.0.1", Port: 8081}, *kademliaID.NewRandomKademliaID())
	assert.NoError(t, err)

	// Store a value in nodeA
	testKey := kademliaID.NewRandomKademliaID()
	testValue := []byte("stored_value")
	nodeA.StoreValue(testValue, testKey)

	// Create request message
	keyPayload, err := json.Marshal(testKey)
	assert.NoError(t, err)

	msg := &network.Message{
		From:        nodeB.Address(),
		FromID:      nodeB.GetRoutingTable().me.ID,
		To:          nodeA.Address(),
		MessageID:   kademliaID.NewRandomKademliaID(),
		PayloadType: FIND_VALUE_REQUEST,
		Payload:     keyPayload,
	}

	// Set up response handler
	responseCh := make(chan *network.Message, 1)
	nodeB.Handle(FIND_VALUE_RESPONSE, func(msg *network.Message, node IKademliaNode) error {
		responseCh <- msg
		return nil
	})

	nodeA.Start()
	nodeB.Start()
	defer nodeA.Close()
	defer nodeB.Close()

	// Handle the request
	err = FindValueRequestHandler(msg, nodeA)
	assert.NoError(t, err)

	// Verify response
	select {
	case response := <-responseCh:
		assert.Equal(t, testValue, response.Payload)
		assert.Equal(t, msg.MessageID.String(), response.MessageID.String())
	case <-time.After(2 * time.Second):
		t.Fatal("Expected to receive FIND_VALUE response")
	}
}

func TestFindValueRequestHandlerValueNotFound(t *testing.T) {
	mockNet := mock.NewMockNetwork()
	nodeA, err := NewKademliaNode(mockNet, network.Address{IP: "127.0.0.1", Port: 8080}, *kademliaID.NewRandomKademliaID())
	assert.NoError(t, err)
	nodeB, err := NewKademliaNode(mockNet, network.Address{IP: "127.0.0.1", Port: 8081}, *kademliaID.NewRandomKademliaID())
	assert.NoError(t, err)
	nodeC, err := NewKademliaNode(mockNet, network.Address{IP: "127.0.0.1", Port: 8082}, *kademliaID.NewRandomKademliaID())
	assert.NoError(t, err)

	// Add some contacts to nodeA's routing table
	nodeA.GetRoutingTable().AddContact(nodeB.GetRoutingTable().me)
	nodeA.GetRoutingTable().AddContact(nodeC.GetRoutingTable().me)

	// Create request for non-existent key
	testKey := kademliaID.NewRandomKademliaID()
	keyPayload, err := json.Marshal(testKey)
	assert.NoError(t, err)

	msg := &network.Message{
		From:        nodeB.Address(),
		FromID:      nodeB.GetRoutingTable().me.ID,
		To:          nodeA.Address(),
		MessageID:   kademliaID.NewRandomKademliaID(),
		PayloadType: FIND_VALUE_REQUEST,
		Payload:     keyPayload,
	}

	// Set up response handler
	responseCh := make(chan *network.Message, 1)
	nodeB.Handle(FIND_VALUE_RESPONSE, func(msg *network.Message, node IKademliaNode) error {
		responseCh <- msg
		return nil
	})

	nodeA.Start()
	nodeB.Start()
	nodeC.Start()
	defer nodeA.Close()
	defer nodeB.Close()
	defer nodeC.Close()

	// Handle the request
	err = FindValueRequestHandler(msg, nodeA)
	assert.NoError(t, err)

	// Verify response contains contacts
	select {
	case response := <-responseCh:
		var contacts []kademliaContact.Contact
		err := json.Unmarshal(response.Payload, &contacts)
		assert.NoError(t, err)
		assert.True(t, len(contacts) > 0, "Should return closest contacts when value not found")

		// Verify the requesting node is not included in the response
		for _, contact := range contacts {
			assert.NotEqual(t, nodeB.GetRoutingTable().me.ID.String(), contact.ID.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Expected to receive FIND_VALUE response")
	}
}

func TestFindValueResponseTempHandlerEmptyPayload(t *testing.T) {
	msg := &network.Message{
		PayloadType: FIND_VALUE_RESPONSE,
		From:        network.Address{IP: "127.0.0.1", Port: 8080},
		To:          network.Address{IP: "127.0.0.1", Port: 8081},
		MessageID:   kademliaID.NewRandomKademliaID(),
		Payload:     []byte{},
	}

	valueCh := make(chan []byte, 1)
	err := FindValueResponseTempHandler(msg, nil, valueCh)

	assert.NoError(t, err)
	assert.Equal(t, []byte{}, <-valueCh)
}

func TestFindValueResponseTempHandlerMalformedJSON(t *testing.T) {
	msg := &network.Message{
		PayloadType: FIND_VALUE_RESPONSE,
		From:        network.Address{IP: "127.0.0.1", Port: 8080},
		To:          network.Address{IP: "127.0.0.1", Port: 8081},
		MessageID:   kademliaID.NewRandomKademliaID(),
		Payload:     []byte("{invalid json}"),
	}

	valueCh := make(chan []byte, 1)
	err := FindValueResponseTempHandler(msg, nil, valueCh)

	assert.NoError(t, err)
	// Should treat malformed JSON as raw value
	assert.Equal(t, []byte("{invalid json}"), <-valueCh)
}
