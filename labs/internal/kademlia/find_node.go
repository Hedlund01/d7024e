package kademlia

import (
	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/pkg/network"
	"encoding/json"
)

func FindNodeResponseTempHandler(msg *network.Message, contactCh chan []kademliaContact.Contact, valueCh chan []byte) error {
	contacts := []kademliaContact.Contact{}
	error := json.Unmarshal(msg.Payload, &contacts)
	contactCh <- contacts
	return error
}

func FindNodeRequestHandler(msg *network.Message, node IKademliaNode) error {
	id := &kademliaID.KademliaID{}
	err := json.Unmarshal(msg.Payload, id)
	if err != nil {
		return err
	}
	contacts := node.GetRoutingTable().FindClosestContacts(id, 3)
	err = node.SendFindNodeResponse(msg.From, contacts, msg.MessageID)
	if err != nil {
		return nil
	}
	return nil
}
