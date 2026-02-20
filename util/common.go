// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package util

import "reflect"

// IsNil checks whether v is nil, handling the case where a nil pointer
// of a concrete type is assigned to an interface (typed nil).
func IsNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

func ArrayToSet[T comparable](arr []T) map[T]struct{} {
	ret := make(map[T]struct{})
	for _, elem := range arr {
		ret[elem] = struct{}{}
	}
	return ret
}
