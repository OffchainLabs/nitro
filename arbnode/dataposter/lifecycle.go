// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dataposter

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/dataposter/metrics"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Gets latest known or finalized block header (depending on config flag),
// gets the nonce of the dataposter sender and stores it if it has increased.
func (p *DataPoster) updateNonce(ctx context.Context, s *state.LockedInternalState) error {
	var blockNumQuery *big.Int
	if p.waitForL1Finality() {
		blockNumQuery = big.NewInt(int64(rpc.FinalizedBlockNumber))
	}
	header, err := p.client.HeaderByNumber(ctx, blockNumQuery)
	if err != nil {
		return fmt.Errorf("failed to get the latest or finalized L1 header: %w", err)
	}
	if s.LastBlock != nil && arbmath.BigEquals(s.LastBlock, header.Number) {
		return nil
	}
	nonce, err := p.client.NonceAt(ctx, p.Sender(), header.Number)
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

// Updates dataposter balance to balance at pending block.
func (p *DataPoster) updateBalance(ctx context.Context, s *state.LockedInternalState) error {
	// Use the pending (represented as -1) balance because we're looking at batches we'd post,
	// so we want to see how much gas we could afford with our pending state.
	balance, err := p.client.BalanceAt(ctx, p.Sender(), big.NewInt(-1))
	if err != nil {
		return err
	}
	s.Balance = balance
	return nil
}

const maxConsecutiveIntermittentErrors = 20

func (p *DataPoster) maybeLogError(err error, s *state.LockedInternalState, tx *storage.QueuedTransaction, msg string) {
	nonce := tx.FullTx.Nonce()
	if err == nil {
		delete(s.ErrorCount, nonce)
		return
	}
	logLevel := log.Error
	isStorageRace := errors.Is(err, storage.ErrStorageRace)
	isFutureReplacePending := strings.Contains(err.Error(), legacypool.ErrFutureReplacePending.Error())
	isNonceTooHigh := strings.Contains(err.Error(), core.ErrNonceTooHigh.Error())
	if isStorageRace || isFutureReplacePending || isNonceTooHigh {
		s.ErrorCount[nonce]++
		if s.ErrorCount[nonce] <= maxConsecutiveIntermittentErrors {
			if isStorageRace {
				logLevel = log.Debug
			} else {
				logLevel = log.Info
			}
		} else if isStorageRace {
			logLevel = log.Warn
		}
	} else {
		delete(s.ErrorCount, nonce)
	}
	logLevel(msg, "err", err, "nonce", nonce, "feeCap", tx.FullTx.GasFeeCap(), "tipCap", tx.FullTx.GasTipCap(), "blobFeeCap", tx.FullTx.BlobGasFeeCap(), "gas", tx.FullTx.Gas())
}

var ErrExceedsMaxMempoolSize = errors.New("posting this transaction will exceed max mempool size")

// Does basic check whether posting transaction with specified nonce would
// result in exceeding maximum queue length or maximum transactions in mempool.
func (p *DataPoster) canPostWithNonce(ctx context.Context, s *state.LockedInternalState, nextNonce uint64, thisWeight uint64) error {
	cfg := p.config()
	// If the queue has reached configured max size, don't post a transaction.
	if cfg.MaxQueuedTransactions > 0 {
		queueLen, err := s.Queue.Length(ctx)
		if err != nil {
			return fmt.Errorf("getting queue length: %w", err)
		}
		if queueLen >= cfg.MaxQueuedTransactions {
			return fmt.Errorf("posting a transaction with nonce: %d will exceed max allowed dataposter queued transactions: %d, current nonce: %d", nextNonce, cfg.MaxQueuedTransactions, s.Nonce)
		}
	}
	// Check that posting a new transaction won't exceed maximum pending
	// transactions in mempool.
	if cfg.MaxMempoolTransactions > 0 {
		unconfirmedNonce, err := p.client.NonceAt(ctx, p.Sender(), nil)
		if err != nil {
			return fmt.Errorf("getting nonce of a dataposter sender: %w", err)
		}
		// #nosec G115
		datapostermetrics.LatestUnconfirmedNonceGauge.Update(int64(unconfirmedNonce))
		if nextNonce >= cfg.MaxMempoolTransactions+unconfirmedNonce {
			return fmt.Errorf("%w: transaction nonce: %d, unconfirmed nonce: %d, max mempool size: %d", ErrExceedsMaxMempoolSize, nextNonce, unconfirmedNonce, cfg.MaxMempoolTransactions)
		}
	}
	// Check that posting a new transaction won't exceed maximum pending
	// weight in mempool.
	if cfg.MaxMempoolWeight > 0 {
		unconfirmedNonce, err := p.client.NonceAt(ctx, p.Sender(), nil)
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
		if p.config().Post4844Blobs {
			maxBlobGasPerBlock, err = p.parentChain.MaxBlobGasPerBlock(ctx, nil)
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

func (p *DataPoster) waitForL1Finality() bool {
	return p.config().WaitForL1Finality && !p.headerReader.IsParentChainArbitrum()
}

// Returns the next nonce, its metadata if stored, a bool indicating if the metadata is present, the cumulative weight, and an error if present.
// Unlike GetNextNonceAndMeta, this does not call the metadataRetriever if the metadata is not stored in the queue.
func (p *DataPoster) getNextNonceAndMaybeMeta(ctx context.Context, s *state.LockedInternalState, thisWeight uint64) (uint64, []byte, bool, uint64, error) {
	// Ensure latest finalized block state is available.
	blockNum, err := p.client.BlockNumber(ctx)
	if err != nil {
		return 0, nil, false, 0, err
	}
	lastQueueItem, err := s.Queue.FetchLast(ctx)
	if err != nil {
		return 0, nil, false, 0, fmt.Errorf("fetching last element from queue: %w", err)
	}
	if lastQueueItem != nil {
		nextNonce := lastQueueItem.FullTx.Nonce() + 1
		if err := p.canPostWithNonce(ctx, s, nextNonce, thisWeight); err != nil {
			return 0, nil, false, 0, err
		}
		return nextNonce, lastQueueItem.Meta, true, lastQueueItem.CumulativeWeight(), nil
	}

	if err := p.updateNonce(ctx, s); err != nil {
		if !s.Queue.IsPersistent() && p.waitForL1Finality() {
			return 0, nil, false, 0, fmt.Errorf("error getting latest finalized nonce (and queue is not persistent): %w", err)
		}
		// Fall back to using a recent block to get the nonce. This is safe because there's nothing in the queue.
		nonceQueryBlock := arbmath.UintToBig(arbmath.SaturatingUSub(blockNum, 1))
		log.Warn("failed to update nonce with queue empty; falling back to using a recent block", "recentBlock", nonceQueryBlock, "err", err)
		nonce, err := p.client.NonceAt(ctx, p.Sender(), nonceQueryBlock)
		if err != nil {
			return 0, nil, false, 0, fmt.Errorf("failed to get nonce at block %v: %w", nonceQueryBlock, err)
		}
		s.LastBlock = nonceQueryBlock
		s.Nonce = nonce
	}
	return s.Nonce, nil, false, s.Nonce, nil
}

// GetNextNonceAndMeta retrieves generates next nonce, validates that a
// transaction can be posted with that nonce, and fetches "Meta" either last
// queued item (if queue isn't empty) or retrieves with last block.
func (p *DataPoster) GetNextNonceAndMeta(ctx context.Context) (uint64, []byte, error) {
	lockedState := p.internalState.Lock()
	defer p.internalState.Unlock()
	nonce, meta, hasMeta, _, err := p.getNextNonceAndMaybeMeta(ctx, lockedState, 1)
	if err != nil {
		return 0, nil, err
	}
	if !hasMeta {
		meta, err = p.metadataRetriever(ctx, lockedState.LastBlock)
	}
	return nonce, meta, err
}
