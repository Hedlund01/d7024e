package cli

import (
	"context"
	"d7024e/internal/kademlia"
	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaNetwork "d7024e/internal/kademlia/network"
	"d7024e/pkg/network"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createStartNodeCommand())
}

func createStartNodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new node",
		Long:  "Start a new node in the network",
		Run: func(cmd *cobra.Command, args []string) {
			// Set log level from flag
			logLevelStr, _ := cmd.Flags().GetString("log-level")
			logLevel, err := log.ParseLevel(logLevelStr)
			if err != nil {
				log.Warnf("Invalid log level '%s', defaulting to info", logLevelStr)
				logLevel = log.InfoLevel
			}
			log.SetLevel(logLevel)

			fmt.Println("Starting new node...")
			startNode()
		},
	}

	cmd.Flags().StringP("log-level", "l", "info", "Set the log level (trace, debug, info, warn, error, fatal, panic)")
	return cmd
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

	// Set up message handlers
	node.Handle(kademlia.PING, kademlia.PingHandler)
	node.Handle(kademlia.PONG, kademlia.PongHandler)
	node.Handle(kademlia.FIND_NODE_REQUEST, kademlia.FindNodeRequestHandler)
	node.Handle(kademlia.FIND_VALUE_REQUEST, kademlia.FindValueRequestHandler)
	node.Handle(kademlia.STORE_REQUEST, kademlia.StoreRequestHandler)

	// Start the node
	node.Start()

	// Resolve and connect to other nodes
	addr, error := kademliaNetwork.ResolveTasks("KademliaStack_kademliaNodes", 8000)

	if error != nil {
		log.Debugln("Error resolving tasks: ", error)
	}

	if len(addr) == 0 {
		log.Infoln("No addresses resolved initially. Waiting up to 15 seconds for contacts to appear in routing table...")

		// Wait up to 15 seconds, checking every second if any contacts appear
		timeout := time.After(15 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		contactsFound := false
		for !contactsFound {
			select {
			case <-timeout:
				// Check one final time before giving up
				if node.GetRoutingTable().GetNumberOfConnections() == 0 {
					log.Fatalf("No contacts found in routing table after 15 seconds. Shutting down node.")
					node.Close()
					cancel()
					return
				}
				contactsFound = true

			case <-ticker.C:
				// Check if any contacts have appeared in the routing table
				if node.GetRoutingTable().GetNumberOfConnections() > 0 {
					log.Infoln("Contacts found in routing table, continuing...")
					contactsFound = true
				} else {
					log.Debugln("Still waiting for contacts...")
				}
			}
		}
	}

	if len(addr) > 0 {
		log.Debugln("Resolved addresses: ", addr)
		contact := kademliaContact.NewContact(node.GetRoutingTable().GetMe().ID, addr[0])
		node.Join(&contact)
	}

	// Create and start the interactive shell
	StartInteractiveShell(node, ctx, cancel)
}
