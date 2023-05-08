package util

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
)

var ErrUnableToBisect = errors.New("unable to bisect")

// Unsigned is a generic constraint for all unsigned numeric primitives.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

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

// Trunc truncates  a byte slice to 4 bytes and pretty-prints as a hex string.
func Trunc(b []byte) string {
	if len(b) < 4 {
		return fmt.Sprintf("%#x", b)
	}
	return fmt.Sprintf("%#x", b[:4])
}

// Reverse a generic slice.
func Reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// Computes the min value of a slice of unsigned elements.
// Returns none if the slice is empty.
func Min[T Unsigned](items []T) Option[T] {
	if len(items) == 0 {
		return None[T]()
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
	return Some(m)
}

// Computes the max value of a slice of unsigned elements.
// Returns none if the slice is empty.
func Max[T Unsigned](items []T) Option[T] {
	if len(items) == 0 {
		return None[T]()
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
	return Some(m)
}
