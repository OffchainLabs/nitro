// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"fmt"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
)

type FullSequencingHooks struct {
	queueItems               []txQueueItem
	sequencedQueueItemsCount int
	sequencedTxsSizeSoFar    int
	maxSequencedTxsSize      int
	txErrors                 []error
	txFilter                 arbos.TxFilter
	blockFilter              arbos.BlockFilter
	txSizeLimitReached       bool
}

var _ BlockSequencingHooks = (*FullSequencingHooks)(nil)

func MakeSequencingHooks(
	items []txQueueItem,
	maxSequencedTxsSize int,
	txFilter arbos.TxFilter,
) *FullSequencingHooks {
	res := &FullSequencingHooks{
		queueItems:               items,
		sequencedQueueItemsCount: 0,
		sequencedTxsSizeSoFar:    0,
		maxSequencedTxsSize:      maxSequencedTxsSize,
		txFilter:                 txFilter,
		blockFilter:              nil, // only used in testing
	}
	return res
}

// MakeZeroTxSizeSequencingHooksForTesting creates sequencing hooks for testing with tx size always zero.
// This allows all transactions to be included in a block regardless of size.
func MakeZeroTxSizeSequencingHooksForTesting(
	txes types.Transactions,
	txFilter arbos.TxFilter,
	blockFilter arbos.BlockFilter,
) *FullSequencingHooks {
	var items []txQueueItem
	for _, tx := range txes {
		items = append(items, txQueueItem{
			tx: tx,
		})
	}
	hooks := MakeSequencingHooks(items, 0, txFilter)
	hooks.blockFilter = blockFilter
	return hooks
}

func (s *FullSequencingHooks) SequencedTxes() ([]TxResult, error) {
	res := make([]TxResult, 0, len(s.txErrors))
	for i, txErr := range s.txErrors {
		tx, err := s.SequencedTx(i)
		if err != nil {
			return nil, err
		}
		res = append(res, TxResult{Tx: tx, Err: txErr})
	}
	return res, nil
}

func (s *FullSequencingHooks) GetTxErrors() []error {
	return s.txErrors
}

func (s *FullSequencingHooks) TxSucceeded() {
	s.txErrors = append(s.txErrors, nil)
}

func (s *FullSequencingHooks) TxFailed(err error) {
	if len(s.txErrors) >= s.sequencedQueueItemsCount {
		log.Error("TxFailed called but entry already exists", "existingErr", s.txErrors[len(s.txErrors)-1], "newErr", err)
	}
	s.txErrors = append(s.txErrors, err)
}

// NextTxToSequence returns the next transaction to be included in the block, or nil if there are no more transactions to include.
// It will skip transactions that would cause the total size of included transactions to exceed maxSequencedTxsSize.
func (s *FullSequencingHooks) NextTxToSequence() (*types.Transaction, *arbitrum_types.ConditionalOptions, error) {
	for {
		// This is not supposed to happen, if so we have a bug
		if len(s.txErrors) != s.sequencedQueueItemsCount {
			return nil, nil, fmt.Errorf("FullSequencingHooks: GetNextTx detected out of order request to sequence tx. hookTxErrors: %d, nextTxIdToBeSequenced: %d", len(s.txErrors), s.sequencedQueueItemsCount)
		}
		if s.sequencedQueueItemsCount > 0 && s.txErrors[s.sequencedQueueItemsCount-1] == nil {
			s.sequencedTxsSizeSoFar += s.queueItems[s.sequencedQueueItemsCount-1].txSize
		}
		if s.sequencedQueueItemsCount >= len(s.queueItems) {
			return nil, nil, nil
		}
		if s.sequencedTxsSizeSoFar+s.queueItems[s.sequencedQueueItemsCount].txSize > s.maxSequencedTxsSize {
			s.sequencedQueueItemsCount += 1
			s.TxFailed(core.ErrGasLimitReached)
			s.txSizeLimitReached = true
		} else {
			s.sequencedQueueItemsCount += 1
			break
		}
	}
	return s.queueItems[s.sequencedQueueItemsCount-1].tx, s.queueItems[s.sequencedQueueItemsCount-1].options, nil
}

func (s *FullSequencingHooks) CanDiscardTx() bool {
	return true
}

func (s *FullSequencingHooks) SupportsGroupRollback() bool {
	return true
}

func (s *FullSequencingHooks) SequencedTx(txId int) (*types.Transaction, error) {
	// This is not supposed to happen, if so we have a bug
	if txId > s.sequencedQueueItemsCount {
		return nil, fmt.Errorf("transaction queried for was not scheduled by the FullSequencingHooks. txId: %d, sequencedCount: %d", txId, s.sequencedQueueItemsCount)
	}
	return s.queueItems[txId].tx, nil
}

func (s *FullSequencingHooks) PreTxFilter(config *params.ChainConfig, header *types.Header, db *state.StateDB, a *arbosState.ArbosState, transaction *types.Transaction, options *arbitrum_types.ConditionalOptions, address common.Address, info *arbos.L1Info, positionInBlock int) error {
	if s.txFilter != nil {
		return s.txFilter.PreTxFilter(config, header, db, a, transaction, options, address, info, positionInBlock)
	}
	return nil
}

func (s *FullSequencingHooks) PostTxFilter(header *types.Header, db *state.StateDB, a *arbosState.ArbosState, transaction *types.Transaction, address common.Address, u uint64, result *core.ExecutionResult, positionInBlock int) error {
	if s.txFilter != nil {
		return s.txFilter.PostTxFilter(header, db, a, transaction, address, u, result, positionInBlock)
	}
	return nil
}

func (s *FullSequencingHooks) BlockFilter(header *types.Header, db *state.StateDB, transactions types.Transactions, receipts types.Receipts) error {
	if s.blockFilter != nil {
		return s.blockFilter.BlockFilter(header, db, transactions, receipts)
	}
	return nil
}
