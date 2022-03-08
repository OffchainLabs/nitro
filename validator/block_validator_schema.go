//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package validator

type lastBlockValidatedDbInfo struct {
	BlockNumber   uint64
	AfterPosition GlobalStatePosition
}

var (
	lastBlockValidatedInfoKey []byte = []byte("_lastBlockValidatedInfo") // contains a rlp encoded lastBlockValidatedDbInfo
)
