// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package lifecycle

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	datapostermetrics "github.com/offchainlabs/nitro/arbnode/dataposter/metrics"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Gets latest known or finalized block header (depending on config flag),
// gets the nonce of the dataposter sender and stores it if it has increased.
func UpdateNonce(ctx context.Context, p dataPoster, s *state.LockedInternalState) error {
	var blockNumQuery *big.Int
	if waitForL1Finality(p) {
		blockNumQuery = big.NewInt(int64(rpc.FinalizedBlockNumber))
	}

	header, err := p.Client().HeaderByNumber(ctx, blockNumQuery)
	if err != nil {
		return fmt.Errorf("failed to get the latest or finalized L1 header: %w", err)
	}

	// State is already up-to-date
	if s.LastBlock != nil && arbmath.BigEquals(s.LastBlock, header.Number) {
		return nil
	}

	nonce, err := p.Client().NonceAt(ctx, p.Sender(), header.Number)
	if err != nil {
		if s.LastBlock != nil {
			log.Warn("Failed to get current nonce", "lastBlock", s.LastBlock, "newBlock", header.Number, "err", err)
			return nil
		}
		return err
	}

	// Ignore if nonce hasn't increased.
	if nonce <= s.Nonce {
		// Still update last block number.
		if nonce == s.Nonce {
			s.LastBlock = header.Number
		}
		return nil
	}

	// #nosec G115
	datapostermetrics.LatestFinalizedNonceGauge.Update(int64(nonce))
	log.Info("Data poster transactions confirmed", "previousNonce", s.Nonce, "newNonce", nonce, "previousL1Block", s.LastBlock, "newL1Block", header.Number)
	if len(s.ErrorCount) > 0 {
		for x := s.Nonce; x < nonce; x++ {
			delete(s.ErrorCount, x)
		}
	}

	// We don't prune the most recent transaction in order to ensure that the data poster
	// always has a reference point in its queue of the latest transaction nonce and metadata.
	// nonce > 0 is implied by nonce > p.nonce, so this won't underflow.
	if err := s.Queue.Prune(ctx, nonce-1); err != nil {
		return err
	}

	// We update these two variables together because they should remain in sync even if there's an error.
	s.LastBlock = header.Number
	s.Nonce = nonce
	return nil
}

// Returns the next nonce, its metadata if stored, a bool indicating if the metadata is present, the cumulative weight, and an error if present.
//
// Unlike GetNextNonceAndMeta, this does not call the metadataRetriever if the metadata is not stored in the queue.
func GetNextNonceAndMaybeMeta(ctx context.Context, p dataPoster, s *state.LockedInternalState, thisWeight uint64) (*NonceAndMaybeMeta, error) {
	// Ensure latest finalized block state is available.
	blockNum, err := p.Client().BlockNumber(ctx)
	if err != nil {
		return nil, err
	}
	lastQueueItem, err := s.Queue.FetchLast(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching last element from queue: %w", err)
	}
	if lastQueueItem != nil {
		nextNonce := lastQueueItem.FullTx.Nonce() + 1
		if err := CanPostWithNonce(ctx, p, s, nextNonce, thisWeight); err != nil {
			return nil, err
		}
		return &NonceAndMaybeMeta{
			Nonce:            nextNonce,
			Meta:             lastQueueItem.Meta,
			MetaPresent:      true,
			CumulativeWeight: lastQueueItem.CumulativeWeight(),
		}, nil
	}

	if err := UpdateNonce(ctx, p, s); err != nil {
		if !s.Queue.IsPersistent() && waitForL1Finality(p) {
			return nil, fmt.Errorf("error getting latest finalized nonce (and queue is not persistent): %w", err)
		}
		// Fall back to using a recent block to get the nonce. This is safe because there's nothing in the queue.
		nonceQueryBlock := arbmath.UintToBig(arbmath.SaturatingUSub(blockNum, 1))
		log.Warn("failed to update nonce with queue empty; falling back to using a recent block", "recentBlock", nonceQueryBlock, "err", err)
		nonce, err := p.Client().NonceAt(ctx, p.Sender(), nonceQueryBlock)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce at block %v: %w", nonceQueryBlock, err)
		}
		s.LastBlock = nonceQueryBlock
		s.Nonce = nonce
	}
	return &NonceAndMaybeMeta{
		Nonce:            s.Nonce,
		Meta:             nil,
		MetaPresent:      false,
		CumulativeWeight: s.Nonce,
	}, nil
}

// GetNextNonceAndMeta retrieves generates next nonce, validates that a
// transaction can be posted with that nonce, and fetches "Meta" either last
// queued item (if queue isn't empty) or retrieves with last block.
func GetNextNonceAndMeta(ctx context.Context, p dataPoster) (*NonceAndMeta, error) {
	lockedState := p.InternalState().Lock()
	defer p.InternalState().Unlock()
	nonce, err := GetNextNonceAndMaybeMeta(ctx, p, lockedState, 1)
	if err != nil {
		return nil, err
	}
	meta := nonce.Meta
	if !nonce.MetaPresent {
		meta, err = p.RetrieveMetadata(ctx, lockedState.LastBlock)
	}
	return &NonceAndMeta{
		Nonce: nonce.Nonce,
		Meta:  meta,
	}, err
}

// Does basic check whether posting transaction with specified nonce would
// result in exceeding maximum queue length or maximum transactions in mempool.
func CanPostWithNonce(ctx context.Context, p dataPoster, s *state.LockedInternalState, nextNonce uint64, thisWeight uint64) error {
	// If the queue has reached configured max size, don't post a transaction.
	err := checkQueueLength(ctx, p, s, nextNonce)
	if err != nil {
		return err
	}

	// Check that posting a new transaction won't exceed maximum pending
	// transactions in mempool.
	err = checkMempoolTransactions(ctx, p, nextNonce)
	if err != nil {
		return err
	}

	// Check that posting a new transaction won't exceed maximum pending
	// weight in mempool.
	err = checkMempoolWeight(ctx, p, s, nextNonce, thisWeight)
	if err != nil {
		return err
	}

	return nil
}

// Check the queue against the configured max length.
//
// Returns an error if the queue is full.
func checkQueueLength(ctx context.Context, p dataPoster, s *state.LockedInternalState, nextNonce uint64) error {
	cfg := p.Config()
	if cfg.MaxQueuedTransactions > 0 {
		queueLen, err := s.Queue.Length(ctx)
		if err != nil {
			return fmt.Errorf("getting queue length: %w", err)
		}
		if queueLen >= cfg.MaxQueuedTransactions {
			return fmt.Errorf("posting a transaction with nonce: %d will exceed max allowed dataposter queued transactions: %d, current nonce: %d", nextNonce, cfg.MaxQueuedTransactions, s.Nonce)
		}
	}
	return nil
}

// Check the maximum pending mempool transactions
//
// Returns an error if we are at the limit of pending transactions in mempool.
func checkMempoolTransactions(ctx context.Context, p dataPoster, nextNonce uint64) error {
	cfg := p.Config()
	if cfg.MaxMempoolTransactions > 0 {
		unconfirmedNonce, err := p.Client().NonceAt(ctx, p.Sender(), nil)
		if err != nil {
			return fmt.Errorf("getting nonce of a dataposter sender: %w", err)
		}
		// #nosec G115
		datapostermetrics.LatestUnconfirmedNonceGauge.Update(int64(unconfirmedNonce))
		if nextNonce >= cfg.MaxMempoolTransactions+unconfirmedNonce {
			return fmt.Errorf("%w: transaction nonce: %d, unconfirmed nonce: %d, max mempool size: %d", ErrExceedsMaxMempoolSize, nextNonce, unconfirmedNonce, cfg.MaxMempoolTransactions)
		}
	}
	return nil
}

// Check the maximum pending weight against the configured maximum
//
// Returns an error if we are past the limit.
func checkMempoolWeight(ctx context.Context, p dataPoster, s *state.LockedInternalState, nextNonce uint64, thisWeight uint64) error {
	cfg := p.Config()
	if cfg.MaxMempoolWeight > 0 {
		unconfirmedNonce, err := p.Client().NonceAt(ctx, p.Sender(), nil)
		if err != nil {
			return fmt.Errorf("getting nonce of a dataposter sender: %w", err)
		}
		// #nosec G115
		datapostermetrics.LatestUnconfirmedNonceGauge.Update(int64(unconfirmedNonce))
		if unconfirmedNonce > nextNonce {
			return fmt.Errorf("latest on-chain nonce %v is greater than to next nonce %v", unconfirmedNonce, nextNonce)
		}

		var confirmedWeight uint64
		if unconfirmedNonce > 0 {
			confirmedMeta, err := s.Queue.Get(ctx, unconfirmedNonce-1)
			if err != nil {
				return err
			}
			if confirmedMeta != nil {
				confirmedWeight = confirmedMeta.CumulativeWeight()
			}
		}
		previousTxMeta, err := s.Queue.FetchLast(ctx)
		if err != nil {
			return err
		}
		var previousTxCumulativeWeight uint64
		if previousTxMeta != nil {
			previousTxCumulativeWeight = previousTxMeta.CumulativeWeight()
		}
		previousTxCumulativeWeight = arbmath.MaxInt(previousTxCumulativeWeight, confirmedWeight)
		newCumulativeWeight := previousTxCumulativeWeight + thisWeight

		maxBlobGasPerBlock := uint64(0)
		if p.Config().Post4844Blobs {
			maxBlobGasPerBlock, err = p.ParentChain().MaxBlobGasPerBlock(ctx, nil)
			if err != nil {
				return err
			}
		}
		weightDiff := arbmath.MinInt(newCumulativeWeight-confirmedWeight, (nextNonce-unconfirmedNonce)*maxBlobGasPerBlock/params.BlobTxBlobGasPerBlob)
		if weightDiff > cfg.MaxMempoolWeight {
			return fmt.Errorf("%w: transaction nonce: %d, transaction cumulative weight: %d, unconfirmed nonce: %d, confirmed weight: %d, new mempool weight: %d, max mempool weight: %d", ErrExceedsMaxMempoolSize, nextNonce, newCumulativeWeight, unconfirmedNonce, confirmedWeight, weightDiff, cfg.MaxMempoolWeight)
		}
	}
	return nil
}

func waitForL1Finality(p dataPoster) bool {
	return p.Config().WaitForL1Finality && !p.HeaderReader().IsParentChainArbitrum()
}
