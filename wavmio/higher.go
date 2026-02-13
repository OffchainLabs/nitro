// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build wasm

package wavmio

import (
	"errors"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
)

const INITIAL_CAPACITY = 128
const QUERY_SIZE = 32

// bytes32
const IDX_LAST_BLOCKHASH = 0
const IDX_SEND_ROOT = 1

// u64
const IDX_INBOX_POSITION = 0
const IDX_POSITION_WITHIN_MESSAGE = 1

// INITIAL_PREIMAGE_ALLOCATION is the initial allocation size. If the preimage is larger than this, more space will be
// acquired and the remaining data (suffix) will be read.
const INITIAL_PREIMAGE_ALLOCATION = 512

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

func OnInit() {}

func OnReady() {
	beforeFirstIO()
}

func OnFinal() {}

func GetLastBlockHash() (hash common.Hash) {
	hashUnsafe := unsafe.Pointer(&hash[0])
	getGlobalStateBytes32(IDX_LAST_BLOCKHASH, hashUnsafe)
	return
}

func ReadInboxMessage(msgNum uint64) []byte {
	return readBuffer(func(offset uint32, buf unsafe.Pointer) uint32 {
		return readInboxMessage(msgNum, offset, buf)
	})
}

func ReadDelayedInboxMessage(seqNum uint64) []byte {
	return readBuffer(func(offset uint32, buf unsafe.Pointer) uint32 {
		return readDelayedInboxMessage(seqNum, offset, buf)
	})
}

func AdvanceInboxMessage() {
	pos := getGlobalStateU64(IDX_INBOX_POSITION)
	setGlobalStateU64(IDX_INBOX_POSITION, pos+1)
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	hashUnsafe := unsafe.Pointer(&hash[0])
	preimage := make([]byte, INITIAL_PREIMAGE_ALLOCATION)

	// 1. Read the preimage prefix (up to INITIAL_PREIMAGE_ALLOCATION bytes)
	preimageLen := readPreimage(uint32(ty), hashUnsafe, unsafe.Pointer(&preimage[0]), 0, INITIAL_PREIMAGE_ALLOCATION)

	// 2. If the preimage fits within the initial allocation, return it
	if preimageLen <= INITIAL_PREIMAGE_ALLOCATION {
		return preimage[:preimageLen], nil
	}

	// 3. Reallocate a buffer of the correct size
	remainingLen := preimageLen - INITIAL_PREIMAGE_ALLOCATION
	preimage = append(preimage, make([]byte, remainingLen)...)

	// 4. Read the remaining preimage data (the suffix)
	preimageLenOnSuffix := readPreimage(uint32(ty), hashUnsafe, unsafe.Pointer(&preimage[INITIAL_PREIMAGE_ALLOCATION]), INITIAL_PREIMAGE_ALLOCATION, remainingLen)
	if preimageLenOnSuffix != preimageLen {
		return nil, errors.New("reading preimage suffix failed")
	}
	return preimage[:preimageLen], nil
}

func ValidateCertificate(ty arbutil.PreimageType, hash common.Hash) bool {
	hashUnsafe := unsafe.Pointer(&hash[0])
	return validateCertificate(uint32(ty), hashUnsafe) != 0
}

func SetLastBlockHash(hash [32]byte) {
	hashUnsafe := unsafe.Pointer(&hash[0])
	setGlobalStateBytes32(IDX_LAST_BLOCKHASH, hashUnsafe)
}

// Note: if a GetSendRoot is ever modified, the validator will need to fill in the previous send root, which it currently does not.
func SetSendRoot(hash [32]byte) {
	hashUnsafe := unsafe.Pointer(&hash[0])
	setGlobalStateBytes32(IDX_SEND_ROOT, hashUnsafe)
}

func GetPositionWithinMessage() uint64 {
	return getGlobalStateU64(IDX_POSITION_WITHIN_MESSAGE)
}

func SetPositionWithinMessage(pos uint64) {
	setGlobalStateU64(IDX_POSITION_WITHIN_MESSAGE, pos)
}

func GetInboxPosition() uint64 {
	return getGlobalStateU64(IDX_INBOX_POSITION)
}
