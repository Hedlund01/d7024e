package tempHandlers

import (
	kademliaContact "d7024e/internal/kademlia/contact"
	"d7024e/pkg/network"
	"encoding/json"
)

func FindNodeResponseTempHandler(msg *network.Message, contactCh chan []kademliaContact.Contact, valueCh chan string) error {
	contacts := []kademliaContact.Contact{}
	error := json.Unmarshal(msg.Payload, &contacts)
	println("FindNodeResponseTempHandler: received contacts:", contacts)
	contactCh <- contacts
	return error
}
