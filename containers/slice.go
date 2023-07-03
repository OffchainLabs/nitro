// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE

package containers

import "fmt"

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
