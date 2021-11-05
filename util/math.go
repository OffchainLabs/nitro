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
	return uint64(64 - bits.LeadingZeros64(value))
}
