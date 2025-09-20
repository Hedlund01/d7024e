package kademlia

import (
	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/pkg/network"
)

func FindValueRequestHandler(msg *network.Message, node IKademliaNode) error {
	id := string(msg.Payload)
	val, err := node.GetValue(kademliaID.NewKademliaID(id))

	if err != nil {
		return node.SendFindValueResponse(msg.From, nil, msg.MessageID)
	}

	return node.SendFindValueResponse(msg.From, val, msg.MessageID)
}

func FindValueResponseTempHandler(msg *network.Message, contactCh chan []kademliaContact.Contact, valueCh chan []byte) error {
	valueCh <- msg.Payload
	return nil
}
