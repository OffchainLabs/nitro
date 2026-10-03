// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package util

func ArrayToSet[T comparable](arr []T) map[T]struct{} {
	ret := make(map[T]struct{})
	for _, elem := range arr {
		ret[elem] = struct{}{}
	}
	return ret
}
