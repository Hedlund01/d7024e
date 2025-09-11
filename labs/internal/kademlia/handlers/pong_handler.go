package handlers
import (
	"d7024e/internal/kademlia"
	"d7024e/pkg/network"
)
func PongHandler(msg *network.Message, kn kademlia.IKademliaNode) error {
	// Handle the PONG message
	return nil
}
