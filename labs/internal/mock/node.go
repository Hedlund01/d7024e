package mock

import (
	"d7024e/pkg/network"
	"fmt"
	"log"
	"d7024e/pkg/node"
	"sync"
)

// MockNode provides a unified abstraction for both sending and receiving messages
type MockNode struct {
	*node.BaseNode
	closeMu   sync.RWMutex
	mu        sync.RWMutex
}



// NewMockNode creates a new node that can both send and receive messages
func NewMockNode(network network.Network, addr network.Address) (*MockNode, error) {
	baseNode, err := node.NewBaseNode(network, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %v", err)
	}

	return &MockNode{
		BaseNode: baseNode,
	}, nil
}

// Start begins listening for incoming messages
func (n *MockNode) Start() {
	go func() {
		for {
			n.closeMu.RLock()
			if n.BaseNode.IsClosed() {
				n.closeMu.RUnlock()
				return
			}
			n.closeMu.RUnlock()

			msg, err := n.BaseNode.GetConnection().Recv()
			if err != nil {
				n.closeMu.RLock()
				if !n.BaseNode.IsClosed() {
					log.Printf("Node %s failed to receive message: %v", n.BaseNode.Address().String(), err)
				}
				n.closeMu.RUnlock()
				return
			}

			n.mu.RLock()
			handler, exists := n.GetHandlers()[msg.PayloadType]
			if !exists {
				handler, exists = n.GetHandlers()["default"]
			}
			n.mu.RUnlock()

			if exists && handler != nil {
				if err := handler(msg, n); err != nil {
					log.Printf("Handler error: %v", err)
				}
			}
		}
	}()
}


// GetNodeData returns mock-specific data (empty for mock nodes)
func (n *MockNode) GetNodeData() interface{} {
	return nil
}

// Close shuts down the node
func (n *MockNode) Close() error {
	n.closeMu.Lock()
	n.UpdateClosed(true)
	n.closeMu.Unlock()
	return n.BaseNode.Close()
}


