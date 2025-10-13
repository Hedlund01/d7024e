package mock

import (
	"d7024e/pkg/network"
	"errors"
	"math/rand/v2"
	"os"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
)

type mockNetwork struct {
	mu         sync.RWMutex
	listeners  map[network.Address]chan network.Message
	partitions map[network.Address]bool // true if the address is partitioned
	dropRate   float64                  // probability of dropping a message
	enableDrop bool                     // whether to enable message dropping
}

func (n *mockNetwork) EnableDropRate(enable bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enableDrop = enable
}

func NewMockNetwork() network.Network {
	dropRate := 0.1
	if val, err := strconv.ParseFloat(os.Getenv("DROP_RATE"), 64); err == nil && val >= 0 {
		dropRate = val
	} else {
		log.WithField("func", "NewMockNetwork").Warnf("DROP_RATE not set or invalid, defaulting to %f", dropRate)
	}
	return &mockNetwork{
		listeners:  make(map[network.Address]chan network.Message),
		partitions: make(map[network.Address]bool),
		dropRate:   dropRate,
		enableDrop: false,
	}
}

func (n *mockNetwork) Listen(addr network.Address) (network.Connection, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.listeners[addr]; exists {
		return nil, errors.New("address already in use")
	}
	ch := make(chan network.Message, 100) // buffered channel
	n.listeners[addr] = ch
	return &mockConnection{addr: addr, network: n, recvCh: ch}, nil
}

func (n *mockNetwork) Dial(addr network.Address) (network.Connection, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if _, exists := n.listeners[addr]; !exists {
		return nil, errors.New("address not found")
	}
	return &mockConnection{addr: addr, network: n}, nil
}

func (n *mockNetwork) Partition(group1, group2 []network.Address) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, addr := range group1 {
		n.partitions[addr] = true
	}
	for _, addr := range group2 {
		n.partitions[addr] = true
	}
}

func (n *mockNetwork) Heal() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.partitions = make(map[network.Address]bool)
}

type mockConnection struct {
	addr    network.Address
	network *mockNetwork
	recvCh  chan network.Message
	mu      sync.RWMutex
	closed  bool
}

func (c *mockConnection) Send(msg *network.Message) error {
	c.network.mu.RLock()

	// Simulate message drop
	if rand.Float64() < c.network.dropRate && c.network.enableDrop {
		c.network.mu.RUnlock()
		return errors.New("message dropped due to network conditions")
	}

	if c.network.partitions[c.addr] || c.network.partitions[msg.To] {
		c.network.mu.RUnlock()
		return errors.New("network partitioned")
	}

	ch, exists := c.network.listeners[msg.To]
	if !exists {
		c.network.mu.RUnlock()
		return errors.New("destination address not found")
	}

	// Keep the lock while sending to prevent the channel from being closed
	select {
	case ch <- *msg:
		c.network.mu.RUnlock()
		return nil
	default:
		c.network.mu.RUnlock()
		return errors.New("message queue full")
	}
}

func (c *mockConnection) Recv() (network.Message, error) {
	c.mu.RLock()
	if c.closed || c.recvCh == nil {
		c.mu.RUnlock()
		return network.Message{}, errors.New("connection not listening")
	}
	ch := c.recvCh
	c.mu.RUnlock()

	msg, ok := <-ch
	if !ok {
		return network.Message{}, errors.New("connection closed")
	}
	return msg, nil
}

func (c *mockConnection) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil // Already closed
	}
	c.closed = true
	c.mu.Unlock()

	c.network.mu.Lock()
	defer c.network.mu.Unlock()

	if c.recvCh != nil {
		close(c.recvCh)
		delete(c.network.listeners, c.addr)
		c.recvCh = nil
	}
	return nil
}
