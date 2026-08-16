// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"errors"
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

// txNotFinalized marks a pulled tx whose result the block processor has not reported yet.
// A surviving marker is a bug (every pulled tx is a user tx, and the block processor reports
// every user tx's result); NextTxToSequence and the execution engine fail the block on one.
var txNotFinalized = errors.New("tx was not finalized by the block processor")

// sequencedTx pairs a pulled queue item with its sequencing result.
type sequencedTx struct {
	queueItem txQueueItem
	err       error
}

type FullSequencingHooks struct {
	// fetcher is the block's candidate source.
	fetcher nextTxFetcher
	// sequencedTxs records the items pulled from the fetcher so far, in order, with their results.
	sequencedTxs          []sequencedTx
	sequencedTxsSizeSoFar int
	maxSequencedTxsSize   int
	txFilter              arbos.TxFilter
	blockFilter           arbos.BlockFilter // only used in testing
	transactionFeedServer transactionBroadcaster
}

var _ BlockSequencingHooks = (*FullSequencingHooks)(nil)

func MakeSequencingHooks(
	fetcher nextTxFetcher,
	maxSequencedTxsSize int,
	txFilter arbos.TxFilter,
	transactionFeedServer transactionBroadcaster,
) *FullSequencingHooks {
	return &FullSequencingHooks{
		fetcher:               fetcher,
		maxSequencedTxsSize:   maxSequencedTxsSize,
		txFilter:              txFilter,
		transactionFeedServer: transactionFeedServer,
	}
}

// makeZeroTxSizeSequencingHooks creates hooks that include all transactions in
// a block regardless of size: every queue item has tx size zero and the size
// limit is explicitly unlimited, so the fetcher's size check can never trip.
func makeZeroTxSizeSequencingHooks(
	txes types.Transactions,
	txFilter arbos.TxFilter,
	blockFilter arbos.BlockFilter,
	transactionFeedServer transactionBroadcaster,
) *FullSequencingHooks {
	var items []txQueueItem
	for _, tx := range txes {
		items = append(items, newBaseTxQueueItem(tx))
	}
	hooks := MakeSequencingHooks(&fixedTxFetcher{items: items}, math.MaxInt, txFilter, transactionFeedServer)
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

func (s *FullSequencingHooks) SequencedTxes() []TxResult {
	res := make([]TxResult, 0, len(s.sequencedTxs))
	for _, st := range s.sequencedTxs {
		res = append(res, TxResult{
			Tx:          st.queueItem.tx,
			Err:         st.err,
			Timeboosted: st.queueItem.isTimeboosted,
		})
	}
	return res
}

func (s *FullSequencingHooks) TxSucceeded() {
	if s.setLastTxResult(nil) {
		queueItem := s.sequencedTxs[len(s.sequencedTxs)-1].queueItem
		s.fetcher.OnTxInclusion(queueItem)
		// Only successful txs consume the block's size budget.
		s.sequencedTxsSizeSoFar += queueItem.txSize
	}
}

func (s *FullSequencingHooks) TxFailed(err error) {
	s.setLastTxResult(err)
}

// setLastTxResult records the result of the last pulled tx, reporting whether it was recorded.
func (s *FullSequencingHooks) setLastTxResult(err error) bool {
	if len(s.sequencedTxs) == 0 {
		log.Error("setLastTxResult called before any tx was sequenced", "err", err)
		return false
	}
	last := &s.sequencedTxs[len(s.sequencedTxs)-1]
	switch {
	case errors.Is(last.err, txNotFinalized):
		last.err = err
	case last.err == nil && err != nil:
		// A late failure (e.g. a group rollback) overrides an earlier success; keeping the
		// success would leave a rolled-back tx in the block message.
		last.err = err
	default:
		log.Error("setLastTxResult called twice", "oldErr", last.err, "newErr", err)
		return false
	}
	return true
}

func (s *FullSequencingHooks) TxAccepted(header *types.Header, tx *types.Transaction, receipt *types.Receipt) {
	if s.transactionFeedServer == nil {
		return
	}

	var pgaRound uint64
	if pgaOrderer, ok := s.fetcher.(*pgaTxOrderer); ok {
		pgaRound = pgaOrderer.CurrentRound()
	}

	msg, err := transactionfeed.BuildFeedMessage(header, tx, receipt, pgaRound)
	if err != nil {
		log.Error("Transaction feed: failed to build message", "block", header.Number, "err", err)
		return
	}
	s.transactionFeedServer.BroadcastTransaction(msg)
}

// NextTxToSequence returns the next transaction to include in the block, or nil when the block is done.
// The fetcher decides how to handle a tx too big for the remaining size or gas budget.
func (s *FullSequencingHooks) NextTxToSequence(statedb *state.StateDB, blockGasLeft uint64) (*types.Transaction, *arbitrum_types.ConditionalOptions, error) {
	// This is not supposed to happen, if so we have a bug
	if n := len(s.sequencedTxs); n > 0 && errors.Is(s.sequencedTxs[n-1].err, txNotFinalized) {
		return nil, nil, fmt.Errorf("NextTxToSequence called before the block processor reported tx %s's result", s.sequencedTxs[n-1].queueItem.tx.Hash())
	}
	item, ok := s.fetcher.NextQueueItem(statedb, s.maxSequencedTxsSize-s.sequencedTxsSizeSoFar, blockGasLeft)
	if !ok {
		return nil, nil, nil
	}
	s.sequencedTxs = append(s.sequencedTxs, sequencedTx{queueItem: item, err: txNotFinalized})
	return item.tx, item.options, nil
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
