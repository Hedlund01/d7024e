package cli

import (
	"bufio"
	"context"
	"d7024e/internal/kademlia"
	kademliaID "d7024e/internal/kademlia/id"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Shell context holds the node instance and other shared state
type ShellContext struct {
	node   kademlia.IKademliaNode
	ctx    context.Context
	cancel context.CancelFunc
}

var shellCtx *ShellContext

// StartInteractiveShell starts the interactive shell directly
func StartInteractiveShell(node kademlia.IKademliaNode, ctx context.Context, cancel context.CancelFunc) {
	shellCtx = &ShellContext{
		node:   node,
		ctx:    ctx,
		cancel: cancel,
	}

	// Create root shell command for internal use only
	shellCmd := &cobra.Command{
		Use:          "shell",
		Short:        "Interactive Kademlia node shell",
		Long:         "Interactive shell for controlling and monitoring the Kademlia node",
		SilenceUsage: true, // Don't show usage on command errors
	}

	// Add shell subcommands
	shellCmd.AddCommand(createStatusCommand())
	shellCmd.AddCommand(createExitCommand())
	shellCmd.AddCommand(createPutCommand())
	shellCmd.AddCommand(createGetCommand())

	startShell(shellCmd)
}

// startShell begins the interactive shell loop
func startShell(rootCmd *cobra.Command) {
	log.Info("=== Kademlia Node Interactive Shell ===")
	log.Info("Type 'help' for available commands or 'exit' to quit")
	handleStatus()

	reader := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("kademlia> ")

		if !reader.Scan() {
			break
		}

		input := strings.TrimSpace(reader.Text())
		if input == "" {
			continue
		}

		// Parse the input into command and arguments
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		// Set the arguments for the root command and execute
		rootCmd.SetArgs(args)

		// Execute the command
		if err := rootCmd.Execute(); err != nil {
			log.Errorf("Error: %v", err)
		}

		// Check if context was cancelled (exit requested)
		select {
		case <-shellCtx.ctx.Done():
			return
		default:
			// Continue with next iteration
		}
	}
}

// createStatusCommand creates the status command
func createStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display node status and routing table information",
		Long:  "Shows detailed information about the current node including ID, address, contacts, and routing table statistics",
		Run: func(cmd *cobra.Command, args []string) {
			handleStatus()
		},
	}
}

// createExitCommand creates the exit command
func createExitCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "exit",
		Aliases: []string{"quit", "q"},
		Short:   "Gracefully shutdown the node and exit",
		Long:    "Closes all network connections, saves state, and exits the interactive shell",
		Run: func(cmd *cobra.Command, args []string) {
			handleExit()
		},
	}
}

func createPutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put",
		Short: "Store a value in the Kademlia network",
		Long:  "Stores a key-value pair in the Kademlia network",
		Run: func(cmd *cobra.Command, args []string) {
			handlePut(cmd)
		},
	}

	cmd.Flags().StringP("value", "v", "", "Value to store in the network")
	return cmd
}

func createGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a value from the Kademlia network",
		Long:  "Fetches a value associated with a key from the Kademlia network",
		Run: func(cmd *cobra.Command, args []string) {
			handleGet(cmd)
		},
	}

	cmd.Flags().StringP("value", "v", "", "Value to be hashed as key and be retrieved from the network")
	return cmd
}

// handleStatus handles the status command logic
func handleStatus() {
	if shellCtx == nil || shellCtx.node == nil {
		log.Error("Error: Node not available")
		return
	}

	rt := shellCtx.node.GetRoutingTable()
	me := rt.GetMe()

	log.Info("=== Node Status ===")
	log.Infof("Node ID: %s", me.ID.String())
	log.Infof("Address: %s", me.GetNetworkAddress().String())
	log.Infof("Number of Contacts: %d", rt.GetNumberOfConnections())
}

// handleExit handles the exit command logic
func handleExit() {
	if shellCtx == nil {
		log.Info("Goodbye!")
		return
	}

	log.Info("Shutting down node...")

	// Close the node gracefully
	if shellCtx.node != nil {
		if err := shellCtx.node.Close(); err != nil {
			log.Errorf("Error closing node: %v", err)
		}
	}

	// Cancel the context to stop the node
	if shellCtx.cancel != nil {
		shellCtx.cancel()
	}

	log.Info("Node shutdown complete. Goodbye!")

}

func handlePut(cmd *cobra.Command) {
	if shellCtx == nil || shellCtx.node == nil {
		log.Error("Error: Node not available")
		return
	}

	value, _ := cmd.Flags().GetString("value")
	if value == "" {
		log.Error("Error: Value is required")
		return
	}

	//convert value to byte array
	valueBytes := []byte(value)

	err := shellCtx.node.Store(valueBytes)
	if err != nil {
		log.Errorf("Error storing value: %v", err)
		return
	}

	log.Info("Value stored successfully")
}

func handleGet(cmd *cobra.Command) {
	if shellCtx == nil || shellCtx.node == nil {
		log.Error("Error: Node not available")
		return
	}

	value, _ := cmd.Flags().GetString("value")
	if value == "" {
		log.Error("Error: Value is required")
		return
	}

	id := kademliaID.NewKademliaID(value)

	retrievedValue, err := shellCtx.node.FindValue(id)
	if err != nil {
		log.Errorf("Error retrieving value: %v", err)
		return
	}

	log.Infof("Retrieved value: %s", retrievedValue)
}
