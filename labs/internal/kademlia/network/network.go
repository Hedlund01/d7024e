package kademliaNetwork

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	network "d7024e/pkg/network"

	log "github.com/sirupsen/logrus"
)

type KademliaNetwork struct {
	// mu      sync.RWMutex
	ctx context.Context
}

type KademliaConnection struct {
	addr    network.Address
	network *KademliaNetwork
	mu      sync.RWMutex
	closed  bool
	conn    *net.UDPConn
}

func NewKademliaNetwork(ctx context.Context) *KademliaNetwork {
	return &KademliaNetwork{
		ctx: ctx,
	}
}

func (network *KademliaNetwork) EnableDropRate(enable bool) {
	log.Warn("EnableDropRate is not implemented in KademliaNetwork")
}

func (network *KademliaNetwork) Listen(addr network.Address) (network.Connection, error) {
	log.Debugf("Listening on %s:%d", addr.IP, addr.Port)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(addr.IP),
		Port: addr.Port,
	})
	if err != nil {
		log.WithField("func", "kademlia/Listen").Fatalln("Failed when starting to listen on UDP address:", err)
		return nil, errors.New("failed to listen on UDP address")
	}

	log.Infof("UDP server listening on %s:%d", addr.IP, addr.Port)

	return &KademliaConnection{
		addr:    addr,
		network: network,
		mu:      sync.RWMutex{},
		closed:  false,
		conn:    conn,
	}, nil
}

func (network *KademliaNetwork) Dial(addr network.Address) (network.Connection, error) {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP(addr.IP),
		Port: addr.Port,
	})
	if err != nil {
		log.Fatalln("Failed to dial up UDP address: ", err)
		return nil, errors.New("failed to dial UDP address")
	}

	return &KademliaConnection{
		addr:    addr,
		network: network,
		mu:      sync.RWMutex{},
		closed:  false,
		conn:    conn,
	}, nil
}

func (kademliaNetwork *KademliaNetwork) Partition(group1, group2 []network.Address) {
	// Implement partitioning logic here
	// TODO
	panic("Partitioning not implemented")
}

func (kademliaNetwork *KademliaNetwork) Heal() {
	// Implement healing logic here
	// TODO
	panic("Healing not implemented")
}

func (kademliaConnection *KademliaConnection) Close() error {
	// kademliaConnection.mu.Lock()
	// defer kademliaConnection.mu.Unlock()

	if kademliaConnection.closed {
		return errors.New("connection already closed")
	}
	kademliaConnection.closed = true

	return kademliaConnection.conn.Close()
}

func (kademliaConnection *KademliaConnection) Send(msg *network.Message) error {
	// kademliaConnection.mu.RLock()
	// defer kademliaConnection.mu.RUnlock()

	if kademliaConnection.closed {
		return errors.New("connection closed")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	log.WithField("msg", msg).WithField("func", "network/send").Debugf("Sending message to %s, with msgID %s", msg.To.String(), msg.MessageID.String())
	_, err = kademliaConnection.conn.Write(data)
	if err != nil {
		log.Errorf("Error sending message: %v", err)
	}
	return err
}

func (kademliaConnection *KademliaConnection) Recv() (network.Message, error) {
	// kademliaConnection.mu.Lock()
	// defer kademliaConnection.mu.Unlock()

	if kademliaConnection.closed {
		return network.Message{}, errors.New("connection closed")
	}

	buffer := make([]byte, 1024)
	n, _, err := kademliaConnection.conn.ReadFromUDP(buffer)
	if err != nil {
		return network.Message{}, err
	}

	var msg network.Message
	err = json.Unmarshal(buffer[:n], &msg)
	if err != nil {
		return network.Message{}, err
	}

	return msg, nil
}

// ResolveTasks returns "ip:port" addresses for all replicas (Swarm).
func (network *KademliaNetwork) ResolveTasks(service string, port int) ([]string, error) {
	hosts, err := net.LookupHost("tasks." + service)
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0, len(hosts))
	for _, h := range hosts {
		addrs = append(addrs, net.JoinHostPort(h, strconv.Itoa(port)))
	}
	return addrs, nil
}

// ResolveService returns "ip:port" address for a single Docker service/container.
func (network *KademliaNetwork) ResolveService(service string, port int) (string, error) {
	hosts, err := net.LookupHost(service)
	if err != nil {
		return "", err
	}

	if len(hosts) == 0 {
		return "", fmt.Errorf("no hosts found for service %s", service)
	}

	// Return the first resolved IP
	return net.JoinHostPort(hosts[0], strconv.Itoa(port)), nil
}

// func (kademliaConnection *KademliaConnection) SendPingMessage(contact *Contact) {

// }

// func (network *Network) SendFindContactMessage(contact *Contact) {
// 	// TODO
// }

// func (network *Network) SendFindDataMessage(hash string) {
// 	// TODO
// }

// func (network *Network) SendStoreMessage(data []byte) {
// 	// TODO
// }
