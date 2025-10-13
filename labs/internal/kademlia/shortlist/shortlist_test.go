package shortlist

import (
	"testing"

	kademliaContact "d7024e/internal/kademlia/contact"
	kademliaID "d7024e/internal/kademlia/id"

	"github.com/stretchr/testify/assert"
)

func TestNewShortlist(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)
	assert.NotNil(t, sl)
	assert.Equal(t, 0, sl.Len())
	assert.True(t, sl.HasImproved())
}

func TestNodeStateString(t *testing.T) {
	assert.Equal(t, "Unprobed", Unprobed.String())
	assert.Equal(t, "Probing", Probing.String())
	assert.Equal(t, "Failed", Failed.String())
	assert.Equal(t, "Succeeded", Succeeded.String())
	assert.Contains(t, NodeState(99).String(), "NodeState(99)")
}

func TestAddContacts(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	sl.AddContacts([]kademliaContact.Contact{contact1})
	assert.Equal(t, 1, sl.Len())
}

func TestAddContactsWithTargetFound(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	targetContact := kademliaContact.NewContact(target, "127.0.0.1:8000")
	sl.AddContacts([]kademliaContact.Contact{targetContact})
	assert.True(t, sl.TargetFound())
}

func TestAddContactsTrimsToK(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 3, 3)

	contacts := []kademliaContact.Contact{}
	for i := 0; i < 10; i++ {
		id := kademliaID.NewRandomKademliaID()
		contact := kademliaContact.NewContact(id, "127.0.0.1:8000")
		contacts = append(contacts, contact)
	}
	sl.AddContacts(contacts)
	assert.Equal(t, 3, sl.Len())
}

func TestGetUnprobed(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	sl.AddContacts([]kademliaContact.Contact{contact1})

	contact, err := sl.GetUnprobed()
	assert.NoError(t, err)
	assert.True(t, contact.ID.Equals(id1))
}

func TestGetUnprobedNoContacts(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	_, err := sl.GetUnprobed()
	assert.Error(t, err)
}

func TestGetAllUnprobed(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	contacts := []kademliaContact.Contact{}
	for i := 0; i < 3; i++ {
		id := kademliaID.NewRandomKademliaID()
		contact := kademliaContact.NewContact(id, "127.0.0.1:8000")
		contacts = append(contacts, contact)
	}
	sl.AddContacts(contacts)

	unprobed, err := sl.GetAllUnprobed()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(unprobed))
}

func TestGetAllUnprobedEmpty(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	_, err := sl.GetAllUnprobed()
	assert.Error(t, err)
}

func TestMarkFailed(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	sl.AddContacts([]kademliaContact.Contact{contact1})

	sl.MarkFailed(contact1)
	assert.Equal(t, 1, sl.Len())
}

func TestMarkSucceeded(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	sl.AddContacts([]kademliaContact.Contact{contact1})

	sl.MarkSucceeded(contact1)
	probed := sl.GetAllProbedContacts()
	assert.Equal(t, 1, len(probed))
}

func TestRemoveFailedNodes(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	id2 := kademliaID.NewKademliaID("contact2")
	contact2 := kademliaContact.NewContact(id2, "127.0.0.1:8002")
	sl.AddContacts([]kademliaContact.Contact{contact1, contact2})

	sl.MarkFailed(contact1)
	sl.RemoveFailedNodes()
	assert.Equal(t, 1, sl.Len())
}

func TestGetProbingCount(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	sl.AddContacts([]kademliaContact.Contact{contact1})

	sl.GetUnprobed()
	assert.Equal(t, 1, sl.GetProbingCount())
}

func TestHasUnprobed(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	assert.False(t, sl.HasUnprobed())

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	sl.AddContacts([]kademliaContact.Contact{contact1})

	assert.True(t, sl.HasUnprobed())
}

func TestSetTargetFound(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	assert.False(t, sl.TargetFound())
	sl.SetTargetFound()
	assert.True(t, sl.TargetFound())
}

func TestGetClosestContact(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	closest := sl.GetClosestContact()
	assert.Nil(t, closest)

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	sl.AddContacts([]kademliaContact.Contact{contact1})

	closest = sl.GetClosestContact()
	assert.NotNil(t, closest)
}

func TestSetAndGetValue(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	hash := kademliaID.NewKademliaID("hash")
	data := []byte("test data")
	sl.SetValue(hash, data)

	value := sl.GetValue()
	assert.True(t, value.Hash.Equals(hash))
	assert.Equal(t, data, value.Value)
}

func TestHasImproved(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	assert.True(t, sl.HasImproved())

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	sl.AddContacts([]kademliaContact.Contact{contact1})

	assert.True(t, sl.HasImproved())
}

func TestGetAllProbedContacts(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	id2 := kademliaID.NewKademliaID("contact2")
	contact2 := kademliaContact.NewContact(id2, "127.0.0.1:8002")
	sl.AddContacts([]kademliaContact.Contact{contact1, contact2})

	sl.MarkSucceeded(contact1)
	sl.MarkSucceeded(contact2)

	probed := sl.GetAllProbedContacts()
	assert.Equal(t, 2, len(probed))
}

func TestGetAllContacts(t *testing.T) {
	target := kademliaID.NewKademliaID("target")
	sl := NewShortlist(target, 20, 3)

	id1 := kademliaID.NewKademliaID("contact1")
	contact1 := kademliaContact.NewContact(id1, "127.0.0.1:8001")
	sl.AddContacts([]kademliaContact.Contact{contact1})

	contacts := sl.GetAllContacts()
	assert.NotEmpty(t, contacts)
	assert.Contains(t, contacts, "127.0.0.1:8001")
}
