package kademlia

import (
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/internal/mock"
	"d7024e/pkg/network"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStoreResponseTempHandlerCorrectData(t *testing.T) {
	msg := &network.Message{
		Payload: []byte(`OK`),
	}
	valueCh := make(chan []byte)
	defer close(valueCh)

	// Run the handler in a goroutine to avoid deadlock
	go func() {
		if err := StoreResponseTempHandler(msg, nil, valueCh); err != nil {
			t.Errorf("StoreResponseTempHandler got an error: %v", err)
		}
	}()

	select {
	case value := <-valueCh:
		assert.Equal(t, `OK`, string(value))
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "StoreResponseTempHandler did not send a value in time")
	}
}

func TestStoreRequestHandler(t *testing.T) {
	mockNet := mock.NewMockNetwork()
	nodeA, _ := NewKademliaNode(mockNet, network.Address{IP: "127.0.0.1", Port: 8080})
	nodeB, _ := NewKademliaNode(mockNet, network.Address{IP: "127.0.0.1", Port: 8081})

	msg := &network.Message{
		From:        nodeB.Address(),
		FromID:      nodeB.GetRoutingTable().me.ID,
		MessageID:   kademliaID.NewRandomKademliaID(),
		To:          nodeA.Address(),
		PayloadType: STORE_REQUEST,
		Payload:     []byte("Hello, World!"),
	}

	go func() {
		err := StoreRequestHandler(msg, nodeA)
		assert.NoError(t, err)
	}()

	<-time.After(5 * time.Second)

	valBytes, err := nodeA.GetValue(msg.MessageID) // This will block until the value is stored
	assert.NoErrorf(t, err, "Failed to get value from nodeA")
	assert.Equal(t, `Hello, World!`, string(valBytes))

}
