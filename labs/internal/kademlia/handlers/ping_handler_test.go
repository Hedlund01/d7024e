package handlers

import (
	mock "d7024e/internal/mock"
	net "d7024e/pkg/network"
	"d7024e/pkg/node"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPingHandler(t *testing.T) {
	network := mock.NewMockNetwork()
	nodeA, err := mock.NewMockNode(network, net.Address{
		IP:   "127.0.0.1",
		Port: 8001,
	})
	if err != nil {
		t.Fatalf("Failed to create nodeA: %v", err)
	}
	nodeB, err := mock.NewMockNode(network, net.Address{
		IP:   "172.0.0.1",
		Port: 8002,
	})
	if err != nil {
		t.Fatalf("Failed to create nodeB: %v", err)
	}

	pongChannel := make(chan net.Message, 1)

	nodeA.Handle("PING", PingHandler)

	nodeB.Handle("PONG", func(msg net.Message, node node.INode) error {
		t.Log("Received PONG")
		pongChannel <- msg
		return nil
	})

	nodeA.Start()
	nodeB.Start()

	nodeB.SendString(nodeA.Address(), "PING", "")

	msg := <-pongChannel
	assert.Equal(t, "PONG", msg.PayloadType)

}
