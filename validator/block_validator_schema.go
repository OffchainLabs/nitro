// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import "github.com/ethereum/go-ethereum/common"

type lastBlockValidatedDbInfo struct {
	BlockNumber   uint64
	BlockHash     common.Hash
	AfterPosition GlobalStatePosition
}

// Not stored in DB but stored in local and redis state trackers
type validationStatus struct {
	PrevHash    common.Hash
	BlockHash   common.Hash
	Validated   bool
	EndPosition GlobalStatePosition
}

var (
	lastBlockValidatedInfoKey []byte = []byte("_lastBlockValidatedInfo") // contains a rlp encoded lastBlockValidatedDbInfo
)
