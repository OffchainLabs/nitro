package math

import (
	"errors"
	"math"
	"math/bits"

	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
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

// Computes the min value of a slice of unsigned elements.
// Returns none if the slice is empty.
func Min[T Unsigned](items []T) option.Option[T] {
	if len(items) == 0 {
		return option.None[T]()
	}
	var m T
	if len(items) > 0 {
		m = items[0]
	}
	for i := 1; i < len(items); i++ {
		if items[i] < m {
			m = items[i]
		}
	}
	return option.Some(m)
}

// Computes the max value of a slice of unsigned elements.
// Returns none if the slice is empty.
func Max[T Unsigned](items []T) option.Option[T] {
	if len(items) == 0 {
		return option.None[T]()
	}
	var m T
	if len(items) > 0 {
		m = items[0]
	}
	for i := 1; i < len(items); i++ {
		if items[i] > m {
			m = items[i]
		}
	}
	return option.Some(m)
}
