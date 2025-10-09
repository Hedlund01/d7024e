package cli

import (
	"bufio"
	"context"
	"d7024e/internal/kademlia"
	kademliaID "d7024e/internal/kademlia/id"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type ShellContext struct {
	node   kademlia.IKademliaNode
	ctx    context.Context
	cancel context.CancelFunc
}

var shellCtx *ShellContext

func shellPrint(msg string) {
	fmt.Printf("[SHELL] %s\n", msg)
}

func shellPrintf(format string, args ...interface{}) {
	fmt.Printf("[SHELL] "+format+"\n", args...)
}

func shellError(msg string) {
	fmt.Fprintf(os.Stderr, "[SHELL ERROR] %s\n", msg)
}

func shellErrorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[SHELL ERROR] "+format+"\n", args...)
}

// StartInteractiveShell starts the interactive shell directly
func StartInteractiveShell(node kademlia.IKademliaNode, ctx context.Context, cancel context.CancelFunc) {
	shellCtx = &ShellContext{
		node:   node,
		ctx:    ctx,
		cancel: cancel,
	}

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
	shellPrint("=== Kademlia Node Interactive Shell ===")
	shellPrint("Type 'help' for available commands")
	shellPrint("Commands: status, put -v \"value\", get -h <hash>, exit, shutdown")
	shellPrint("")

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

		args := parseCommandLine(input)
		if len(args) == 0 {
			continue
		}

		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			shellErrorf("Command execution failed: %v", err)
		}

		// Check if context is cancelled to exit gracefully
		select {
		case <-shellCtx.ctx.Done():
			shellPrint("\nShell exiting due to shutdown signal...")
			return
		default:
		}
	}
}

// parseCommandLine parses a command line string respecting quoted strings (both single and double quotes)
func parseCommandLine(input string) []string {
	var args []string
	var current strings.Builder
	var quoteChar rune = 0 // 0 means not in quote, otherwise holds the quote character
	escaped := false

	for i, char := range input {
		if escaped {
			current.WriteRune(char)
			escaped = false
			continue
		}

		switch char {
		case '\\':
			escaped = true
		case '"', '\'':
			if quoteChar == 0 {
				// Starting a quoted section
				quoteChar = char
			} else if quoteChar == char {
				// Ending a quoted section
				quoteChar = 0
			} else {
				// Different quote character inside quotes
				current.WriteRune(char)
			}
		case ' ', '\t':
			if quoteChar != 0 {
				// Inside quotes, keep the space
				current.WriteRune(char)
			} else if current.Len() > 0 {
				// Outside quotes, space is a separator
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}

		// Handle end of string
		if i == len(input)-1 && current.Len() > 0 {
			args = append(args, current.String())
		}
	}

	return args
}

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

func createExitCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "exit",
		Aliases: []string{"quit", "q", "stop"},
		Short:   "Exit the interactive shell (node continues running)",
		Long:    "Closes all network connections, shuts down the node, and exits the application",
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
		Long:  "Fetches a value using its hash from the Kademlia network",
		Run: func(cmd *cobra.Command, args []string) {
			handleGet(cmd)
		},
	}

	cmd.Flags().StringP("hash", "", "", "Hash of the value to retrieve from the network")
	return cmd
}

func handleStatus() {
	if shellCtx == nil || shellCtx.node == nil {
		shellError("Node not available")
		return
	}

	rt := shellCtx.node.GetRoutingTable()
	me := rt.GetMe()

	shellPrint("=== Node Status ===")
	shellPrintf("Node ID: %s", me.ID.String())
	shellPrintf("Address: %s", me.GetNetworkAddress().String())
	shellPrintf("Number of Contacts: %d", rt.GetNumberOfConnections())
}

// handleExit handles the exit command logic (shell only)
func handleExit() {
	if shellCtx == nil {
		shellPrint("Goodbye!")
		return
	}

	shellPrint("Exiting interactive shell...")

	if shellCtx.cancel != nil {
		shellCtx.cancel()
	}
}

func handlePut(cmd *cobra.Command) {
	if shellCtx == nil || shellCtx.node == nil {
		shellError("Node not available")
		return
	}

	value, _ := cmd.Flags().GetString("value")
	if value == "" {
		shellError("Value is required")
		return
	}

	valueBytes := []byte(value)

	// Calculate the hash that will be used as the key
	hash := kademliaID.NewKademliaID(value)

	err := shellCtx.node.Store(valueBytes)
	if err != nil {
		shellErrorf("Error storing value: %v", err)
		return
	}

	shellPrint("Value stored successfully")
	shellPrintf("Hash: %s", hash.String())
}

func handleGet(cmd *cobra.Command) {
	if shellCtx == nil || shellCtx.node == nil {
		shellError("Node not available")
		return
	}

	hash, _ := cmd.Flags().GetString("hash")
	if hash == "" {
		shellError("Hash is required")
		return
	}

	id, err := kademliaID.NewKademliaIDFromHexString(hash)
	if err != nil {
		shellErrorf("Invalid hash format: %v", err)
		return
	}

	shellPrintf("Looking for value with hash: %s", id.String())

	retrievedValue, err := shellCtx.node.FindValue(id)
	if err != nil {
		shellErrorf("Error retrieving value: %v", err)
		return
	}

	shellPrintf("Retrieved value: %s", string(retrievedValue))
}
