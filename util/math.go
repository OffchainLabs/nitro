//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package util

import (
	"math"
	"math/big"
	"math/bits"
)

func NextPowerOf2(value uint64) uint64 {
	return 1 << Log2ceil(value)
}

func Log2ceil(value uint64) uint64 {
	return uint64(64 - bits.LeadingZeros64(value))
}

func BigMulByFrac(value *big.Int, numerator int64, denominator int64) *big.Int {
	value = new(big.Int).Set(value)
	value.Mul(value, big.NewInt(numerator))
	value.Div(value, big.NewInt(denominator))
	return value
}

func SaturatingAdd(augend int64, addend int64) int64 {
	sum := augend + addend
	if addend > 0 && sum < augend {
		sum = math.MaxInt64
	}
	if addend < 0 && sum > augend {
		sum = math.MinInt64
	}
	return sum
}

func SaturatingCast(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

func WordsForBytes(nbytes uint64) uint64 {
	return (nbytes + 31) / 32
}
