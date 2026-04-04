// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/bold/containers/fsm"
)

// txByLogFetcher is wrapper around ParentChainReader to implement TransactionByLog method
type txByLogFetcher struct {
	client ParentChainReader
}

func (f *txByLogFetcher) TransactionByLog(ctx context.Context, log *types.Log) (*types.Transaction, error) {
	if log == nil {
		return nil, errors.New("transactionByLog got nil log value")
	}
	tx, _, err := f.client.TransactionByHash(ctx, log.TxHash)
	return tx, err
}

func (m *MessageExtractor) processNextBlock(ctx context.Context, current *fsm.CurrentState[action, FSMState]) (time.Duration, error) {
	processAction, ok := current.SourceEvent.(processNextBlock)
	if !ok {
		return m.config.RetryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
	}
	preState := processAction.melState
	// If the current parent chain block is not safe/finalized we wait till it becomes safe/finalized as determined by the ReadMode
	if m.config.ReadMode != "latest" && preState.ParentChainBlockNumber+1 > m.lastBlockToRead.Load() {
		return m.config.RetryInterval, nil
	}
	parentChainBlock, err := m.logsAndHeadersPreFetcher.getHeaderByNumber(ctx, preState.ParentChainBlockNumber+1)
	if err == nil && parentChainBlock == nil {
		return m.config.RetryInterval, fmt.Errorf("parent chain block %d returned nil without error", preState.ParentChainBlockNumber+1)
	}
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// If the block with the specified number is not found, it likely has not
			// been posted yet to the parent chain, so we can retry
			// without returning an error from the FSM.
			if !m.caughtUp && m.config.ReadMode == "latest" {
				if latestBlk, err := m.parentChainReader.HeaderByNumber(ctx, big.NewInt(rpc.LatestBlockNumber.Int64())); err != nil {
					log.Error("Error fetching LatestBlockNumber from parent chain to determine if mel has caught up", "err", err)
				} else if latestBlk == nil {
					log.Error("Parent chain returned nil header for latest block")
				} else if latestBlk.Number.Uint64()-preState.ParentChainBlockNumber <= 5 { // tolerance of catching up i.e parent chain might have progressed in the time between the above two function calls
					m.caughtUp = true
					close(m.caughtUpChan)
				}
			}
			m.consecutiveNotFound++
			if m.config.StallTolerance > 0 && m.consecutiveNotFound > 2*m.config.StallTolerance {
				return m.config.RetryInterval, fmt.Errorf("MEL block %d not found for %d consecutive attempts, possible parent chain stall", preState.ParentChainBlockNumber+1, m.consecutiveNotFound)
			}
			if m.consecutiveNotFound > m.config.StallTolerance {
				log.Warn("MEL block not found for extended period", "block", preState.ParentChainBlockNumber+1, "consecutiveNotFound", m.consecutiveNotFound, "caughtUp", m.caughtUp)
			}
			return m.config.RetryInterval, nil
		}
		m.consecutiveNotFound = 0
		return m.config.RetryInterval, err
	}
	m.consecutiveNotFound = 0
	if parentChainBlock.ParentHash != preState.ParentChainBlockHash {
		log.Info("MEL detected L1 reorg", "block", preState.ParentChainBlockNumber) // Log level is Info because L1 reorgs are a common occurrence
		return 0, m.fsm.Do(reorgToOldBlock{
			melState: preState,
		})
	}
	// Previous FSM step was a reorg. Rebuild delayed message preimage cache from
	// the rewound state and notify the block validator so it can discard stale work.
	if processAction.prevStepWasReorg {
		if err := preState.RebuildDelayedMsgPreimages(m.melDB.FetchDelayedMessage); err != nil {
			return m.config.RetryInterval, fmt.Errorf("error rebuilding delayed msg preimages after reorg: %w", err)
		}
		m.consecutivePreimageRebuilds = 0
		m.consecutiveNotFound = 0
		if m.blockValidator != nil {
			m.blockValidator.ReorgToBatchCount(preState.BatchCount)
		}
		if err := m.sendReorgNotification(ctx, preState.ParentChainBlockNumber); err != nil {
			return 0, err
		}
	}
	if err = m.logsAndHeadersPreFetcher.fetch(ctx, preState); err != nil {
		return m.config.RetryInterval, err
	}
	postState, msgs, delayedMsgs, batchMetas, err := melextraction.ExtractMessages(
		ctx,
		preState,
		parentChainBlock,
		m.dataProviders,
		m.melDB,
		&txByLogFetcher{m.parentChainReader},
		m.logsAndHeadersPreFetcher,
		m.chainConfig,
	)
	if err != nil {
		if errors.Is(err, mel.ErrDelayedMessagePreimageNotFound) {
			return m.handlePreimageCacheMiss(preState)
		}
		return m.config.RetryInterval, err
	}
	m.consecutivePreimageRebuilds = 0
	// Begin the next FSM state immediately.
	return 0, m.fsm.Do(saveMessages{
		preStateMsgCount: preState.MsgCount,
		postState:        postState,
		messages:         msgs,
		delayedMessages:  delayedMsgs,
		batchMetas:       batchMetas,
	})
}

// handlePreimageCacheMiss attempts to rebuild the delayed message preimage
// cache. Returns (0, nil) on success for immediate retry, or an error if the
// rebuild limit has been reached or the rebuild itself fails.
func (m *MessageExtractor) handlePreimageCacheMiss(preState *mel.State) (time.Duration, error) {
	m.consecutivePreimageRebuilds++
	if m.consecutivePreimageRebuilds >= 3 {
		return m.config.RetryInterval, fmt.Errorf("repeated preimage rebuild at block %d after %d attempts, possible systemic issue", preState.ParentChainBlockNumber, m.consecutivePreimageRebuilds)
	}
	log.Warn("Rebuilding delayed message preimages due to cache miss during extraction", "block", preState.ParentChainBlockNumber, "attempt", m.consecutivePreimageRebuilds)
	if rebuildErr := preState.RebuildDelayedMsgPreimages(m.melDB.FetchDelayedMessage); rebuildErr != nil {
		return m.config.RetryInterval, fmt.Errorf("error rebuilding delayed msg preimages when missing some preimages: %w", rebuildErr)
	}
	// Rebuild succeeded; retry immediately without incrementing the stall counter.
	// The consecutivePreimageRebuilds counter limits repeated rebuilds independently.
	return 0, nil
}
