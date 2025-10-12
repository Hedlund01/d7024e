package kademliaID

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewKademliaID(t *testing.T) {
	idStr := "Hello World!"
	id := NewKademliaID(idStr)
	assert.NotNil(t, id)
	assert.Equal(t, IDLength, len(id))

	id2 := NewKademliaID(idStr)
	assert.True(t, id.Equals(id2))

	differentStr := "Different String"
	id3 := NewKademliaID(differentStr)
	assert.False(t, id.Equals(id3))
}



func TestLess(t *testing.T) {
	id1 := NewKademliaID("test1")
	id2 := NewKademliaID("test1")
	assert.False(t, id1.Less(id2))
	assert.False(t, id2.Less(id1))

	smallerID, _ := NewKademliaIDFromHexString("0100000000000000000000000000000000000000")
	largerID, _ := NewKademliaIDFromHexString("0200000000000000000000000000000000000000")
	assert.True(t, smallerID.Less(largerID))
	assert.False(t, largerID.Less(smallerID))

	id5, _ := NewKademliaIDFromHexString("0000000000000000000000000000000000000001")
	id6, _ := NewKademliaIDFromHexString("0000000000000000000000000000000000000002")
	assert.True(t, id5.Less(id6))
	assert.False(t, id6.Less(id5))
}

func TestEquals(t *testing.T) {
	id1 := NewKademliaID("test")
	id2 := NewKademliaID("test")
	assert.True(t, id1.Equals(id2))

	id3 := NewKademliaID("different")
	assert.False(t, id1.Equals(id3))

	id4, _ := NewKademliaIDFromHexString("000102030405060708090a0b0c0d0e0f10111213")
	id5, _ := NewKademliaIDFromHexString("000102030405060708090a0b0c0d0e0f10111213")
	assert.True(t, id4.Equals(id5))
}

func TestCalcDistance(t *testing.T) {
	id1 := NewKademliaID("test1")
	distanceToSelf := id1.CalcDistance(id1)
	assert.NotNil(t, distanceToSelf)

	expectedZero, _ := NewKademliaIDFromHexString("0000000000000000000000000000000000000000")
	assert.True(t, distanceToSelf.Equals(expectedZero))

	id2 := NewKademliaID("test2")
	distance1to2 := id1.CalcDistance(id2)
	distance2to1 := id2.CalcDistance(id1)
	assert.True(t, distance1to2.Equals(distance2to1))

	id3, _ := NewKademliaIDFromHexString("ffffffffffffffffffffffffffffffffffffffff")
	id4, _ := NewKademliaIDFromHexString("0000000000000000000000000000000000000000")
	distance := id3.CalcDistance(id4)
	expectedDistance, _ := NewKademliaIDFromHexString("ffffffffffffffffffffffffffffffffffffffff")
	assert.True(t, distance.Equals(expectedDistance))
}

func TestString(t *testing.T) {
	id, _ := NewKademliaIDFromHexString("000102030405060708090a0b0c0d0e0f10111213")
	str := id.String()
	assert.NotEmpty(t, str)
	assert.Equal(t, IDLength*2, len(str))
	assert.Equal(t, "000102030405060708090a0b0c0d0e0f10111213", str)

	id2 := NewKademliaID("test")
	str2 := id2.String()
	assert.NotEmpty(t, str2)
	assert.Equal(t, IDLength*2, len(str2))
}

func TestNewKademliaIDFromHexString(t *testing.T) {
	originalID := NewKademliaID("test")
	hexStr := originalID.String()
	parsedID, err := NewKademliaIDFromHexString(hexStr)
	assert.NoError(t, err)
	assert.NotNil(t, parsedID)
	assert.True(t, originalID.Equals(parsedID))

	validHex := "0102030405060708090a0b0c0d0e0f1011121314"
	id, err := NewKademliaIDFromHexString(validHex)
	assert.NoError(t, err)
	assert.NotNil(t, id)

	invalidHex := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	_, err = NewKademliaIDFromHexString(invalidHex)
	assert.Error(t, err)

	tooShortHex := "0102030405"
	_, err = NewKademliaIDFromHexString(tooShortHex)
	assert.Error(t, err)

	tooLongHex := "0102030405060708090a0b0c0d0e0f101112131415161718"
	_, err = NewKademliaIDFromHexString(tooLongHex)
	assert.Error(t, err)

	emptyHex := ""
	_, err = NewKademliaIDFromHexString(emptyHex)
	assert.Error(t, err)
}
