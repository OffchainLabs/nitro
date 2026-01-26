// Copyright 2026-2027, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/melwavmio"
)

type relevantTxIndicesFetcher struct{}

func (rf *relevantTxIndicesFetcher) FetchRelevantTxIndices(
	ctx context.Context, parentChainBlockHash common.Hash,
) ([]uint, error) {
	rawTxIndices, err := melwavmio.GetRelevantTxIndices(parentChainBlockHash)
	if err != nil {
		return nil, err
	}
	var txIndices []uint
	if err := rlp.DecodeBytes(rawTxIndices, &txIndices); err != nil {
		return nil, err
	}
	return txIndices, nil
}
