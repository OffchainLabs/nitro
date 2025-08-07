// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package messageextraction

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbos/storage"
)

type Extraction struct {
	backingStorage      *storage.Storage
	parentChainBlockNum storage.StorageBackedUint64
}

func OpenExtraction(backingStorage *storage.Storage) *Extraction {
	return &Extraction{
		backingStorage.WithoutCache(),
		backingStorage.OpenStorageBackedUint64(0),
	}
}

func (e *Extraction) MELStateHash(blockNum uint64) (common.Hash, error) {
	return e.backingStorage.GetByUint64(blockNum)
}

func (e *Extraction) RecordMELStateHash(parentChainBlockNum uint64, melStateHash common.Hash) error {
	if err := e.backingStorage.SetByUint64(parentChainBlockNum, melStateHash); err != nil {
		return err
	}
	return e.parentChainBlockNum.Set(parentChainBlockNum)
}

func (e *Extraction) RunExtractionAlgorithm(
	startMelState *mel.State,
	parentChainBlockHeader *types.Header,
	melDataProvider melextraction.MELDataProvider,
) error {
	postState, _, _, _, err := melextraction.ExtractMessages(
		context.Background(),
		startMelState,
		parentChainBlockHeader,
		nil, // TODO: Provide da readers here.
		melDataProvider,
		melDataProvider,
		melDataProvider,
	)
	if err != nil {
		return fmt.Errorf("failed to run extraction algorithm: %w", err)
	}
	return e.RecordMELStateHash(postState.ParentChainBlockNumber, postState.Hash())
}
