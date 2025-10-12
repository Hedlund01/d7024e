package kademlia

import (
	"encoding/json"

	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/pkg/network"

	log "github.com/sirupsen/logrus"
)

func FindValueRequestHandler(msg *network.Message, node IKademliaNode) error {
	log.WithField("from", msg.From.String()).WithField("msgID", msg.MessageID.String()).Debugf("Node %s received FIND_VALUE request", node.Address().String())
	id := kademliaID.KademliaID{}

	json.Unmarshal(msg.Payload, &id)
	val, err := node.GetValue(&id)
	if err != nil {
		contacts := node.GetRoutingTable().FindClosestContacts(&id, 4)
		finalContacts := []kademliaContact.Contact{}
		for _, contact := range contacts {
			if !contact.ID.Equals(msg.FromID) {
				finalContacts = append(finalContacts, contact)
			}
		}
		finalContacts = finalContacts[:min(len(finalContacts), 3)] // Send at most 3 contacts
		data, err := json.Marshal(finalContacts)
		if err != nil {
			log.WithField("msgID", msg.MessageID.String()).Error("Could not marshal contacts")
			return err
		}
		log.WithField("msgID", msg.MessageID.String()).Debugf("Value not found locally, sending contacts")
		err = node.SendFindValueResponse(msg.From, data, msg.MessageID)
		if err != nil {
			log.WithField("msgID", msg.MessageID.String()).WithField("func", "FindValueRequestHandler").Error("Could not send FIND_VALUE response")
		}
		return err
	}

	log.WithField("msgID", msg.MessageID.String()).Debugf("Value found locally, sending value")
	return node.SendFindValueResponse(msg.From, val, msg.MessageID)
}

func FindValueResponseTempHandler(msg *network.Message, contactCh chan []kademliaContact.Contact, valueCh chan []byte) error {
	log.WithField("from", msg.From.String()).WithField("msgID", msg.MessageID.String()).Debugf("Node %s received FIND_VALUE response", msg.To.String())

	// First try to unmarshal as contacts
	contacts := []kademliaContact.Contact{}
	if err := json.Unmarshal(msg.Payload, &contacts); err == nil && len(contacts) > 0 {
		contactCh <- contacts
		return nil
	}

	log.WithField("msgID", msg.MessageID.String()).Debugf("FIND_VALUE response does not contain contacts, treating as raw value")
	// If that fails, treat as raw value
	valueCh <- msg.Payload
	return nil
}
