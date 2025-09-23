package kademlia

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/internal/kademlia/shortlist"
	"d7024e/internal/mock"
	"d7024e/pkg/network"

	"github.com/stretchr/testify/assert"
)

func TestStoreResponseTempHandlerCorrectData(t *testing.T) {
	msg := &network.Message{
		Payload: []byte(`OK`),
	}
	valueCh := make(chan []byte)
	defer close(valueCh)

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
	text := "Hello World"
	data := shortlist.StoreData{Hash: kademliaID.NewKademliaID(text), Value: []byte(text)}
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Errorf("Failed to marshal data: %v", err)
		return
	}
	msg := &network.Message{
		From:        nodeB.Address(),
		FromID:      nodeB.GetRoutingTable().me.ID,
		MessageID:   kademliaID.NewRandomKademliaID(),
		To:          nodeA.Address(),
		PayloadType: STORE_REQUEST,
		Payload:     jsonData,
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		err := StoreRequestHandler(msg, nodeA)
		assert.NoError(t, err)
	})

	wg.Wait()

	valBytes, err := nodeA.GetValue(data.Hash) // This will block until the value is stored
	assert.NoErrorf(t, err, "Failed to get value from nodeA")
	assert.Equal(t, text, string(valBytes))
}
