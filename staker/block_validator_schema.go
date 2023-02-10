// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator"
)

// Todo: we could create an upgrade scheme for moving from lastMessageValidated to lastBlockValidated
// not a must, since even without this index, we'll start validation from last assertion made
// the other option is to remove lastBlockValidated* from code

// type legacyLastBlockValidatedDbInfo struct {
// 	BlockNumber   uint64
// 	BlockHash     common.Hash
// 	AfterPosition GlobalStatePosition
// }

type GlobalStateValidatedInfo struct {
	GlobalState validator.GoGlobalState
	WasmRoots   []common.Hash
}

var (
	lastGlobalStateValidatedInfoKey = []byte("_lastGlobalStateValidatedInfo") // contains a rlp encoded lastBlockValidatedDbInfo
	// legacyLastBlockValidatedInfoKey = []byte("_lastBlockValidatedInfo")       // contains a rlp encoded lastBlockValidatedDbInfo
)
