// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

import "unsafe"

func SliceToPointer[T any](slice []T) *T {
	if len(slice) == 0 {
		return nil
	}
	return &slice[0]
}

func PointerToSlice[T any](pointer *T, length int) []T {
	output := make([]T, length)
	source := unsafe.Slice(pointer, length)
	copy(output, source)
	return output
}
