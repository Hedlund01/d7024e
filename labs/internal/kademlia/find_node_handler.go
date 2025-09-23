package kademlia

import (
	"encoding/json"

	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/pkg/network"

	log "github.com/sirupsen/logrus"
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
	contacts := node.GetRoutingTable().FindClosestContacts(id, 4)
	finalContacts := []kademliaContact.Contact{}
	for _, contact := range contacts {
		if !contact.ID.Equals(msg.FromID) {
			finalContacts = append(finalContacts, contact)
		}
	}
	log.WithField("func", "FindNodeRequestHandler").WithField("from", msg.From.String()).WithField("targetID", id.String()).WithField("contactCount", len(contacts)).Debugf("Node %s received find node request from %s for id %s. Responding with %d contacts.", node.GetRoutingTable().me.Address, msg.From.String(), id.String(), len(finalContacts))
	err = node.SendFindNodeResponse(msg.From, finalContacts, msg.MessageID)
	if err != nil {
		return nil
	}
	return nil
}
