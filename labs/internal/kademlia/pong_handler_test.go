package kademlia

import (
	kademliaID "d7024e/internal/kademlia/id"
	mock "d7024e/internal/mock"
	net "d7024e/pkg/network"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPongHandler(t *testing.T) {
	network := mock.NewMockNetwork()
	nodeA, err := NewKademliaNode(network, net.Address{
		IP:   "127.0.0.1",
		Port: 8001,
	})
	if err != nil {
		t.Fatalf("Failed to create nodeA: %v", err)
	}
	nodeB, err := NewKademliaNode(network, net.Address{
		IP:   "172.0.0.1",
		Port: 8002,
	})
	if err != nil {
		t.Fatalf("Failed to create nodeB: %v", err)
	}

	done := make(chan net.Message, 1)

	// Set up handlers
	nodeA.Handle("PING", func(msg *net.Message, node IKademliaNode) error {
		// When nodeA receives a PING, it should send a PONG back
		return node.SendPongMessage(msg.From, msg.MessageID)
	})

	nodeB.Handle("PONG", func(msg *net.Message, node IKademliaNode) error {
		// When nodeB receives a PONG, capture it for testing
		done <- *msg
		return PongHandler(msg, node)
	})

	nodeA.Start()
	nodeB.Start()

	// NodeB sends PING to NodeA
	err = nodeB.SendPingMessage(nodeA.Address())
	assert.NoError(t, err)

	// Wait for the PONG response
	msg := <-done
	assert.Equal(t, "PONG", msg.PayloadType)
	assert.Equal(t, nodeA.Address(), msg.From)
	assert.Equal(t, nodeB.Address(), msg.To)

	nodeA.Close()
	nodeB.Close()
}

func TestDoublePong(t *testing.T) {
	network := mock.NewMockNetwork()
	nodeA, err := NewKademliaNode(network, net.Address{
		IP:   "127.0.0.1",
		Port: 8001,
	})
	if err != nil {
		t.Fatalf("Failed to create nodeA: %v", err)
	}
	nodeB, err := NewKademliaNode(network, net.Address{
		IP:   "172.0.0.1",
		Port: 8002,
	})
	if err != nil {
		t.Fatalf("Failed to create nodeB: %v", err)
	}

	done := make(chan net.Message, 2)

	// Set up handlers
	nodeA.Handle("PING", func(msg *net.Message, node IKademliaNode) error {
		// When nodeA receives a PING, it should send a PONG back
		return node.SendPongMessage(msg.From, msg.MessageID)
	})

	nodeB.Handle("PONG", func(msg *net.Message, node IKademliaNode) error {
		// When nodeB receives a PONG, capture it for testing
		done <- *msg
		return PongHandler(msg, node)
	})

	nodeA.Start()
	nodeB.Start()

	// NodeB sends PING to NodeA
	err = nodeB.SendPingMessage(nodeA.Address())
	assert.NoError(t, err)

	err = nodeA.SendPongMessage(nodeB.Address(), kademliaID.NewRandomKademliaID()) // Extra PONG
	assert.NoError(t, err)

	// Wait a moment for messages to be processed
	time.Sleep(100 * time.Millisecond)
	close(done)
	

	select {
	case <-done:
		return
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for first PONG message")
	}

	// Check if there's a second message (there should be exactly one)
	select {
	case extraMsg := <-done:
		// If we get here, we got an extra message, which we expect
		t.Fatalf("Received unexpected extra PONG message: %v", extraMsg)
	case <-time.After(1 * time.Second):
		return
	}
	nodeA.Close()
	nodeB.Close()
}
