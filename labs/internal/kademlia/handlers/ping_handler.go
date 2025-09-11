package handlers

import (
	"d7024e/internal/kademlia"
	"d7024e/pkg/network"
)

func PingHandler(msg *network.Message, kn kademlia.IKademliaNode) error {
	err := kn.SendPongMessage(msg.From, msg.MessageID)
	if err != nil {
		return err
	}

	return nil
}
