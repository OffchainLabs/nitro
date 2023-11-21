// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

package wavmio

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

const INITIAL_CAPACITY = 128
const QUERY_SIZE = 32

// bytes32
const IDX_LAST_BLOCKHASH = 0
const IDX_SEND_ROOT = 1
const IDX_HOTSHOT_HEADER = 2

// u64
const IDX_INBOX_POSITION = 0
const IDX_POSITION_WITHIN_MESSAGE = 1

func readBuffer(f func(uint32, []byte) uint32) []byte {
	buf := make([]byte, 0, INITIAL_CAPACITY)
	offset := 0
	for {
		if len(buf) < offset+QUERY_SIZE {
			buf = append(buf, make([]byte, offset+QUERY_SIZE-len(buf))...)
		}
		read := f(uint32(offset), buf[offset:(offset+QUERY_SIZE)])
		offset += int(read)
		if read < QUERY_SIZE {
			buf = buf[:offset]
			return buf
		}
	}
}

func readBufferHotShot(f func(uint32, []byte) uint32) []byte {
	buf := make([]byte, 0, INITIAL_CAPACITY)
	offset := 0
	for {
		read := f(uint32(offset), buf[offset:(offset+QUERY_SIZE)])
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
	getGlobalStateBytes32(IDX_LAST_BLOCKHASH, hash[:])
	return
}

func ReadInboxMessage(msgNum uint64) []byte {
	return readBuffer(func(offset uint32, buf []byte) uint32 {
		return readInboxMessage(msgNum, offset, buf)
	})
}

func ReadHotShotHeader(seqNum uint64) (header [32]byte) {
	readHotShotHeader(seqNum, header[:])
	return
}

func ReadDelayedInboxMessage(seqNum uint64) []byte {
	return readBuffer(func(offset uint32, buf []byte) uint32 {
		return readDelayedInboxMessage(seqNum, offset, buf)
	})
}

func AdvanceInboxMessage() {
	pos := getGlobalStateU64(IDX_INBOX_POSITION)
	setGlobalStateU64(IDX_INBOX_POSITION, pos+1)
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	return readBuffer(func(offset uint32, buf []byte) uint32 {
		return resolveTypedPreimage(uint8(ty), hash[:], offset, buf)
	}), nil
}

func SetLastBlockHash(hash [32]byte) {
	setGlobalStateBytes32(IDX_LAST_BLOCKHASH, hash[:])
}

func SetHotShotHeader(header []byte) {
	setGlobalStateBytes32(IDX_HOTSHOT_HEADER, header[:])
}

// Note: if a GetSendRoot is ever modified, the validator will need to fill in the previous send root, which it currently does not.
func SetSendRoot(hash [32]byte) {
	setGlobalStateBytes32(IDX_SEND_ROOT, hash[:])
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
