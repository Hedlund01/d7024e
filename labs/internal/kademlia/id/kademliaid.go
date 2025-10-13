package kademliaID

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
)

// the static number of bytes in a KademliaID
const IDLength = 20

// type definition of a KademliaID
type KademliaID [IDLength]byte

// NewKademliaID returns a new instance of a KademliaID based on the string input
func NewKademliaID(data string) *KademliaID {
	hasher := sha256.New()
	hasher.Write([]byte(data))
	hash := hasher.Sum(nil)

	newKademliaID := KademliaID{}
	for i := 0; i < IDLength; i++ {
		newKademliaID[i] = hash[i]
	}

	return &newKademliaID
}



// NewRandomKademliaID returns a new instance of a random KademliaID,
// change this to a better version if you like
func NewRandomKademliaID() *KademliaID {
	hasher := sha256.New()
	randomData := make([]byte, 256)
	rand.Read(randomData)
	hasher.Write(randomData)
	hash := hasher.Sum(nil)
	newKademliaID := KademliaID{}
	for i := 0; i < IDLength; i++ {
		newKademliaID[i] = hash[i]
	}
	return &newKademliaID
}

// Less returns true if kademliaID < otherKademliaID (bitwise)
func (kademliaID KademliaID) Less(otherKademliaID *KademliaID) bool {
	for i := 0; i < IDLength; i++ {
		if kademliaID[i] != otherKademliaID[i] {
			return kademliaID[i] < otherKademliaID[i]
		}
	}
	return false
}

// Equals returns true if kademliaID == otherKademliaID (bitwise)
func (kademliaID KademliaID) Equals(otherKademliaID *KademliaID) bool {
	for i := 0; i < IDLength; i++ {
		if kademliaID[i] != otherKademliaID[i] {
			return false
		}
	}
	return true
}

// CalcDistance returns a new instance of a KademliaID that is built
// through a bitwise XOR operation betweeen kademliaID and target
func (kademliaID KademliaID) CalcDistance(target *KademliaID) *KademliaID {
	result := KademliaID{}
	for i := 0; i < IDLength; i++ {
		result[i] = kademliaID[i] ^ target[i]
	}
	return &result
}

// String returns a simple string representation of a KademliaID
func (kademliaID *KademliaID) String() string {
	return hex.EncodeToString(kademliaID[0:IDLength])
}

// NewKademliaIDFromHexString parses a hex string and returns a KademliaID
func NewKademliaIDFromHexString(hexStr string) (*KademliaID, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	if len(bytes) != IDLength {
		return nil, hex.ErrLength
	}
	newKademliaID := KademliaID{}
	for i := 0; i < IDLength; i++ {
		newKademliaID[i] = bytes[i]
	}
	return &newKademliaID, nil
}
