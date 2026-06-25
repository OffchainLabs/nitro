// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbmath

import (
	"errors"
	"math"
	"math/bits"
)

var ErrUnableToBisect = errors.New("unable to bisect")

// Bisect returns the midpoint in (pre, post] aligned to the highest matching prefix.
func Bisect(pre, post uint64) (uint64, error) {
	if pre+2 > post {
		return 0, ErrUnableToBisect
	}
	if pre+2 == post {
		return pre + 1, nil
	}
	matchingBits := bits.LeadingZeros64((post - 1) ^ pre)
	mask := uint64(math.MaxUint64) << (63 - matchingBits)
	return (post - 1) & mask, nil
}

// Log2Floor returns the integer logarithm base 2 of u (rounded down).
func Log2Floor(u uint64) int {
	if u == 0 {
		panic("log2 undefined for non-positive values")
	}
	return bits.Len64(u) - 1
}

// Log2Ceil returns the integer logarithm base 2 of u (rounded up).
func Log2Ceil(u uint64) int {
	r := Log2Floor(u)
	if isPowerOfTwo(u) {
		return r
	}
	return r + 1
}

func isPowerOfTwo(u uint64) bool {
	return u&(u-1) == 0
}
