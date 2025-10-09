package kademliaContact

import (
	kademliaID "d7024e/internal/kademlia/id"
	"d7024e/pkg/network"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Contact definition
// stores the KademliaID, the ip address and the distance
type Contact struct {
	ID       *kademliaID.KademliaID
	Address  string
	distance *kademliaID.KademliaID
}

// NewContact returns a new instance of a Contact
func NewContact(id *kademliaID.KademliaID, address string) Contact {
	return Contact{id, address, nil}
}

func (contact *Contact) CheckIfDistanceIsZero() bool {
	for _, b := range contact.distance {
		if b != 0 {
			return false
		}
	}
	return true
}

// CalcDistance calculates the distance to the target and
// fills the contacts distance field
func (contact *Contact) CalcDistance(target *kademliaID.KademliaID) {
	contact.distance = contact.ID.CalcDistance(target)
}

// Less returns true if contact.distance < otherContact.distance
func (contact *Contact) Less(otherContact *Contact) bool {
	return contact.distance.Less(otherContact.distance)
}

// Equals returns true if contact.distance = otherContact.distance
func (contact *Contact) Equals(otherContact *Contact) bool {
	return contact.distance.Equals(otherContact.distance)
}

// String returns a simple string representation of a Contact
func (contact *Contact) String() string {
	return fmt.Sprintf(`contact("%s", "%s")`, contact.ID, contact.Address)
}

func (contact *Contact) GetNetworkAddress() network.Address {
	ipPort := strings.Split(contact.Address, ":")
	port, err := strconv.Atoi(ipPort[1])
	if err != nil {
		port = 0 // or handle error as needed
	}
	return network.Address{
		IP:   ipPort[0],
		Port: port,
	}
}

// ContactCandidates definition
// stores an array of Contacts
type ContactCandidates struct {
	contacts []Contact
}

// Append an array of Contacts to the ContactCandidates
func (candidates *ContactCandidates) Append(contacts []Contact) {
	candidates.contacts = append(candidates.contacts, contacts...)
}

// GetContacts returns the first count number of Contacts
func (candidates *ContactCandidates) GetContacts(count int) []Contact {
	return candidates.contacts[:count]
}

// Sort the Contacts in ContactCandidates
func (candidates *ContactCandidates) Sort() {
	sort.Sort(candidates)
}

// Len returns the length of the ContactCandidates
func (candidates *ContactCandidates) Len() int {
	return len(candidates.contacts)
}

// Swap the position of the Contacts at i and j
// WARNING does not check if either i or j is within range
func (candidates *ContactCandidates) Swap(i, j int) {
	candidates.contacts[i], candidates.contacts[j] = candidates.contacts[j], candidates.contacts[i]
}

// Less returns true if the Contact at index i is smaller than
// the Contact at index j
func (candidates *ContactCandidates) Less(i, j int) bool {
	return candidates.contacts[i].Less(&candidates.contacts[j])
}
