// Copyright 2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package math

import "math/bits"

// Log2Floor returns the integer logarithm base 2 of u (rounded down).
// Returns 0 for u == 0 to maintain consistency with arbmath.Log2ceil.
func Log2Floor(u uint64) int {
	if u == 0 {
		return 0
	}
	return bits.Len64(u) - 1
}

// Log2Ceil returns the integer logarithm base 2 of u (rounded up).
// Returns 0 for u == 0 to maintain consistency with arbmath.Log2ceil.
func Log2Ceil(u uint64) int {
	r := Log2Floor(u)
	if u == 0 || isPowerOfTwo(u) {
		return r
	}
	return r + 1
}

func isPowerOfTwo(u uint64) bool {
	return u&(u-1) == 0
}
