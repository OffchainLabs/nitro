// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/transactionfeed"
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
	transactionFeedServer    transactionBroadcaster
}

var _ BlockSequencingHooks = (*FullSequencingHooks)(nil)

func MakeSequencingHooks(
	items []txQueueItem,
	maxSequencedTxsSize int,
	txFilter arbos.TxFilter,
	transactionFeedServer transactionBroadcaster,
) *FullSequencingHooks {
	res := &FullSequencingHooks{
		queueItems:               items,
		sequencedQueueItemsCount: 0,
		sequencedTxsSizeSoFar:    0,
		maxSequencedTxsSize:      maxSequencedTxsSize,
		txFilter:                 txFilter,
		blockFilter:              nil, // only used in testing
		transactionFeedServer:    transactionFeedServer,
	}
	return res
}

// makeZeroTxSizeSequencingHooks creates hooks that include all transactions in
// a block regardless of size: every queue item has tx size zero and the size
// limit is explicitly unlimited, so the limit check in NextTxToSequence can
// never trip.
func makeZeroTxSizeSequencingHooks(
	txes types.Transactions,
	txFilter arbos.TxFilter,
	blockFilter arbos.BlockFilter,
	transactionFeedServer transactionBroadcaster,
) *FullSequencingHooks {
	var items []txQueueItem
	for _, tx := range txes {
		items = append(items, txQueueItem{
			tx: tx,
		})
	}
	hooks := MakeSequencingHooks(items, math.MaxInt, txFilter, transactionFeedServer)
	hooks.blockFilter = blockFilter
	return hooks
}

// MakeResequencingHooks creates filterless, size-unlimited hooks for re-sequencing reorged txs.
func MakeResequencingHooks(txes types.Transactions, transactionFeedServer transactionBroadcaster) BlockSequencingHooks {
	return makeZeroTxSizeSequencingHooks(txes, nil, nil, transactionFeedServer)
}

// MakeZeroTxSizeSequencingHooksForTesting creates sequencing hooks for testing with tx size always zero.
// This allows all transactions to be included in a block regardless of size.
func MakeZeroTxSizeSequencingHooksForTesting(
	txes types.Transactions,
	txFilter arbos.TxFilter,
	blockFilter arbos.BlockFilter,
) *FullSequencingHooks {
	return makeZeroTxSizeSequencingHooks(txes, txFilter, blockFilter, nil)
}

func (s *FullSequencingHooks) SequencedTxes() ([]TxResult, error) {
	// This is not supposed to happen, if so we have a bug
	if len(s.txErrors) > s.sequencedQueueItemsCount {
		return nil, fmt.Errorf("FullSequencingHooks: more tx results than sequenced txs. txErrors: %d, sequencedQueueItemsCount: %d", len(s.txErrors), s.sequencedQueueItemsCount)
	}
	res := make([]TxResult, 0, len(s.txErrors))
	for i := range s.txErrors {
		res = append(res, TxResult{
			Tx:          s.queueItems[i].tx,
			Err:         s.txErrors[i],
			Timeboosted: s.queueItems[i].isTimeboosted,
		})
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

func (s *FullSequencingHooks) TxAccepted(header *types.Header, tx *types.Transaction, receipt *types.Receipt, collectTips bool) {
	if s.transactionFeedServer == nil {
		return
	}
	msg, err := transactionfeed.BuildFeedMessage(header, tx, receipt, collectTips)
	if err != nil {
		log.Error("Transaction feed: failed to build message", "block", header.Number, "err", err)
		return
	}
	s.transactionFeedServer.BroadcastTransaction(msg)
}

// NextTxToSequence returns the next transaction to be included in the block, or nil if there are no more transactions to include.
// It will skip transactions that would cause the total size of included transactions to exceed maxSequencedTxsSize.
func (s *FullSequencingHooks) NextTxToSequence() (*types.Transaction, *arbitrum_types.ConditionalOptions, error) {
	for {
		// This is not supposed to happen, if so we have a bug
		if len(s.txErrors) != s.sequencedQueueItemsCount {
			return nil, nil, fmt.Errorf("FullSequencingHooks: NextTxToSequence detected out of order request to sequence tx. txErrors: %d, sequencedQueueItemsCount: %d", len(s.txErrors), s.sequencedQueueItemsCount)
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
