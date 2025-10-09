package cli

import (
	"context"
	"d7024e/internal/kademlia"
	kademliaID "d7024e/internal/kademlia/id"
	kademliaNetwork "d7024e/internal/kademlia/network"
	"d7024e/pkg/network"
	"fmt"
	"math/rand"
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
			bootstrap, _ := cmd.Flags().GetString("bootstrap")
			joinID, _ := cmd.Flags().GetString("join-id")

			if bootstrap == "" {
				randDelay := 5 //default to 5 seconds
				if os.Getenv("RAND_DELAY") != "" {
					randDelay, _ = strconv.Atoi(os.Getenv("RAND_DELAY"))
				}
				//Add random delay
				if randDelay > 0 {
					delay := time.Duration(rand.Intn(randDelay)) * time.Second
					log.Infof("Adding random delay of %s before starting node...", delay)
					time.Sleep(delay)
				}
			}

			startNode(bootstrap, joinID)
		},
	}

	cmd.Flags().StringP("bootstrap", "", "", "Start node as a bootstrap Kademlia node, value should be ID to be encrypted to kademliaID")
	cmd.Flags().StringP("join-id", "", "", "Kademlia ID of an existing node to join the network through")
	cmd.MarkFlagsMutuallyExclusive("bootstrap", "join-id")

	return cmd
}

func startNode(bootstrap string, joinID string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	kademliaNetwork := kademliaNetwork.NewKademliaNetwork(ctx)

	var kademliaId *kademliaID.KademliaID
	if bootstrap != "" {
		kademliaId = kademliaID.NewKademliaID(bootstrap)
	} else {
		kademliaId = kademliaID.NewRandomKademliaID()
	}

	node, err := kademlia.NewKademliaNode(kademliaNetwork, network.Address{
		IP:   localIP,
		Port: 8000,
	}, *kademliaId)
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

	if bootstrap == "" {
		addrs, error := kademliaNetwork.ResolveTasks("KademliaStack_bootstrapNode", 8000)
		addr := addrs[0]

		if error != nil {
			log.Debugln("Error resolving service: ", error)
		}

		joinIDKademliaID := kademliaID.NewKademliaID(joinID)

		if len(addr) > 0 {
			log.Infof("Resolved addresses: %v", addr)
			joinNetwork(node, addr, joinIDKademliaID)
		}
	} else {
		log.Info("Starting in bootstrap mode - skipping network discovery and joining")
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

func joinNetwork(node kademlia.IKademliaNode, addressToJoin string, joinID *kademliaID.KademliaID) {
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

	go node.Join(&network.Address{IP: ip, Port: port}, joinID)
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
