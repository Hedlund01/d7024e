package node

import (
	"d7024e/pkg/network"
)

// MessageHandler is a function that processes incoming messages
// The second parameter is now an interface to avoid circular dependencies
type MessageHandler func(msg network.Message, node INode) error

// INode defines the interface that all node types must implement
type INode interface {
	// Core networking methods
	Send(to network.Address, msgType string, data []byte) error
	SendString(to network.Address, msgType, data string) error
	Address() network.Address

	// Message handling
	Handle(msgType string, handler MessageHandler)
	Start()
	Close() error

	// Extension point for implementation-specific data access
	// This allows different node types to provide their own data
	GetNodeData() interface{}
}

// BaseNode provides common functionality that can be embedded
type BaseNode struct {
	address    network.Address
	network    network.Network
	connection network.Connection
	handlers   map[string]MessageHandler
	closed     bool
}

// NewBaseNode creates the common parts of a node
func NewBaseNode(network network.Network, addr network.Address) (*BaseNode, error) {
	conn, err := network.Listen(addr)
	if err != nil {
		return nil, err
	}

	return &BaseNode{
		address:    addr,
		network:    network,
		connection: conn,
		handlers:   make(map[string]MessageHandler),
		closed:     false,
	}, nil
}

// Address returns the node's address
func (b *BaseNode) Address() network.Address {
	return b.address
}

// Handle registers a message handler for a specific message type
func (b *BaseNode) Handle(msgType string, handler MessageHandler) {
	b.handlers[msgType] = handler
}

// Send sends a message to the target address
func (b *BaseNode) Send(to network.Address, msgType string, data []byte) error {
	connection, err := b.network.Dial(to)
	if err != nil {
		return err
	}
	defer connection.Close()

	msg := network.Message{
		From:        b.address,
		To:          to,
		Payload:     data,
		PayloadType: msgType,
	}

	return connection.Send(msg)
}

// SendString is a convenience method for sending string messages
func (b *BaseNode) SendString(to network.Address, msgType, data string) error {
	return b.Send(to, msgType, []byte(data))
}

// Close shuts down the base node
func (b *BaseNode) Close() error {
	b.closed = true
	return b.connection.Close()
}

// GetConnection returns the connection for use by concrete implementations
func (b *BaseNode) GetConnection() network.Connection {
	return b.connection
}

// GetHandlers returns the handlers map for use by concrete implementations
func (b *BaseNode) GetHandlers() map[string]MessageHandler {
	return b.handlers
}

// IsClosed returns whether the node is closed
func (b *BaseNode) IsClosed() bool {
	return b.closed
}

func (b *BaseNode) UpdateClosed(closed bool) {
	b.closed = closed
}
