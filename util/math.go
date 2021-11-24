//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package util

import (
	"math/bits"
)

func NextPowerOf2(value uint64) uint64 {
	return 1 << Log2ceil(value)
}

func Log2ceil(value uint64) uint64 {
	return uint64(bits.Len64(value))
}

func Log2floor(value uint64) uint64 {
	if value == 0 {
		return 0
	}
	l2c := uint64(bits.Len64(value))
	if value == 1<<l2c {
		return l2c
	} else {
		return l2c - 1
	}
}
