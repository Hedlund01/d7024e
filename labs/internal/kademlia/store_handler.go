package kademlia

import (
	"encoding/json"

	kademliaContact "d7024e/internal/kademlia/contact"
	"d7024e/internal/kademlia/shortlist"
	"d7024e/pkg/network"
)

func StoreRequestHandler(msg *network.Message, node IKademliaNode) error {
	data := shortlist.StoreData{}
	err := json.Unmarshal(msg.Payload, &data)
	if err != nil {
		errSend := node.SendStoreResponse(msg.From, err, msg.MessageID)
		if errSend != nil {
			return errSend
		}
	}
	err = node.StoreValue(data.Value, data.Hash)
	if err != nil {
		errSend := node.SendStoreResponse(msg.From, err, msg.MessageID)
		if errSend != nil {
			return errSend
		}
		return err
	}
	errSend := node.SendStoreResponse(msg.From, nil, msg.MessageID)
	if errSend != nil {
		return errSend
	}
	return nil
}

func StoreResponseTempHandler(msg *network.Message, contactCh chan []kademliaContact.Contact, valueCh chan []byte) error {
	valueCh <- msg.Payload
	return nil
}
