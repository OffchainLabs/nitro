package util

import (
	"errors"
	"math"
	"math/bits"
)

var ErrUnableToBisect = errors.New("unable to bisect")

func BisectionPoint(pre, post uint64) (uint64, error) {
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
