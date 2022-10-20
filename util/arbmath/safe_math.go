// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbmath

import "errors"

// add two uint64's without overflow
func SafeUAdd(augend uint64, addend uint64) (uint64, error) {
	sum := augend + addend
	if sum < augend || sum < addend {
		return 0, errors.New("overflow")
	}
	return sum, nil
}
