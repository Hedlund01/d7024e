package cli

import (
	"context"
	"d7024e/internal/kademlia"
	kademliaNetwork "d7024e/internal/kademlia/network"
	"d7024e/pkg/network"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
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
		Long:  "Start a new node in the network or in isolation mode",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting new node...")
			isolationMode, _ := cmd.Flags().GetBool("isolation")
			startNode(isolationMode)
		},
	}

	cmd.Flags().BoolP("isolation", "i", false, "Start node in isolation mode (skip network discovery and joining)")

	return cmd
}

func startNode(isolationMode bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	kademliaNetwork := kademliaNetwork.NewKademliaNetwork(ctx)

	node, err := kademlia.NewKademliaNode(kademliaNetwork, network.Address{
		IP:   localIP,
		Port: 8000,
	})
	if err != nil {
		log.Fatalf("Failed to create Kademlia node: %v", err)
		return
	}

	node.Start()

	go func() {
		sig := <-sigChan
		log.Infof("Received signal %v, initiating graceful shutdown...", sig)
		cancel()
		shutdownNode(node)
	}()

	node.Handle(kademlia.PING, kademlia.PingHandler)
	node.Handle(kademlia.PONG, kademlia.PongHandler)
	node.Handle(kademlia.FIND_NODE_REQUEST, kademlia.FindNodeRequestHandler)
	node.Handle(kademlia.FIND_VALUE_REQUEST, kademlia.FindValueRequestHandler)
	node.Handle(kademlia.STORE_REQUEST, kademlia.StoreRequestHandler)

	if !isolationMode {
		addr, error := kademliaNetwork.ResolveTasks("KademliaStack_kademliaNodes", 8000)

		if error != nil {
			log.Debugln("Error resolving tasks: ", error)
		}

		if len(addr) == 0 {
			log.Info("No addresses resolved initially. Waiting up to 15 seconds for contacts to appear in routing table...")

			timeout := time.After(15 * time.Second)
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			contactsFound := false
			for !contactsFound {
				select {
				case <-timeout:
					if node.GetRoutingTable().GetNumberOfConnections() == 0 {
						log.Warn("No contacts found in routing table after 15 seconds. Shutting down")
						cancel()
						return
					} else {
						contactsFound = true
					}

				case <-ticker.C:
					if node.GetRoutingTable().GetNumberOfConnections() > 0 {
						log.Info("Contacts found in routing table, continuing...")
						contactsFound = true
					} else {
						continue
					}
				}
			}
		}

		if len(addr) > 0 {
			for _, a := range addr {
				log.Infof("Resolved addresses: %v", addr)
				joinNetwork(node, a)
			}
		}
	} else {
		log.Info("Starting in isolation mode - skipping network discovery and joining")
	}

	log.Info("Starting node operations in background...")

	time.Sleep(1 * time.Second)

	log.Info("Starting interactive shell in main process...")
	StartInteractiveShell(node, ctx, cancel)

	log.Info("Interactive shell exited, initiating shutdown...")

	<-ctx.Done()
	log.Info("Context cancelled, initiating graceful shutdown...")
	shutdownNode(node)
}

func joinNetwork(node kademlia.IKademliaNode, addressToJoin string) {
	addressParts := strings.Split(addressToJoin, ":")
	if len(addressParts) != 2 {
		log.Warnf("Invalid address format: %s, continuing without joining", addressToJoin)
		return
	}

	ip := addressParts[0]
	port, err := strconv.Atoi(addressParts[1])
	if err != nil {
		log.Warnf("Invalid port number: %s, continuing without joining", addressParts[1])
		return
	}

	log.Infof("Joining node at %s:%d", ip, port)

	go node.Join(&network.Address{IP: ip, Port: port})
}

func shutdownNode(node kademlia.IKademliaNode) {
	log.Info("Shutting down Kademlia node...")

	if node != nil {
		if err := node.Close(); err != nil {
			log.Errorf("Error during node shutdown: %v", err)
		} else {
			log.Info("Node shutdown complete")
		}
	}

	log.Info("Application shutting down...")
	os.Exit(0)
}
