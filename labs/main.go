// TODO: Add package documentation for `main`, like this:
// Package main something something...
package main

import (
	"context"
	"d7024e/internal/kademlia"
	build "d7024e/pkg/build"
	"net"
	"os"
	"time"

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

	// log.Debugf("Pretending to run the kademlia app...")
	// Using stuff from the kademlia package here. Something like...
	// id := kademlia.NewKademliaID("FFFFFFFF00000000000000000000000000000000")
	// contact := kademlia.NewContact(id, "kademliaNodes-1:8000")
	// log.Debugf("Contact: %s", contact.String())
	// log.Debugf("Contact: %v", contact)

	ctx, cancel := context.WithCancel(context.Background())

	go kademlia.Listen("0.0.0.0", 8000, ctx)

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
		kademlia.SendSimplePing(a)
	}

	time.Sleep(200 * time.Second)
	cancel()

}
