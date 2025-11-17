// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
//
//go:build sp1

package wavmio

import (
	"unsafe"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
)

//go:wasmimport wavmio greedyResolveTypedPreimage
func greedyResolveTypedPreimage(ty uint32, hash unsafe.Pointer, offset uint32, available uint32, output unsafe.Pointer) uint32

const GREEDY_READ_INITIAL_CAPACITY = 1024

func greedyReadBuffer(f func(uint32, uint32, unsafe.Pointer) uint32) []byte {
	buf := make([]byte, GREEDY_READ_INITIAL_CAPACITY)
	full_length := f(0, GREEDY_READ_INITIAL_CAPACITY, unsafe.Pointer(&buf[0]))
	if full_length <= GREEDY_READ_INITIAL_CAPACITY {
		buf = buf[:full_length]
		return buf
	}
	// Enlarge buf to read remaining data
	remaining := full_length - GREEDY_READ_INITIAL_CAPACITY
	buf = append(buf, make([]byte, remaining)...)
	remaining_full_length := f(GREEDY_READ_INITIAL_CAPACITY, remaining, unsafe.Pointer(&buf[GREEDY_READ_INITIAL_CAPACITY]))
	if remaining_full_length != remaining {
		panic("Invalid second greedy read!")
	}
	return buf
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	return greedyReadBuffer(func(offset uint32, available uint32, buf unsafe.Pointer) uint32 {
		hashUnsafe := unsafe.Pointer(&hash[0])
		return greedyResolveTypedPreimage(uint32(ty), hashUnsafe, offset, available, buf)
	}), nil
}
