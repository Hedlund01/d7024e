// TODO: Add package documentation for `main`, like this:
// Package main something something...
package main

import (
	"context"
	"crypto/sha1"
	"d7024e/internal/kademlia"
	build "d7024e/pkg/build"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func getLocalIP() (string, error) {
	// 1) Try DNS for container hostname (matches Docker DNS)
	if hn, err := os.Hostname(); err == nil {
		if ips, err := net.LookupHost(hn); err == nil && len(ips) > 0 {
			return ips[0], nil
		}
	}
	return "", nil
}

type DefaultFieldHook struct{}

var localIP = ""

func (hook *DefaultFieldHook) Fire(entry *log.Entry) error {

	entry.Data["containerId"] = localIP
	return nil
}

// Levels returns the log levels for which this hook is triggered.
func (hook *DefaultFieldHook) Levels() []log.Level {
	return log.AllLevels
}

func printRT(rt *kademlia.RoutingTable) {
	log.Debugf("Routing Table:")
	contacts := rt.FindClosestContacts(kademlia.NewKademliaID("2111111400000000000000000000000000000000"), 20)
	for i := range contacts {
		fmt.Println(contacts[i].String())
	}
}

func main() {

	// Set log level
	log.SetLevel(log.DebugLevel)
	log.AddHook(&DefaultFieldHook{})

	localip, err := getLocalIP()
	if err != nil {
		log.Fatalf("Failed to get local IP: %v", err)
	}

	localIP = localip

	// Log buildtime and version
	log.Infof("Build version: %s", build.BuildVersion)
	log.Infof("Build time: %s", build.BuildTime)

	uuid, err := uuid.NewUUID()
	if err != nil {
		log.Fatalf("Failed to create UUID: %v", err)
	}

	hash := sha1.New()
	hash.Write(uuid[:])
	id := kademlia.NewKademliaID(fmt.Sprintf("%x", hash.Sum(nil)))
	contact := kademlia.NewContact(id, fmt.Sprintf("%s:8000", localIP))
	log.Debugf("Contact: %s", contact.String())
	log.Debugf("Contact: %v", contact)
	rt := kademlia.NewRoutingTable(contact)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer printRT(rt)

	ipAddr, err := netip.ParseAddr(localIP)
	if err != nil {
		log.Fatalf("Failed to parse local IP: %v", err)
	}
	network := kademlia.Network{
		IP:   ipAddr,
		Port: 8000,
		RT:   rt,
	}

	go network.Listen(ctx)

	time.Sleep(2 * time.Second) // Wait for all nodes to start listening

	addr, error := kademlia.ResolveTasks("KademliaStack_kademliaNodes", 8000)

	if error != nil {
		log.Debugln("Error resolving tasks: ", error)
	}

	log.Debugln("Resolved addresses: ", addr)

	for _, a := range addr {
		if a == localIP+":8000" {
			continue
		}
		err := network.SendInitPing(a)
		if err != nil {
			log.Debugln("Error sending PING: ", err)
		}

	}

	time.Sleep(10 * time.Second)
	cancel()

}
