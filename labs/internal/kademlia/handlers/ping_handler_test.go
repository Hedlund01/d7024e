package handlers

import (
	"d7024e/internal/kademlia"
	mock "d7024e/internal/mock"
	net "d7024e/pkg/network"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPingHandler(t *testing.T) {
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

	pongChannel := make(chan net.Message, 1)

	nodeA.Handle("PING", PingHandler)

	nodeB.Handle("PONG", func(msg *net.Message, node kademlia.IKademliaNode) error {
		t.Log("Received PONG")
		pongChannel <- *msg
		return nil
	})

	nodeA.Start()
	nodeB.Start()

	nodeB.SendPingMessage(nodeA.Address())

	msg := <-pongChannel

	nodeA.Close()
	nodeB.Close()

	
	assert.Equal(t, "PONG", msg.PayloadType)
	assert.Equal(t, nodeA.Address(), msg.From)
	assert.Equal(t, nodeB.Address(), msg.To)
}
