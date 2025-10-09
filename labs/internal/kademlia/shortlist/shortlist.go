package shortlist

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"

	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"
)

type NodeState int

const (
	Unprobed NodeState = iota
	Probing
	Failed
	Succeeded
)

func (ns NodeState) String() string {
	switch ns {
	case Unprobed:
		return "Unprobed"
	case Probing:
		return "Probing"
	case Failed:
		return "Failed"
	case Succeeded:
		return "Succeeded"
	default:
		return fmt.Sprintf("NodeState(%d)", int(ns))
	}
}

type StoreData struct {
	Hash  *kademliaID.KademliaID
	Value []byte
}

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
	value       StoreData
}

func NewShortlist(target *kademliaID.KademliaID, k int, alpha int) *Shortlist {
	return &Shortlist{
		nodes:       make([]ShortlistNode, 0),
		nodeMap:     make(map[kademliaID.KademliaID]bool),
		target:      target,
		k:           k,
		alpha:       alpha,
		hasImproved: true,
	}
}

func (sl *Shortlist) AddContacts(contacts []kademliaContact.Contact) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	previousClosest := sl.getClosestContact()
	for _, contact := range contacts {
		if contact.ID.Equals(sl.target) {
			sl.targetFound = true
		}

		if !sl.nodeMap[*contact.ID] {
			contact.CalcDistance(sl.target)

			sl.nodes = append(sl.nodes, ShortlistNode{
				Contact: contact,
				State:   Unprobed,
			})
			sl.nodeMap[*contact.ID] = true
		}
	}

	sl.sort()

	if len(sl.nodes) > sl.k {
		sl.nodes = sl.nodes[:sl.k]
	}

	// Improved if we added contacts and got a better closest contact
	currentClosest := sl.getClosestContact()

	sl.hasImproved = false
	if previousClosest == nil {
		sl.hasImproved = true
		return
	} else {
		if currentClosest.ID.Equals(previousClosest.ID) {
			sl.hasImproved = false
			return
		}
		if currentClosest.Less(previousClosest) || currentClosest.Equals(previousClosest) {
			sl.hasImproved = true
			return
		}

		return

	}

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

func (sl *Shortlist) MarkSucceeded(contact kademliaContact.Contact) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	for i := range sl.nodes {
		if sl.nodes[i].Contact.ID.Equals(contact.ID) {
			sl.nodes[i].State = Succeeded
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

func (sl *Shortlist) SetTargetFound() {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.targetFound = true
}

func (sl *Shortlist) TargetFound() bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.targetFound
}

func (sl *Shortlist) GetClosestContact() *kademliaContact.Contact {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.getClosestContact()
}

func (sl *Shortlist) SetValue(hash *kademliaID.KademliaID, data []byte) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.value = StoreData{Hash: hash, Value: data}
}

func (sl *Shortlist) GetValue() StoreData {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.value
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

func (sl *Shortlist) GetAllProbedContacts() []kademliaContact.Contact {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	contacts := []kademliaContact.Contact{}
	for _, node := range sl.nodes {
		if node.State == Succeeded {
			contacts = append(contacts, node.Contact)
		}
	}
	return contacts
}

func (sl *Shortlist) sort() {
	sort.Slice(sl.nodes, func(i, j int) bool {
		iDistance := sl.nodes[i].Contact.ID.CalcDistance(sl.target)
		jDistance := sl.nodes[j].Contact.ID.CalcDistance(sl.target)
		return iDistance.Less(jDistance)
	})
}

func (sl *Shortlist) GetAllContacts() string {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	contacts := ""
	// Sort based on port
	sort.SliceStable(sl.nodes, func(i, j int) bool {
		return sl.nodes[i].Contact.Address < sl.nodes[j].Contact.Address
	})
	for _, node := range sl.nodes {
		contacts += node.Contact.Address + " (" + node.State.String() + "), "
	}
	contacts += " (Total: " + strconv.Itoa(len(sl.nodes)) + ")" + " (TargetFound: " + strconv.FormatBool(sl.targetFound) + ")"
	return contacts
}
