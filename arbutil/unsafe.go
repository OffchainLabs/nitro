// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

import "unsafe"

func SliceToPointer[T any](slice []T) *T {
	if len(slice) == 0 {
		return nil
	}
	return &slice[0]
}

func SliceToUnsafePointer[T any](slice []T) unsafe.Pointer {
	return unsafe.Pointer(SliceToPointer(slice))
}

// does a defensive copy due to Go's lake of immutable types
func PointerToSlice[T any](pointer *T, length int) []T {
	return CopySlice(unsafe.Slice(pointer, length))
}

func CopySlice[T any](slice []T) []T {
	output := make([]T, len(slice))
	copy(output, slice)
	return output
}
