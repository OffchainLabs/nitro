package protocol

import (
	"math"
	"math/bits"
)

func bisectionPoint(pre, post uint64) (uint64, error) {
	if pre+2 > post {
		return 0, ErrInvalid
	}
	if pre+2 == post {
		return pre + 1, nil
	}
	matchingBits := bits.LeadingZeros64((post - 1) ^ pre)
	mask := uint64(math.MaxUint64) << (63 - matchingBits)
	return (post - 1) & mask, nil
}

func oldBisectionPointAlgorithm(pre, post uint64) (uint64, error) {
	if pre+2 > post {
		return 0, ErrInvalid
	}
	bitLen := 63 - bits.LeadingZeros64(post^pre)
	bitsMask := (uint64(1) << bitLen) - 1
	if post&bitsMask != 0 {
		return post - (1 << bits.TrailingZeros64(post)), nil
	} else {
		mask := uint64(1)
		for pre+(mask<<1) < post {
			mask <<= 1
		}
		return post - mask, nil
	}
}
