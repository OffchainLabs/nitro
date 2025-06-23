// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build wasm
// +build wasm

package melwavmio

import (
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

const INITIAL_CAPACITY = 128
const QUERY_SIZE = 32

const IDX_LAST_BLOCKHASH = 0
const IDX_SEND_ROOT = 1
const IDX_MEL_ROOT = 2

func readBuffer(f func(uint32, unsafe.Pointer) uint32) []byte {
	buf := make([]byte, 0, INITIAL_CAPACITY)
	offset := 0
	for {
		if len(buf) < offset+QUERY_SIZE {
			buf = append(buf, make([]byte, offset+QUERY_SIZE-len(buf))...)
		}
		read := f(uint32(offset), unsafe.Pointer(&buf[offset]))
		offset += int(read)
		if read < QUERY_SIZE {
			buf = buf[:offset]
			return buf
		}
	}
}

func StubInit() {
}

func StubFinal() {
}

func GetStartMELRoot() (hash common.Hash) {
	hashUnsafe := unsafe.Pointer(&hash[0])
	getGlobalStateBytes32(IDX_MEL_ROOT, hashUnsafe)
	return
}

func GetEndParentChainBlockHash() (hash common.Hash) {
	hashUnsafe := unsafe.Pointer(&hash[0])
	getEndParentChainBlockHash(hashUnsafe)
	return
}

func SetEndMELRoot(hash common.Hash) {
	hashUnsafe := unsafe.Pointer(&hash[0])
	setGlobalStateBytes32(IDX_MEL_ROOT, hashUnsafe)
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	return readBuffer(func(offset uint32, buf unsafe.Pointer) uint32 {
		hashUnsafe := unsafe.Pointer(&hash[0])
		return resolveTypedPreimage(uint32(ty), hashUnsafe, offset, buf)
	}), nil
}
