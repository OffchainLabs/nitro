// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package melreplay

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// RELEVANT_TX_INDEXES_PREFIX represents the prefix appended to a blockHash and the hash of the resulting string
// maps to the list of MEL-relevant tx indexes in a parent chain block
const RELEVANT_TX_INDEXES_PREFIX string = "TX_INDEX_DATA"

func RelevantTxIndexesKey(parentChainBlockHash common.Hash) common.Hash {
	return crypto.Keccak256Hash([]byte(RELEVANT_TX_INDEXES_PREFIX), parentChainBlockHash.Bytes())
}
