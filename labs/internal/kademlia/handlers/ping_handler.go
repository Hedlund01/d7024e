package handlers

import (
	"d7024e/internal/kademlia"
	"d7024e/pkg/network"
	"d7024e/pkg/node"
)

func PingHandler(msg network.Message, n node.INode) error {
	err := msg.ReplyString("PONG", "")

	// Access Kademlia-specific data through the interface
	if nodeData := n.GetNodeData(); nodeData != nil {
		if kademliaData, ok := nodeData.(*kademlia.KademliaNodeData); ok {
			kademliaData.RoutingTable.AddContact(kademlia.NewContact(kademlia.NewRandomKademliaID(), msg.From.String()))
		}
	}

	return err
}
