//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

//go:build js
// +build js

package wavmio

import "github.com/ethereum/go-ethereum/common"

const INITIAL_CAPACITY = 128
const QUERY_SIZE = 32

const IDX_LAST_BLOCKHASH = 0
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

func GetLastBlockHash() (hash common.Hash) {
	getGlobalStateBytes32(IDX_LAST_BLOCKHASH, hash[:])
	return
}

func ReadInboxMessage(msgNum uint64) []byte {
	return readBuffer(func(offset uint32, buf []byte) uint32 {
		return readInboxMessage(msgNum, offset, buf)
	})
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

func ResolvePreImage(hash common.Hash) []byte {
	return readBuffer(func(offset uint32, buf []byte) uint32 {
		return resolvePreImage(hash[:], offset, buf)
	})
}

func SetLastBlockHash(hash [32]byte) {
	setGlobalStateBytes32(IDX_LAST_BLOCKHASH, hash[:])
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
