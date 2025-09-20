package kademlia

import (
	kademliaContact "d7024e/internal/kademlia/contact"
	"d7024e/pkg/network"
)

func StoreRequestHandler(msg *network.Message, node IKademliaNode) error {
	err := node.StoreValue(msg.Payload, msg.MessageID)
	if err != nil {
		errSend := node.SendStoreResponse(msg.From, err, msg.MessageID)
		if errSend != nil {
			return errSend
		}
		return err
	}
	return nil
}

func StoreResponseTempHandler(msg *network.Message, contactCh chan []kademliaContact.Contact, valueCh chan []byte) error {
	valueCh <- msg.Payload
	return nil
}
