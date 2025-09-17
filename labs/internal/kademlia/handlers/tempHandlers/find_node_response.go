package tempHandlers

import (
	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/pkg/network"
)

type ContactsWithId struct {
	Contacts []kademliaContact.Contact
	MsgId    *kademliaID.KademliaID
}

func FindNodeResponseTempHandler(msg *network.Message, contactCh chan ContactsWithId, valueCh chan string) error {
	return nil
}
