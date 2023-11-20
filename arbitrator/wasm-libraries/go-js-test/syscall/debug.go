// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package syscall

func debugPoolHash() uint64

func PoolHash() uint64 {
	return debugPoolHash()
}
