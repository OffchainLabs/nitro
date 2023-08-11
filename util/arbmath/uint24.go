// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbmath

import (
	"errors"
	"math/big"
)

const MaxUint24 = 1<<24 - 1 // 16777215

type Uint24 uint32

func (value Uint24) ToBig() *big.Int {
	return UintToBig(uint64(value))
}

func (value Uint24) ToUint32() uint32 {
	return uint32(value)
}

func IntToUint24[T uint32 | uint64](value T) (Uint24, error) {
	if value > T(MaxUint24) {
		return Uint24(MaxUint24), errors.New("value out of range")
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
	return Uint24(value.Uint64())
}
