//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package util

import (
	"math"
	"math/big"
	"math/bits"
)

// the smallest power of two greater than the input
func NextPowerOf2(value uint64) uint64 {
	return 1 << Log2ceil(value)
}

// the log2 of the int, rounded up
func Log2ceil(value uint64) uint64 {
	return uint64(64 - bits.LeadingZeros64(value))
}

// check huge equality
func BigEquals(first, second *big.Int) bool {
	return first.Cmp(second) == 0
}

// check if a huge is less than another
func BigLessThan(first, second *big.Int) bool {
	return first.Cmp(second) < 0
}

// add a huge to another
func BigAdd(augend *big.Int, addend *big.Int) *big.Int {
	return new(big.Int).Add(augend, addend)
}

// subtract from a huge another
func BigSub(minuend *big.Int, subtrahend *big.Int) *big.Int {
	return new(big.Int).Sub(minuend, subtrahend)
}

// multiply a huge by another
func BigMul(multiplicand *big.Int, multiplier *big.Int) *big.Int {
	return new(big.Int).Mul(multiplicand, multiplier)
}

// divide a huge by another
func BigDiv(dividend *big.Int, divisor *big.Int) *big.Int {
	return new(big.Int).Mul(dividend, divisor)
}

// multiply a huge by a rational
func BigMulByFrac(value *big.Int, numerator int64, denominator int64) *big.Int {
	value = new(big.Int).Set(value)
	value.Mul(value, big.NewInt(numerator))
	value.Div(value, big.NewInt(denominator))
	return value
}

// multiply a huge by a rational whose components are non-negative
func BigMulByUfrac(value *big.Int, numerator uint64, denominator uint64) *big.Int {
	value = new(big.Int).Set(value)
	value.Mul(value, new(big.Int).SetUint64(numerator))
	value.Div(value, new(big.Int).SetUint64(denominator))
	return value
}

// multiply a huge by an integer
func BigMulByInt(multiplicand *big.Int, multiplier int64) *big.Int {
	return new(big.Int).Mul(multiplicand, big.NewInt(multiplier))
}

// multiply a huge by a unsigned integer
func BigMulByUint(multiplicand *big.Int, multiplier uint64) *big.Int {
	return new(big.Int).Mul(multiplicand, new(big.Int).SetUint64(multiplier))
}

// add two int64's without overflow
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

// cast a uint64 to an int64, clipping to [0, 2^63-1]
func SaturatingCast(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

// the number of eth-words needed to store n bytes
func WordsForBytes(nbytes uint64) uint64 {
	return (nbytes + 31) / 32
}
