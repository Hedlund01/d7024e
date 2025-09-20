package kademlia

import (
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/internal/mock"
	"d7024e/pkg/network"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFindValueResponseTempHandlerCorrectData(t *testing.T) {
	// Create a test message with payload
	testPayload := []byte("test_data")
	msg := &network.Message{
		PayloadType: FIND_VALUE_RESPONSE,
		From:        network.Address{IP: "192.168.1.2", Port: 8080},
		MessageID:   kademliaID.NewRandomKademliaID(),
		Payload:     testPayload,
	}

	valueCh := make(chan []byte, 1)
	err := FindValueResponseTempHandler(msg, nil, valueCh)

	assert.NoError(t, err)
	assert.Equal(t, testPayload, <-valueCh)
}

func TestFindValueRequestHandlerHasValue(t *testing.T) {
	mockNetwork := mock.NewMockNetwork()
	nodeA, _ := NewKademliaNode(mockNetwork, network.Address{IP: "192.168.1.1", Port: 8080})
	nodeB, _ := NewKademliaNode(mockNetwork, network.Address{IP: "192.168.1.2", Port: 8080})

	// Store a value first
	testValue := []byte("1234567890abcdef1234567890abcdef12345678")
	targetID := kademliaID.NewKademliaID(string(testValue))
	nodeA.StoreValue(testValue, targetID)
	msg := &network.Message{
		PayloadType: FIND_VALUE_REQUEST,
		From:        nodeB.Address(),
		MessageID:   kademliaID.NewRandomKademliaID(),
		Payload:     []byte(targetID.String()),
	}

	valCh := make(chan []byte, 1)

	nodeB.Handle(FIND_VALUE_RESPONSE, func(msg *network.Message, node IKademliaNode) error {
		valCh <- msg.Payload
		return nil
	})
	nodeA.Start()
	nodeB.Start()

	<-time.After(1 * time.Second) // Give nodes time to start
	err := FindValueRequestHandler(msg, nodeA)
	assert.NoError(t, err)

	select {
	case val := <-valCh:
		assert.Equal(t, val, testValue)
	case <-time.After(15 * time.Second):
		t.Error("Timeout waiting for FIND_VALUE_RESPONSE")
	}

}

func TestFindValueRequestHandlerValueNotFound(t *testing.T) {
	mockNetwork := mock.NewMockNetwork()
	nodeA, _ := NewKademliaNode(mockNetwork, network.Address{IP: "192.168.1.1", Port: 8080})
	nodeB, _ := NewKademliaNode(mockNetwork, network.Address{IP: "192.168.1.2", Port: 8080})

	// Store a value first
	testValue := []byte("1234567890abcdef1234567890abcdef12345678")
	targetID := kademliaID.NewKademliaID(string(testValue))
	msg := &network.Message{
		PayloadType: FIND_VALUE_REQUEST,
		From:        nodeB.Address(),
		MessageID:   kademliaID.NewRandomKademliaID(),
		Payload:     []byte(targetID.String()),
	}

	valCh := make(chan []byte, 1)

	nodeB.Handle(FIND_VALUE_RESPONSE, func(msg *network.Message, node IKademliaNode) error {
		valCh <- msg.Payload
		return nil
	})
	nodeA.Start()
	nodeB.Start()

	<-time.After(1 * time.Second) // Give nodes time to start
	err := FindValueRequestHandler(msg, nodeA)
	assert.NoError(t, err)

	select {
	case val := <-valCh:
		assert.Nil(t, val)
	case <-time.After(15 * time.Second):
		t.Error("Timeout waiting for FIND_VALUE_RESPONSE")
	}
}
