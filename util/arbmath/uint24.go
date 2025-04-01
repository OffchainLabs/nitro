// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbmath

import (
	"encoding/binary"
	"errors"
	"math/big"
)

type Uint24 uint32

const MaxUint24 = 1<<24 - 1 // 16777215

func (value Uint24) ToBig() *big.Int {
	return UintToBig(uint64(value))
}

func (value Uint24) ToUint32() uint32 {
	return uint32(value)
}

func (value Uint24) ToUint64() uint64 {
	return uint64(value)
}

func IntToUint24[T uint32 | uint64](value T) (Uint24, error) {
	// #nosec G115
	if value > T(MaxUint24) {
		return MaxUint24, errors.New("value out of range")
	}
	return Uint24(value), nil
}

// Casts a huge to a uint24, panicking if out of bounds
func BigToUint24OrPanic(value *big.Int) Uint24 {
	if value.Sign() < 0 {
		panic("big.Int value is less than 0")
	}
	if !value.IsUint64() || value.Uint64() > MaxUint24 {
		panic("big.Int value exceeds the max Uint24")
	}
	// #nosec G115
	return Uint24(value.Uint64())
}

// creates a uint24 from its big-endian representation
func BytesToUint24(value []byte) Uint24 {
	value32 := ConcatByteSlices([]byte{0}, value)
	return Uint24(binary.BigEndian.Uint32(value32))
}

// casts a uint24 to its big-endian representation
func Uint24ToBytes(value Uint24) []byte {
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, value.ToUint32())
	return result[1:]
}
