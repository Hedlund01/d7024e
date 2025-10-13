package kademlia

import (
	"d7024e/pkg/network"
)

func PingHandler(msg *network.Message, node IKademliaNode) error {
	err := node.SendPongMessage(msg.From, msg.MessageID)
	if err != nil {
		return err
	}

	return nil
}
