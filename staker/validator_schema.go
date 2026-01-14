// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package staker

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator"
)

type legacyLastBlockValidatedDbInfo struct {
	BlockNumber   uint64
	BlockHash     common.Hash
	AfterPosition GlobalStatePosition
}

type GlobalStateValidatedInfo struct {
	GlobalState validator.GoGlobalState
	WasmRoots   []common.Hash
}

var (
	lastMELGlobalStateValidatedInfoKey = []byte("_lastMELGlobalStateValidatedInfo") // contains a rlp encoded GlobalStateValidatedInfo of the last validated MEL state
	lastGlobalStateValidatedInfoKey    = []byte("_lastGlobalStateValidatedInfo")    // contains a rlp encoded lastBlockValidatedDbInfo
	legacyLastBlockValidatedInfoKey    = []byte("_lastBlockValidatedInfo")          // LEGACY - contains a rlp encoded lastBlockValidatedDbInfo
)
