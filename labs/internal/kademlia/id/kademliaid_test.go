package kademliaID

import (
	"fmt"
	"testing"
)

func TestKademliaIDAndFromBytes(t *testing.T) {
	idStr := "Hello World!"
	id := NewKademliaID(idStr)
	if id == nil {
		t.Errorf("Expected KademliaID to be created, got nil")
	}

	idFromBytes := NewKademliaIDFromBytes(id[:])
	if idFromBytes == nil {
		t.Errorf("Expected KademliaID to be created from bytes, got nil")
	}
	fmt.Printf("id: %v\n", id)
	if !id.Equals(idFromBytes) {
		t.Errorf("Expected KademliaIDs to be equal")
	}
	if id.Less(idFromBytes) || idFromBytes.Less(id) {
		t.Errorf("Expected KademliaIDs to not be less than each other")
	}
}
