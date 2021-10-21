//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

//go:build !js
// +build !js

package wavmio

import "github.com/ethereum/go-ethereum/common"

func GetLastBlockHash() (hash common.Hash) {
	return common.Hash{} // needed to avoid linter problems
}

func ReadInboxMessage() []byte {
	panic("not on wavm platform")
}

func ReadDelayedInboxMessage(seqNum uint64) []byte {
	panic("not on wavm platform")
}

func AdvanceInboxMessage() {
	panic("not on wavm platform")
}

func ResolvePreImage(hash common.Hash) []byte {
	panic("not on wavm platform")
}

func SetLastBlockHash(hash [32]byte) {
	panic("not on wavm platform")
}

func GetPositionWithinMessage() uint64 {
	panic("not on wavm platform")
}

func SetPositionWithinMessage(pos uint64) {
	panic("not on wavm platform")
}

func GetInboxPosition() uint64 {
	panic("not on wavm platform")
}
