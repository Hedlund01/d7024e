package kademlia

import (
	"testing"
	"time"

	"d7024e/internal/mock"

	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"

	net "d7024e/pkg/network"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
)

func Init() {
	log.SetLevel(log.DebugLevel)
}

func TestRPCValiadtion(t *testing.T) {
	log.WithField("func", "TestRPCValidation").Info("Starting TestRPCValidation")
	network := mock.NewMockNetwork()
	alice, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 8080}, *kademliaID.NewRandomKademliaID())
	bob, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 8081}, *kademliaID.NewRandomKademliaID())

	done := make(chan *net.Message, 2)

	alice.Handle(PING, func(msg *net.Message, node IKademliaNode) error {
		t.Logf("Alice gets: %s\n", msg.Payload)
		return node.send(msg.From, PONG, []byte("REAL PONG"), msg.MessageID, true)
	})

	bob.Handle(PONG, func(msg *net.Message, node IKademliaNode) error {
		t.Logf("Bob gets: %s\n", msg.Payload)
		done <- msg
		return nil
	})

	alice.Start()
	bob.Start()

	bob.SendPingMessage(alice.Address())

	alice.SendPongMessage(bob.Address(), kademliaID.NewRandomKademliaID())

	// Extract all messages from done channel may be two or one
	messagesReceived := 0
	for i := 0; i < 2; i++ {
		select {
		case msg := <-done:
			log.WithField("payload", string(msg.Payload)).Debug("Extracted message from done channel")
			messagesReceived++
		case <-time.After(1000 * time.Millisecond):
			// Timeout waiting for message, break out of loop
			return
		}
	}

	alice.Close()
	bob.Close()

	assert.Equal(t, 1, messagesReceived, "Expected exactly one message to be received")
}

func TestIterativeNodeLookup(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	log.WithField("func", "TestSimpleIterativeNodeLookup").Info("Starting TestSimpleIterativeNodeLookup")
	network := mock.NewMockNetwork()

	n := 5 // Change size of network here
	nodes := make([]IKademliaNode, n)
	for i := 0; i < n; i++ {
		port := 8081 + i
		node, err := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: port}, *kademliaID.NewRandomKademliaID())
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
	}

	for _, node := range nodes {
		node.Start()
	}

	for node := range nodes {
		nodes[node].Handle(PING, PingHandler)
		nodes[node].Handle(PONG, PongHandler)
		nodes[node].Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	}

	lastNode := nodes[n-1]
	rootNode := nodes[0]

	for i, node := range nodes {
		for j := 0; j < n; j++ {
			if i != j {
				node.SendPingMessage(nodes[j].Address())
			}
		}
	}

	time.Sleep(1 * time.Second)

	newNode, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 9000}, *kademliaID.NewRandomKademliaID())
	newNode.Start()
	newNode.Handle(PING, PingHandler)
	newNode.Handle(PONG, PongHandler)
	newNode.Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)

	time.Sleep(1 * time.Second)

	newNode.SendPingMessage(rootNode.Address())

	time.Sleep(1 * time.Second)

	result := newNode.FindNode(lastNode.GetRoutingTable().GetMe().ID)

	assert.NotNil(t, result, "LookupContact returned nil")
	t.Logf("Found node: %s at %s\n", result.ID.String(), result.Address)
	t.Logf("Last node:  %s at %s\n", lastNode.GetRoutingTable().GetMe().ID.String(), lastNode.Address().String())
	assert.True(t, lastNode.GetRoutingTable().me.ID.Equals(result.ID), "Expected to find the last node in the network")

	for _, node := range nodes {
		node.Close()
	}
	newNode.Close()
}

func TestNodeJoin(t *testing.T) {
	log.WithField("func", "TestNodeJoin").Info("Starting TestNodeJoin")
	network := mock.NewMockNetwork()

	n := 100
	nodes := make([]IKademliaNode, n)
	for i := range nodes {
		port := 8081 + i
		node, err := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: port}, *kademliaID.NewRandomKademliaID())
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
	}

	for _, node := range nodes {
		node.Start()
	}

	// Register Ping, Pong, FindNode handlers for root node
	for node := range nodes {
		nodes[node].Handle(PING, PingHandler)
		nodes[node].Handle(PONG, PongHandler)
		nodes[node].Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	}

	rootNode := nodes[0]

	for i, node := range nodes {
		for j := range nodes {
			if i != j {
				node.SendPingMessage(nodes[j].Address())
			}
		}
	}

	time.Sleep(1 * time.Second)

	newNode, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 9000}, *kademliaID.NewRandomKademliaID())
	newNode.Start()
	newNode.Handle(PING, PingHandler)
	newNode.Handle(PONG, PongHandler)
	newNode.Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	rootAddr := rootNode.Address()
	newNode.Join(&rootAddr, rootNode.GetRoutingTable().GetMe().ID)

	contactCount := 0
	for bucket := range newNode.GetRoutingTable().buckets {
		contactCount += newNode.GetRoutingTable().buckets[bucket].Len()
	}
	println("New node contact count:", contactCount)

	assert.Greater(t, contactCount, 3, "Expected new node to have contacts in its routing table after joining")

	// Stop all nodes
	for _, node := range nodes {
		node.Close()
	}
	newNode.Close()
}

func TestGetAndStoreValue(t *testing.T) {
	log.WithField("func", "TestGetAndStoreValue").Info("Starting TestGetAndStoreValue")
	mockNetwork := mock.NewMockNetwork()
	node, err := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: 8081}, *kademliaID.NewRandomKademliaID())
	assert.NoErrorf(t, err, "Failed to create node: %v", err)

	// Start the node
	node.Start()
	<-time.After(1 * time.Second)

	// Test storing a value
	value := []byte("1234567890abcdef1234567890abcdef12345678")
	key := kademliaID.NewKademliaID(string(value)) // Must be greater than 20 bytes

	err = node.StoreValue(value, key)
	assert.NoErrorf(t, err, "Failed to store value: %v", err)

	// Test retrieving the value
	retrievedValue, err := node.GetValue(key)
	assert.NoErrorf(t, err, "Failed to get value: %v", err)
	assert.Equalf(t, value, retrievedValue, "Expected retrieved value to match")

	// Stop the node
	node.Close()
}

func TestStore(t *testing.T) {
	log.WithField("func", "TestStore").Info("Starting TestStore")
	mockNetwork := mock.NewMockNetwork()

	n := 20 // Change size of network here
	nodes := make([]IKademliaNode, n)
	for i := 0; i < n; i++ {
		port := 8081 + i
		node, err := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: port}, *kademliaID.NewRandomKademliaID())
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
	}

	for _, node := range nodes {
		node.Start()
	}

	time.Sleep(1 * time.Second) // Give nodes time to start
	value := []byte("1234567890abcdef1234567890abcdef12345678")
	for node := range nodes {
		nodes[node].Handle(PING, PingHandler)
		nodes[node].Handle(PONG, PongHandler)
		nodes[node].Handle(FIND_VALUE_REQUEST, FindValueRequestHandler)
		nodes[node].Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
		nodes[node].Handle(STORE_REQUEST, StoreRequestHandler)
	}

	// Have each node join the network via pinging each other
	for i, node := range nodes {
		for j := range nodes {
			if i != j {
				node.SendPingMessage(nodes[j].Address())
			}
		}
	}

	time.Sleep(1 * time.Second)

	rootNode := nodes[0]

	value = []byte("1234567890abcdef1234567890abcdef12345678")
	key := kademliaID.NewKademliaID(string(value)) // Must be greater than 20 bytes
	log.WithField("func", "TestStore").WithField("value", string(value)).WithField("key", key.String()).Info("Storing value")

	err := rootNode.Store(value)
	assert.NoErrorf(t, err, "Failed to store value: %v", err)

	// Retrieve the value
	var foundValue []byte
	for i, node := range nodes {
		retrievedValue, err := node.GetValue(key)
		log.WithField("func", "TestStore").WithField("nodeIndex", i).WithField("err", err).Debugf("Checking node %d for stored value", i)
		if err == nil && retrievedValue != nil {
			foundValue = retrievedValue
			break
		}
	}

	assert.NotNil(t, foundValue, "Expected at least one node to store the value")
	assert.Equalf(t, value, foundValue, "Expected found value to match")

	// Stop all nodes
	for _, node := range nodes {
		node.Close()
	}
}

func TestFindValue(t *testing.T) {
	log.WithField("func", "TestFindValue").Info("Starting TestFindValue")
	mockNetwork := mock.NewMockNetwork()

	n := 5
	nodes := make([]IKademliaNode, n)
	for i := range nodes {
		port := 8081 + i
		node, err := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: port}, *kademliaID.NewRandomKademliaID())
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
	}

	// Start all nodes
	for _, node := range nodes {
		node.Start()
	}

	time.Sleep(1 * time.Second) // Give nodes time to start

	// Register Ping, Pong, FindNode handlers for root node
	for node := range nodes {
		nodes[node].Handle(PING, PingHandler)
		nodes[node].Handle(PONG, PongHandler)
		nodes[node].Handle(FIND_VALUE_REQUEST, FindValueRequestHandler)
		nodes[node].Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
		nodes[node].Handle(STORE_REQUEST, StoreRequestHandler)
	}

	// Have each node join the network via pinging each other
	for i, node := range nodes {
		for j := range nodes {
			if i != j {
				node.SendPingMessage(nodes[j].Address())
			}
		}
	}

	time.Sleep(2 * time.Second)

	rootNode := nodes[0]

	value := []byte("1234567890abcdef1234567890abcdef12345678")
	key := kademliaID.NewKademliaID(string(value)) // Must be greater than 20 bytes
	log.WithField("func", "TestFindValue").WithField("value", string(value)).WithField("key", key.String()).Debug("Storing value")

	err := rootNode.Store(value)
	assert.NoErrorf(t, err, "Failed to store value: %v", err)

	time.Sleep(1 * time.Second) // Give time for value to propagate

	newNode, _ := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: 9000}, *kademliaID.NewRandomKademliaID())
	newNode.Start()
	newNode.Handle(PING, PingHandler)
	newNode.Handle(PONG, PongHandler)
	newNode.Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	contact := kademliaContact.NewContact(rootNode.GetRoutingTable().GetMe().ID, rootNode.Address().String())
	newNode.GetRoutingTable().AddContact(contact)
	val, err := newNode.FindValue(key)
	assert.NoErrorf(t, err, "Failed to find value: %v", err)
	assert.NotNil(t, val, "Expected to find the stored value")
	assert.Equalf(t, value, val, "Expected found value to match")

	newNode.Close()
	// Stop all nodes
	for _, node := range nodes {
		node.Close()
	}
}

func TestJoinAndFindAll(t *testing.T) {
	log.WithField("func", "TestNodeJoin").Info("Starting TestNodeJoin")
	network := mock.NewMockNetwork()

	n := 100
	nodes := make([]IKademliaNode, n)
	for i := range nodes {
		port := 8081 + i
		node, err := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: port}, *kademliaID.NewRandomKademliaID())
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
	}

	for _, node := range nodes {
		if node != nil { // Add nil check
			node.Start()
		}
	}

	// Register Ping, Pong, FindNode handlers for root node
	for node := range nodes {
		nodes[node].Handle(PING, PingHandler)
		nodes[node].Handle(PONG, PongHandler)
		nodes[node].Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	}

	rootNode := nodes[0]

	for i, node := range nodes {
		if i != 0 {
			addr := rootNode.Address()
			node.Join(&addr, rootNode.GetRoutingTable().GetMe().ID)
		}
	}

	time.Sleep(1 * time.Second)

	newNode, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 9000}, *kademliaID.NewRandomKademliaID())
	newNode.Start()
	newNode.Handle(PING, PingHandler)
	newNode.Handle(PONG, PongHandler)
	newNode.Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	lastNode := nodes[n-1]
	lastNodeAddr := lastNode.Address()
	newNode.Join(&lastNodeAddr, lastNode.GetRoutingTable().GetMe().ID)

	time.Sleep(1 * time.Second)

	contactCount := 0
	for bucket := range newNode.GetRoutingTable().buckets {
		contactCount += newNode.GetRoutingTable().buckets[bucket].Len()
	}
	println("New node contact count:", contactCount)

	t.Logf("New node has %d contacts in its routing table after joining", contactCount)

	rootRoutingTableCount := 0
	for bucket := range rootNode.GetRoutingTable().buckets {
		rootRoutingTableCount += rootNode.GetRoutingTable().buckets[bucket].Len()
	}
	t.Logf("Root node has %d contacts in its routing table", rootRoutingTableCount)

	assert.Greater(t, contactCount, 3, "Expected new node to have contacts in its routing table after joining")

	// Check for disconnected groupings in the network
	visited := make(map[string]bool)
	queue := []IKademliaNode{rootNode}
	visited[rootNode.GetRoutingTable().GetMe().ID.String()] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		var contacts []kademliaContact.Contact
		for bucket := range current.GetRoutingTable().buckets {
			contacts = append(contacts, current.GetRoutingTable().buckets[bucket].GetContacts()...)
		}
		for _, contact := range contacts {
			if !visited[contact.ID.String()] {
				visited[contact.ID.String()] = true

				for _, node := range nodes {
					if node != nil && node.GetRoutingTable().GetMe().ID.String() == contact.ID.String() {
						queue = append(queue, node)
						break
					}
				}
			}
		}

	}

	if len(visited) < len(nodes)+1 {
		t.Fatalf("Network partition detected: only %d/%d nodes are connected", len(visited), len(nodes))
	} else if len(visited) == len(nodes)+1 {
		t.Logf("All nodes are connected in a single grouping")
	} else {
		t.Fatalf("Unexpected number of visited nodes: %d", len(visited))
	}

	// Stop all nodes
	for _, node := range nodes {
		if node != nil { // Add nil check
			node.Close()
		}
	}
	newNode.Close()

}
