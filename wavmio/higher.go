//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package wavmio

import "github.com/ethereum/go-ethereum/common"

const INITIAL_CAPACITY = 128
const QUERY_SIZE = 32

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
	getLastBlockHash(hash[:])
	return
}

func ReadInboxMessage() []byte {
	return readBuffer(func(offset uint32, buf []byte) uint32 {
		return readInboxMessage(offset, buf)
	})
}

func AdvanceInboxMessage() {
	advanceInboxMessage()
}

func ResolvePreImage(hash common.Hash) []byte {
	return readBuffer(func(offset uint32, buf []byte) uint32 {
		return resolvePreImage(hash[:], offset, buf)
	})
}

func SetLastBlockHash(hash [32]byte) {
	setLastBlockHash(hash[:])
}

func GetPositionWithinMessage() uint64 {
	return getPositionWithinMessage()
}

func SetPositionWithinMessage(pos uint64) {
	setPositionWithinMessage(pos)
}
