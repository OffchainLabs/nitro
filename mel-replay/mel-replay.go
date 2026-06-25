// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package melreplay

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// RELEVANT_TX_INDEXES_PREFIX represents the prefix appended to a blockHash and the hash of the resulting string
// maps to the list of MEL-relevant tx indexes in a parent chain block
const RELEVANT_TX_INDEXES_PREFIX string = "TX_INDEX_DATA"

func RelevantTxIndexesKey(parentChainBlockHash common.Hash) common.Hash {
	return crypto.Keccak256Hash([]byte(RELEVANT_TX_INDEXES_PREFIX), parentChainBlockHash.Bytes())
}

type PreimageResolver interface {
	ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error)
}

type typeBasedPreimageResolver struct {
	ty           arbutil.PreimageType
	preimagesMap daprovider.PreimagesMap
}

func NewTypeBasedPreimageResolver(ty arbutil.PreimageType, preimagesMap daprovider.PreimagesMap) PreimageResolver {
	return &typeBasedPreimageResolver{ty, preimagesMap}
}

func (t *typeBasedPreimageResolver) ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	if preimageType != t.ty {
		return nil, fmt.Errorf("unsupported preimageType: %d, want: %d", preimageType, t.ty)
	}
	if targetMap, ok := t.preimagesMap[preimageType]; ok {
		if preimage, ok := targetMap[hash]; ok {
			return preimage, nil
		}
	}
	return nil, fmt.Errorf("preimage not found for hash: %v", hash)
}
