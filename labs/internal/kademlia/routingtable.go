package kademlia

import (
	kademliaBucket "d7024e/internal/kademlia/bucket"
	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"

	log "github.com/sirupsen/logrus"
)

// RoutingTable definition
// keeps a refrence contact of me and an array of buckets
type RoutingTable struct {
	me      kademliaContact.Contact
	buckets [kademliaID.IDLength * 8]*kademliaBucket.Bucket
}

// NewRoutingTable returns a new instance of a RoutingTable
func NewRoutingTable(me kademliaContact.Contact) *RoutingTable {
	routingTable := &RoutingTable{}
	for i := 0; i < kademliaID.IDLength*8; i++ {
		routingTable.buckets[i] = kademliaBucket.NewBucket()
	}
	routingTable.me = me
	return routingTable
}

func (routingTable *RoutingTable) GetNumberOfConnections() int {
	total := 0
	for _, bucket := range routingTable.buckets {
		total += bucket.Len()
	}
	return total
}

// AddContact add a new contact to the correct Bucket
func (routingTable *RoutingTable) AddContact(contact kademliaContact.Contact) {
	bucketIndex := routingTable.getBucketIndex(contact.ID)
	bucket := routingTable.buckets[bucketIndex]
	bucket.AddContact(contact)
}

func (routingTable *RoutingTable) RemoveContact(contact kademliaContact.Contact) {
	bucketIndex := routingTable.getBucketIndex(contact.ID)
	bucket := routingTable.buckets[bucketIndex]
	bucket.RemoveContact(contact)
}

// FindClosestContacts finds the count closest Contacts to the target in the RoutingTable
func (routingTable *RoutingTable) FindClosestContacts(target *kademliaID.KademliaID, count int) []kademliaContact.Contact {
	var candidates kademliaContact.ContactCandidates
	bucketIndex := routingTable.getBucketIndex(target)
	bucket := routingTable.buckets[bucketIndex]

	candidates.Append(bucket.GetContactAndCalcDistance(target))

	for i := 1; (bucketIndex-i >= 0 || bucketIndex+i < kademliaID.IDLength*8) && candidates.Len() < count; i++ {
		if bucketIndex-i >= 0 {
			bucket = routingTable.buckets[bucketIndex-i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
		if bucketIndex+i < kademliaID.IDLength*8 {
			bucket = routingTable.buckets[bucketIndex+i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
	}

	candidates.Sort()

	if count > candidates.Len() {
		count = candidates.Len()
	}

	return candidates.GetContacts(count)
}

func (routingTable *RoutingTable) GetMe() kademliaContact.Contact {
	return routingTable.me
}

// getBucketIndex get the correct Bucket index for the KademliaID
func (routingTable *RoutingTable) getBucketIndex(id *kademliaID.KademliaID) int {
	myID := routingTable.GetMe().ID
	log.WithField("func", "getBucketIndex").WithField("myID", myID.String()).WithField("targetID", id.String()).Debug("Calculating bucket index.")
	distance := id.CalcDistance(myID)
	for i := 0; i < kademliaID.IDLength; i++ {
		for j := 0; j < 8; j++ {
			if (distance[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}

	return kademliaID.IDLength*8 - 1
}
