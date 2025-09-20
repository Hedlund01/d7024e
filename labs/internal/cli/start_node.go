package cli

import (
	"context"
	"d7024e/internal/kademlia"
	kademliaNetwork "d7024e/internal/kademlia/network"
	"d7024e/pkg/network"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(StartNodeCmd)
}

var StartNodeCmd = &cobra.Command{
	Use:   "start-node",
	Short: "Start a new node",
	Long:  "Start a new node in the network",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting new node...")
		startNode()
	},
}

func startNode() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kademliaNetwork := kademliaNetwork.NewKademliaNetwork(ctx)

	node, err := kademlia.NewKademliaNode(kademliaNetwork, network.Address{
		IP:   localIP,
		Port: 8000,
	})
	if err != nil {
		log.Fatalf("Failed to create Kademlia node: %v", err)
		return
	}

	node.Handle("PING", kademlia.PingHandler)
	node.Handle("PONG", kademlia.PongHandler)
	node.Start()

	addr, error := kademliaNetwork.ResolveTasks("KademliaStack_kademliaNodes", 8000)

	if error != nil {
		log.Debugln("Error resolving tasks: ", error)
	}

	log.Debugln("Resolved addresses: ", addr)

	for _, a := range addr {
		if a == localIP+":8000" {
			continue
		}
		node.SendPingMessage(network.Address{
			IP:   strings.Split(a, ":")[0],
			Port: 8000,
		})
	}

	time.Sleep(200 * time.Second)
	cancel()
}
