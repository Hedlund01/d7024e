package kademlia

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Network struct {
}

func Listen(ip string, port int, ctx context.Context) {

	conn, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	for {
		select {
		case <-ctx.Done():
			log.Infoln("Shutting down listener...")
			return
		default:
			client, err := conn.Accept()

			if err != nil {
				log.Fatalln(err)
			}

			go handleRequest(client)
			
		}

	}

}
func handleRequest(conn net.Conn) string {
	i := 0
	scanner := bufio.NewScanner(conn)
	connectedAddr := conn.RemoteAddr().String()
	log.Debugf("New connection from %s", connectedAddr)
	for scanner.Scan() {
		ln := scanner.Text()
		log.Debugln(ln)
		if ln == "" {
			break
		}
		i++
	}

	return ""
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


// sendSimplePing sends a single "PING" to the given address.
func SendSimplePing(addr string) error {
    tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
    if err != nil {
        return err
    }
    conn, err := net.DialTCP("tcp", nil, tcpAddr)
    if err != nil {
        return err
    }
    defer conn.Close()
    _, err = conn.Write([]byte("PING"))
    return err
}

func (network *Network) SendPingMessage(contact *Contact) {
	// TODO
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
