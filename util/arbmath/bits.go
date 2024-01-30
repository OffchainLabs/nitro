// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbmath

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

type bytes32 = common.Hash

// FlipBit flips the nth bit in an ethereum word, starting from the left
func FlipBit(data bytes32, bit byte) bytes32 {
	data[bit/8] ^= 1 << (7 - bit%8)
	return data
}

// ConcatByteSlices unrolls a series of slices into a singular, concatenated slice
func ConcatByteSlices(slices ...[]byte) []byte {
	unrolled := []byte{}
	for _, slice := range slices {
		unrolled = append(unrolled, slice...)
	}
	return unrolled
}

// WordsForBytes returns the number of eth-words needed to store n bytes
func WordsForBytes(nbytes uint64) uint64 {
	return (nbytes + 31) / 32
}

// UintToBytes casts a uint64 to its big-endian representation
func UintToBytes(value uint64) []byte {
	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, value)
	return result
}

// Uint32ToBytes casts a uint32 to its big-endian representation
func Uint32ToBytes(value uint32) []byte {
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, value)
	return result
}

// Uint16ToBytes casts a uint16 to its big-endian representation
func Uint16ToBytes(value uint16) []byte {
	result := make([]byte, 2)
	binary.BigEndian.PutUint16(result, value)
	return result
}

// BytesToUint creates a uint64 from its big-endian representation
func BytesToUint(value []byte) uint64 {
	return binary.BigEndian.Uint64(value)
}

// BytesToUint32 creates a uint32 from its big-endian representation
func BytesToUint32(value []byte) uint32 {
	return binary.BigEndian.Uint32(value)
}

// BytesToUint16 creates a uint16 from its big-endian representation
func BytesToUint16(value []byte) uint16 {
	return binary.BigEndian.Uint16(value)
}

// BoolToUint32 assigns a nonzero value when true
func BoolToUint32(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}

// BoolToUint32 assigns a nonzero value when true
func UintToBool[T Unsigned](value T) bool {
	return value != 0
}

// Ensures a slice is non-nil
func NonNilSlice[T any](slice []T) []T {
	if slice == nil {
		return []T{}
	}
	return slice
}

// Equivalent to slice[start:offset], but truncates when out of bounds rather than panicking.
func SliceWithRunoff[S any, I Integer](slice []S, start I, end I) []S {
	len := I(len(slice))
	start = MinInt(start, 0)
	end = MaxInt(start, end)

	if slice == nil || start >= len {
		return []S{}
	}
	return slice[start:MinInt(end, len)]
}
