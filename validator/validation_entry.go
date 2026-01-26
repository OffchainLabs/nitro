// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package validator

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/daprovider"
)

type BatchInfo struct {
	Number uint64
	Data   []byte
}

// lint:require-exhaustive-initialization
type ValidationInput struct {
	Id                           uint64
	HasDelayedMsg                bool
	DelayedMsgNr                 uint64
	Preimages                    daprovider.PreimagesMap
	UserWasms                    map[rawdb.WasmTarget]map[common.Hash][]byte
	BatchInfo                    []BatchInfo
	DelayedMsg                   []byte
	StartState                   GoGlobalState
	DebugChain                   bool
	EndParentChainBlockHash      common.Hash
	RelevantTxIndicesByBlockHash map[common.Hash][]uint
}

func CopyPreimagesInto(dest, source daprovider.PreimagesMap) {
	for piType, piMap := range source {
		if dest[piType] == nil {
			dest[piType] = make(map[common.Hash][]byte, len(piMap))
		}
		for hash, preimage := range piMap {
			dest[piType][hash] = preimage
		}
	}
}
