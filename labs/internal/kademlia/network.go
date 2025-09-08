package kademlia

import (
	"context"
	"encoding/json"
	"net"
	"net/netip"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Network struct {
	IP   netip.Addr
	Port int
	RT   *RoutingTable
}

type Message struct {
	Type     string
	Source   netip.AddrPort
	SourceID string
}

func (network *Network) Listen(ctx context.Context) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   network.IP.AsSlice(),
		Port: network.Port,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	log.Infof("UDP server listening on %s:%d", network.IP, network.Port)

	for {
		select {
		case <-ctx.Done():
			log.Infoln("Shutting down listener...")
			return
		default:
			buffer := make([]byte, 1024)
			n, clientAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				log.Errorf("Error reading UDP message: %v", err)
				continue
			}

			// Handle the request in a goroutine
			go handleRequest(network, string(buffer[:n]), clientAddr, conn)
		}
	}
}

func handleRequest(network *Network, message string, clientAddr *net.UDPAddr, conn *net.UDPConn) {
	var msg Message
	json.Unmarshal([]byte(message), &msg)

	// Handle different message types
	switch msg.Type {
	case "PING":
		log.Debugf("Received PING from %s", msg.Source)
		responseMessage := Message{
			Type:     "PONG",
			Source:   netip.AddrPortFrom(network.IP, uint16(network.Port)),
			SourceID: network.RT.me.ID.String(),
		}

		// Convert message to bytes
		responseMessageBytes, err := json.Marshal(responseMessage)
		if err != nil {
			log.Errorf("Failed to marshal response: %v", err)
			return
		}

		// Respond to ping
		_, err = conn.WriteToUDPAddrPort(responseMessageBytes, msg.Source)
		if err != nil {
			log.Errorf("Failed to send PONG: %v", err)
		}

		network.RT.AddContact(NewContact(NewKademliaID(msg.SourceID), msg.Source.String()))

	case "PONG":
		log.WithField("message", msg).Debugf("Received PONG from %s", msg.Source)
		network.RT.AddContact(NewContact(NewKademliaID(msg.SourceID), msg.Source.String()))
	default:
		log.WithField("message", msg).Warnf("Unknown message type from %s", msg.Source)

	}
}

// resolveTasks returns "ip:port" addresses for all replicas (Swarm).
func ResolveTasks(service string, port int) ([]string, error) {
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

func (network *Network) SendInitPing(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
		return err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatalf("Failed to dial UDP: %v", err)
		return err
	}
	defer conn.Close()

	message := Message{
		Type:     "PING",
		Source:   netip.AddrPortFrom(network.IP, uint16(network.Port)),
		SourceID: network.RT.me.ID.String(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
		return err
	}

	// Send PING
	_, err = conn.Write(messageBytes)
	if err != nil {
		log.Fatalf("Failed to send PING: %v", err)
		return err
	}

	return nil
}

func (network *Network) SendPingMessage(contact *Contact) {
	udpAddr, err := net.ResolveUDPAddr("udp", contact.Address)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatalf("Failed to dial UDP: %v", err)
	}
	defer conn.Close()

	message := Message{
		Type:     "PING",
		Source:   netip.AddrPortFrom(network.IP, 8000),
		SourceID: network.RT.me.ID.String(),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
	}
	// Send PING
	_, err = conn.Write(messageBytes)
	if err != nil {
		log.Fatalf("Failed to send PING: %v", err)
	}
}

func (network *Network) SendFindContactMessage(contact *Contact) {
	// TODO
}

func (network *Network) SendFindDataMessage(hash string) {
	// TODO
}

func (network *Network) SendStoreMessage(data []byte) {
	// TODO
}
