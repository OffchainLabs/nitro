// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build wasm
// +build wasm

package wavmio

import (
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
const IDX_ESPRESSO_HEIGHT = 2

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

func StubInit() {}

func StubFinal() {}

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

func ReadHotShotCommitment(h uint64) (commitment [32]byte) {
	readHotShotCommitment(h, unsafe.Pointer(&commitment))
	return commitment
}

func IsHotShotLive(l1Height uint64) bool {
	return isHotShotLive(l1Height) > 0
}

func GetEspressoHeight() uint64 {
	return getGlobalStateU64(IDX_ESPRESSO_HEIGHT)
}

func SetEspressoHeight(h uint64) {
	setGlobalStateU64(IDX_ESPRESSO_HEIGHT, h)
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
	return readBuffer(func(offset uint32, buf unsafe.Pointer) uint32 {
		hashUnsafe := unsafe.Pointer(&hash[0])
		return resolveTypedPreimage(uint32(ty), hashUnsafe, offset, buf)
	}), nil
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
