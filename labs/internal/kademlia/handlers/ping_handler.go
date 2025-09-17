package handlers

import (
	"d7024e/internal/kademlia"
	"d7024e/pkg/network"
)

func PingHandler(msg *network.Message, node kademlia.IKademliaNode) error {
	err := node.SendPongMessage(msg.From, msg.MessageID)
	if err != nil {
		return err
	}

	return nil
}
