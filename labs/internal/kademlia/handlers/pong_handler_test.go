package handlers

import (
	"d7024e/internal/kademlia"
	mock "d7024e/internal/mock"
	net "d7024e/pkg/network"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPongHandler(t *testing.T) {
	network := mock.NewMockNetwork()
	nodeA, err := kademlia.NewKademliaNode(network, net.Address{
		IP:   "127.0.0.1",
		Port: 8001,
	})
	if err != nil {
		t.Fatalf("Failed to create nodeA: %v", err)
	}
	nodeB, err := kademlia.NewKademliaNode(network, net.Address{
		IP:   "172.0.0.1",
		Port: 8002,
	})
	if err != nil {
		t.Fatalf("Failed to create nodeB: %v", err)
	}

	done := make(chan net.Message, 1)

	// Set up handlers
	nodeA.Handle("PING", func(msg *net.Message, node kademlia.IKademliaNode) error {
		// When nodeA receives a PING, it should send a PONG back
		return node.SendPongMessage(msg.From, msg.MessageID)
	})

	nodeB.Handle("PONG", func(msg *net.Message, node kademlia.IKademliaNode) error {
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
