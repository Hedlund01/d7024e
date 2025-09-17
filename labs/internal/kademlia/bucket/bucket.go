package kademliaBucket

import (
	"container/list"
	kademliaID "d7024e/internal/kademlia/id"
	kademliaContact "d7024e/internal/kademlia/contact"
)

const bucketSize = 20


// Bucket definition
// contains a List
type Bucket struct {
	list *list.List
}
	
// NewBucket returns a new instance of a bucket
func NewBucket() *Bucket {
	bucket := &Bucket{}
	bucket.list = list.New()
	return bucket
}

func GetBucketSize() int {
	return bucketSize
}

// AddContact adds the Contact to the front of the bucket
// or moves it to the front of the bucket if it already existed
func (bucket *Bucket) AddContact(newContact kademliaContact.Contact) {
	var element *list.Element
	for e := bucket.list.Front(); e != nil; e = e.Next() {
		nodeID := e.Value.(kademliaContact.Contact).ID

		if (newContact).ID.Equals(nodeID) {
			element = e
		}
	}

	if element == nil {
		if bucket.list.Len() < bucketSize {
			bucket.list.PushFront(newContact)
		}
	} else {
		bucket.list.MoveToFront(element)
	}
}

// GetContactAndCalcDistance returns an array of Contacts where
// the distance has already been calculated
func (bucket *Bucket) GetContactAndCalcDistance(target *kademliaID.KademliaID) []kademliaContact.Contact {
	var contacts []kademliaContact.Contact

	for elt := bucket.list.Front(); elt != nil; elt = elt.Next() {
		contact := elt.Value.(kademliaContact.Contact)
		contact.CalcDistance(target)
		contacts = append(contacts, contact)
	}

	return contacts
}

// Len return the size of the bucket
func (bucket *Bucket) Len() int {
	return bucket.list.Len()
}
