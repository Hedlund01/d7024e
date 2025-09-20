package shortlist

import (
	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
	"errors"
	"sort"
	"sync"
)

type NodeState int

const (
	Unprobed NodeState = iota
	Probing
	Failed
)

type ShortlistNode struct {
	Contact kademliaContact.Contact
	State   NodeState
}

type Shortlist struct {
	nodes       []ShortlistNode
	nodeMap     map[kademliaID.KademliaID]bool
	target      *kademliaID.KademliaID
	k           int
	alpha       int
	hasImproved bool
	targetFound bool
	mu          sync.RWMutex
}

func NewShortlist(target *kademliaID.KademliaID, k int, alpha int) *Shortlist {
	return &Shortlist{
		nodes:       make([]ShortlistNode, 0),
		target:      target,
		k:           k,
		alpha:       alpha,
		hasImproved: true,
	}
}

func (sl *Shortlist) AddContact(contact kademliaContact.Contact) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	contact.CalcDistance(sl.target)

	node := ShortlistNode{
		Contact: contact,
		State:   Unprobed,
	}

	sl.nodes = append(sl.nodes, node)
	sl.sort()

	if len(sl.nodes) > sl.k {
		sl.nodes = sl.nodes[:sl.k]
	}
}

func (sl *Shortlist) AddContacts(contacts []kademliaContact.Contact) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	previousClosest := sl.getClosestContact()
	initialCount := len(sl.nodes)

	for _, contact := range contacts {
		if contact.ID.Equals(sl.target) {
			sl.targetFound = true
			continue
		}

		// Check for duplicates
		duplicate := false
		for _, node := range sl.nodes {
			if node.Contact.ID.Equals(contact.ID) {
				duplicate = true
				break
			}
		}

		if !duplicate {
			contact.CalcDistance(sl.target)
			sl.nodes = append(sl.nodes, ShortlistNode{
				Contact: contact,
				State:   Unprobed,
			})
		}
	}

	sl.sort()

	if len(sl.nodes) > sl.k {
		sl.nodes = sl.nodes[:sl.k]
	}

	// Improved if we added contacts and got a better closest contact
	currentClosest := sl.getClosestContact()
	sl.hasImproved = len(sl.nodes) > initialCount &&
		(previousClosest == nil || (currentClosest != nil && currentClosest.Less(previousClosest)))
}

func (sl *Shortlist) GetUnprobed() (kademliaContact.Contact, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	for i := range sl.nodes {
		if sl.nodes[i].State == Unprobed {
			contact := sl.nodes[i].Contact
			sl.nodes[i].State = Probing
			return contact, nil
		}
	}

	return kademliaContact.Contact{}, errors.New("no unprobed contacts available")
}

func (sl *Shortlist) GetAllUnprobed() ([]kademliaContact.Contact, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	contacts := []kademliaContact.Contact{}

	for i := range sl.nodes {
		if sl.nodes[i].State == Unprobed {
			contacts = append(contacts, sl.nodes[i].Contact)
			sl.nodes[i].State = Probing
		}
	}
	if len(contacts) == 0 {
		return nil, errors.New("no unprobed contacts available")
	}
	return contacts, nil
}

func (sl *Shortlist) MarkFailed(contact kademliaContact.Contact) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	for i := range sl.nodes {
		if sl.nodes[i].Contact.ID.Equals(contact.ID) {
			sl.nodes[i].State = Failed
			break
		}
	}
}

func (sl *Shortlist) RemoveFailedNodes() {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	filtered := make([]ShortlistNode, 0, len(sl.nodes))
	for _, node := range sl.nodes {
		if node.State != Failed {
			filtered = append(filtered, node)
		}
	}
	sl.nodes = filtered
}

func (sl *Shortlist) GetProbingCount() int {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	count := 0
	for _, node := range sl.nodes {
		if node.State == Probing {
			count++
		}
	}
	return count
}

func (sl *Shortlist) HasUnprobed() bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	for _, node := range sl.nodes {
		if node.State == Unprobed {
			return true
		}
	}
	return false
}

func (sl *Shortlist) TargetFound() bool {
	return sl.targetFound
}

func (sl *Shortlist) GetClosestContact() *kademliaContact.Contact {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.getClosestContact()
}

func (sl *Shortlist) getClosestContact() *kademliaContact.Contact {
	if len(sl.nodes) == 0 {
		return nil
	}
	return &sl.nodes[0].Contact
}

func (sl *Shortlist) HasImproved() bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.hasImproved
}

func (sl *Shortlist) Len() int {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	return len(sl.nodes)
}

func (sl *Shortlist) sort() {
	sort.Slice(sl.nodes, func(i, j int) bool {
		return sl.nodes[i].Contact.Less(&sl.nodes[j].Contact)
	})
}
