// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package staker

import "github.com/ethereum/go-ethereum/common"

type LastBlockValidatedDbInfo struct {
	BlockNumber   uint64
	BlockHash     common.Hash
	AfterPosition GlobalStatePosition
}

var (
	lastBlockValidatedInfoKey = []byte("_lastBlockValidatedInfo") // contains a rlp encoded lastBlockValidatedDbInfo
)
