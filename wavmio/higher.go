package wavmio

import "github.com/ethereum/go-ethereum/common"

const INITIAL_CAPACITY = 1024

func withSizeRetry(f func([]byte) bool) []byte {
	buf := make([]byte, 0, INITIAL_CAPACITY)
	for {
		success := f(buf)
		if success {
			return buf
		}
		buf = make([]byte, 0, cap(buf)*2)
	}
}

func GetLastBlockHash() common.Hash {
	return getLastBlockHash()
}

func ReadInboxMessage() []byte {
	return withSizeRetry(func(buf []byte) bool {
		return readInboxMessage(buf)
	})
}

func AdvanceInboxMessage() {
	advanceInboxMessage()
}

func ResolvePreImage(hash common.Hash) []byte {
	return withSizeRetry(func(buf []byte) bool {
		return resolvePreImage(hash, buf)
	})
}

func SetLastBlockHash(hash [32]byte) {
	setLastBlockHash(hash)
}
