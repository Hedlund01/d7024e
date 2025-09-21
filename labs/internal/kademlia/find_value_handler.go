package kademlia

import (
	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/pkg/network"

	log "github.com/sirupsen/logrus"
)

func FindValueRequestHandler(msg *network.Message, node IKademliaNode) error {
	log.WithField("from", msg.From.String()).WithField("msgID", msg.MessageID.String()).Debugf("Node %s received FIND_VALUE request", node.Address().String())

	id := kademliaID.NewKademliaID(string(msg.Payload))
	val, err := node.GetValue(id)

	if err != nil {
		log.WithField("msgID", msg.MessageID.String()).Debugf("Value not found locally, sending contacts")
		return node.SendFindValueResponse(msg.From, nil, msg.MessageID)
	}

	log.WithField("msgID", msg.MessageID.String()).Debugf("Value found locally, sending value")
	return node.SendFindValueResponse(msg.From, val, msg.MessageID)
}

func FindValueResponseTempHandler(msg *network.Message, contactCh chan []kademliaContact.Contact, valueCh chan []byte) error {
	valueCh <- msg.Payload
	return nil
}
