// Package math defines utilities for performing operations critical to the
// computations performed during a challenge in BOLD.
//
// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package math

import (
	"errors"
	"math"
	"math/bits"
)

var ErrUnableToBisect = errors.New("unable to bisect")

// Unsigned is a generic constraint for all unsigned numeric primitives.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

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
