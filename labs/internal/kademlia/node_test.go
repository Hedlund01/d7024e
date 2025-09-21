package kademlia

import (
	"d7024e/internal/mock"
	"testing"
	"time"

	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"

	net "d7024e/pkg/network"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
)

func Init(){
	log.SetLevel(log.DebugLevel)
}

func TestRPCValiadtion(t *testing.T) {
	log.WithField("func", "TestRPCValidation").Info("Starting TestRPCValidation")
	network := mock.NewMockNetwork()
	alice, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 8080})
	bob, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 8081})

	done := make(chan *net.Message, 2)

	alice.Handle(PING, func(msg *net.Message, node IKademliaNode) error {
		t.Logf("Alice gets: %s\n", msg.Payload)
		return node.send(msg.From, PONG, []byte("REAL PONG"), msg.MessageID)
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

func TestSimpleIterativeNodeLookup(t *testing.T) {
	log.WithField("func", "TestSimpleIterativeNodeLookup").Info("Starting TestSimpleIterativeNodeLookup")
	network := mock.NewMockNetwork()
	nodeA, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 8080})
	nodeB, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 8081})
	nodeC, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 8082})

	nodeA.Handle(PING, func(msg *net.Message, node IKademliaNode) error {
		return node.SendPongMessage(msg.From, msg.MessageID)
	})

	nodeA.Handle(PONG, func(msg *net.Message, node IKademliaNode) error {
		return nil
	})

	nodeB.Handle(PING, func(msg *net.Message, node IKademliaNode) error {
		return node.SendPongMessage(msg.From, msg.MessageID)
	})

	nodeB.Handle(PONG, func(msg *net.Message, node IKademliaNode) error {
		return nil
	})

	nodeC.Handle(PING, func(msg *net.Message, node IKademliaNode) error {
		return node.SendPongMessage(msg.From, msg.MessageID)
	})

	id := kademliaID.NewRandomKademliaID()

	nodeB.Handle(FIND_NODE_REQUEST, func(msg *net.Message, node IKademliaNode) error {
		t.Logf("Node B received FIND_NODE_REQUEST for ID: %s", id)
		contacts := node.GetRoutingTable().FindClosestContacts(id, 3)
		err := nodeB.SendFindNodeResponse(msg.From, contacts, msg.MessageID)
		if err != nil {
			t.Logf("Error sending FIND_NODE_RESPONSE from Node B: %v", err)
			return nil
		}
		t.Logf("Node B sent FIND_NODE_RESPONSE with contacts: %v", contacts)

		return nil
	})

	nodeA.Start()
	nodeB.Start()
	nodeC.Start()

	time.Sleep(1 * time.Second)

	nodeA.SendPingMessage(nodeB.Address())
	nodeB.SendPingMessage(nodeC.Address())

	time.Sleep(1 * time.Second)

	nodeAContactReturned := nodeA.LookupContact(nodeC.GetRoutingTable().me.ID)

	t.Logf("Node A looked up contact: %v", nodeAContactReturned)

	closestContact := nodeAContactReturned.GetClosestContact()

	assert.NotNil(t, nodeAContactReturned, "Expected contact to be found")
	assert.Equal(t, nodeC.address.Port, closestContact.GetNetworkAddress().Port, "Expected found contact port to match")
	assert.Equal(t, nodeC.GetRoutingTable().me.ID.String(), closestContact.ID.String(), "Expected found contact ID to match")

	nodeA.Close()
	nodeB.Close()
	nodeC.Close()
}

func TestNodeJoin(t *testing.T) {
	log.WithField("func", "TestNodeJoin").Info("Starting TestNodeJoin")
	network := mock.NewMockNetwork()

	// Create n new nodes and have them join the network via the root node
	n := 5
	nodes := make([]IKademliaNode, n)
	for i := 0; i < n; i++ {
		port := 8081 + i
		node, err := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: port})
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
	}

	// Start all nodes
	for _, node := range nodes {
		node.Start()
	}

	// Register Ping, Pong, FindNode handlers for root node
	for node := range nodes {
		nodes[node].Handle(PING, PingHandler)
		nodes[node].Handle(PONG, PongHandler)
		nodes[node].Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	}

	// Have each node join the network via the root node
	rootNode := nodes[0]
	for i, node := range nodes {
		for j := 0; j < n; j++ {
			if i != j {
				node.SendPingMessage(nodes[j].Address())
			}
		}
	}

	time.Sleep(1 * time.Second)

	newNode, _ := NewKademliaNode(network, net.Address{IP: "127.0.0.1", Port: 9000})
	newNode.Start()
	newNode.Handle(PING, PingHandler)
	newNode.Handle(PONG, PongHandler)
	newNode.Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	contact := kademliaContact.NewContact(rootNode.GetRoutingTable().GetMe().ID, rootNode.Address().String())
	newNode.Join(&contact)

	contactCount := 0
	for bucket := range newNode.GetRoutingTable().buckets {
		contactCount += newNode.GetRoutingTable().buckets[bucket].Len()
	}

	assert.Greater(t, contactCount, 3, "Expected new node to have contacts in its routing table after joining")

	// Stop all nodes
	for _, node := range nodes {
		node.Close()
	}
	newNode.Close()
}

func TestFindNode_KnowsTheNode(t *testing.T) {
	log.WithField("func", "TestFindNode_KnowsTheNode").Info("Starting TestFindNode_KnowsTheNode")
	mockNetwork := mock.NewMockNetwork()

	n := 2
	nodes := make([]IKademliaNode, n)
	for i := 0; i < n; i++ {
		port := 8081 + i
		node, err := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: port})
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		t.Logf("Created node %d: %v", i, node.GetRoutingTable().me.ID.String())
	}

	// Start all nodes
	for _, node := range nodes {
		node.Start()
	}

	<-time.After(1 * time.Second) // Give nodes time to start

	// Register Ping, Pong, FindNode handlers for root node
	for node := range nodes {
		nodes[node].Handle(PING, PingHandler)
		nodes[node].Handle(PONG, PongHandler)
		nodes[node].Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	}

	// Have each node join the network via the root node
	rootNode := nodes[0]
	for i, node := range nodes {
		for j := 0; j < n; j++ {
			if i != j {
				node.SendPingMessage(nodes[j].Address())
			}
		}
	}

	time.Sleep(5 * time.Second)

	testNode := nodes[1]

	lookup, err := rootNode.FindNode(testNode.GetRoutingTable().GetMe().ID)
	assert.NoError(t, err)
	assert.NotNil(t, lookup, "Expected found contact to be non-nil")

	assert.Equal(t, testNode.GetRoutingTable().GetMe().ID.String(), lookup.ID.String(), "Expected found contact ID to match")

	// Stop all nodes
	for _, node := range nodes {
		node.Close()
	}

}

func TestFindNode_DontKnowTheNode(t *testing.T) {
	log.WithField("func", "TestFindNode_DontKnowTheNode").Info("Starting TestFindNode_DontKnowTheNode")
	mockNetwork := mock.NewMockNetwork()

	n := 5
	nodes := make([]IKademliaNode, n)
	for i := 0; i < n; i++ {
		port := 8081 + i
		node, err := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: port})
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		t.Logf("Created node %d: %v", i, node.GetRoutingTable().me.ID.String())
	}

	// Start all nodes
	for _, node := range nodes {
		node.Start()
	}

	<-time.After(1 * time.Second) // Give nodes time to start

	// Register Ping, Pong, FindNode handlers for root node
	for node := range nodes {
		nodes[node].Handle(PING, PingHandler)
		nodes[node].Handle(PONG, PongHandler)
		nodes[node].Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)
	}

	// Have each node join the network via pinging each other except for the last which only pings all except one and itself
	for i, node := range nodes {
		for j := 0; j < n; j++ {
			if i != j {
				node.SendPingMessage(nodes[j].Address())
			}
		}
	}

	<-time.After(3 * time.Second)

	testNode, _ := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: 9083})
	testNode.Start()
	<-time.After(1 * time.Second)

	testNode.Handle(PING, PingHandler)
	testNode.Handle(PONG, PongHandler)
	testNode.Handle(FIND_NODE_REQUEST, FindNodeRequestHandler)

	for i, node := range nodes {
		if i != len(nodes)-1 {
			testNode.SendPingMessage(node.Address())
		}
	}

	<-time.After(5 * time.Second)

	actualLookedUpNode := nodes[len(nodes)-1]

	lookedUpNode, err := testNode.FindNode(actualLookedUpNode.GetRoutingTable().GetMe().ID)
	assert.NoError(t, err)
	assert.NotNil(t, lookedUpNode, "Expected found contact to be non-nil")

	assert.Equalf(t, actualLookedUpNode.GetRoutingTable().GetMe().ID.String(), lookedUpNode.ID.String(), "Expected found contact ID to match")

	time.Sleep(2 * time.Second)

	// Stop all nodes
	for _, node := range nodes {
		node.Close()
	}
	testNode.Close()

}

func TestGetAndStoreValue(t *testing.T) {
	log.WithField("func", "TestGetAndStoreValue").Info("Starting TestGetAndStoreValue")
	mockNetwork := mock.NewMockNetwork()
	node, err := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: 8081})
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

	n := 5
	nodes := make([]IKademliaNode, n)
	for i := 0; i < n; i++ {
		port := 8081 + i
		node, err := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: port})
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		t.Logf("Created node %d: %v", i, node.GetRoutingTable().me.ID.String())
	}

	// Start all nodes
	for _, node := range nodes {
		node.Start()
	}

	<-time.After(1 * time.Second) // Give nodes time to start

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
		for j := 0; j < n; j++ {
			if i != j {
				node.SendPingMessage(nodes[j].Address())
			}
		}
	}

	<-time.After(3 * time.Second)

	rootNode := nodes[0]

	value := []byte("1234567890abcdef1234567890abcdef12345678")
	key := kademliaID.NewKademliaID(string(value)) // Must be greater than 20 bytes
	log.WithField("func", "TestStore").WithField("value", string(value)).WithField("key", key.String()).Debug("Storing value")

	err := rootNode.Store(value)
	assert.NoErrorf(t, err, "Failed to store value: %v", err)

	<-time.After(3 * time.Second)

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
	log.SetLevel(log.DebugLevel)
	mockNetwork := mock.NewMockNetwork()

	n := 5
	nodes := make([]IKademliaNode, n)
	for i := 0; i < n; i++ {
		port := 8081 + i
		node, err := NewKademliaNode(mockNetwork, net.Address{IP: "127.0.0.1", Port: port})
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		nodes[i] = node
		t.Logf("Created node %d: %v", i, node.GetRoutingTable().me.ID.String())
	}

	// Start all nodes
	for _, node := range nodes {
		node.Start()
	}

	<-time.After(1 * time.Second) // Give nodes time to start

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
		for j := 0; j < n; j++ {
			if i != j {
				node.SendPingMessage(nodes[j].Address())
			}
		}
	}

	<-time.After(3 * time.Second)

	rootNode := nodes[0]

	value := []byte("1234567890abcdef1234567890abcdef12345678")
	key := kademliaID.NewKademliaID(string(value)) // Must be greater than 20 bytes
	log.WithField("func", "TestFindValue").WithField("value", string(value)).WithField("key", key.String()).Debug("Storing value")

	err := rootNode.Store(value)
	assert.NoErrorf(t, err, "Failed to store value: %v", err)

	<-time.After(3 * time.Second)

	// Retrieve the value

	val, err := rootNode.FindValue(key)
	assert.NoErrorf(t, err, "Failed to find value: %v", err)
	assert.NotNil(t, val, "Expected to find the stored value")
	assert.Equalf(t, value, val, "Expected found value to match")

	// Stop all nodes
	for _, node := range nodes {
		node.Close()
	}
}
