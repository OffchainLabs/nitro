// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
)

// nextTxFetcher supplies a block's tx candidates to the sequencing hooks one at a time.
type nextTxFetcher interface {
	// NextQueueItem yields the next block candidate, reporting false on exhaustion.
	NextQueueItem(statedb *state.StateDB, remainingBlockSize int) (txQueueItem, bool)

	// OnTxInclusion notifies the orderer that the last yielded tx made it into the block.
	OnTxInclusion(queueItem txQueueItem)
}

// fixedTxFetcher yields a pre-set candidate list.
type fixedTxFetcher struct {
	items     []txQueueItem
	exhausted []txQueueItem

	sizeLimited bool
}

var _ nextTxFetcher = (*fixedTxFetcher)(nil)

func (f *fixedTxFetcher) NextQueueItem(statedb *state.StateDB, remainingBlockSize int) (txQueueItem, bool) {
	if len(f.items) == 0 {
		return txQueueItem{}, false
	}
	item := f.items[0]
	f.items = f.items[1:]

	// If the tx is too big for the remaining block size, we exhaust it and continue to the next one.
	if item.txSize > remainingBlockSize {
		if !f.sizeLimited {
			dataLimitedBlocksCounter.Inc(1)
			f.sizeLimited = true
		}
		f.exhausted = append(f.exhausted, item)
		return f.NextQueueItem(statedb, remainingBlockSize)
	}

	return item, true
}

// OnTxInclusion is a no-op: the fixed fetcher doesn't react to inclusions.
func (f *fixedTxFetcher) OnTxInclusion(queueItem txQueueItem) {}

// TakeRemaining removes and returns the not-yet-yielded candidates.
func (f *fixedTxFetcher) TakeRemaining() []txQueueItem {
	items := f.items
	items = append(items, f.exhausted...)
	f.items = nil
	f.exhausted = nil
	return items
}

// txOrderer decides the tx order of one block. The sequencer creates an orderer per regular-tx
// block and drives it under the createBlockMutex, so implementations don't need to be
// thread-safe.
type txOrderer interface {
	// nextTxFetcher yields the candidates of the block armed by StartBlock one at a time.
	nextTxFetcher

	// StartBlock collects and orders the block's candidate txs, reporting whether there is
	// any work.
	StartBlock(statedb *state.StateDB) (hasWork bool)

	// TakeRemaining removes and returns the never-yielded candidates for the caller to
	// dispose of.
	TakeRemaining() []txQueueItem

	// OnNonceGapResolved hands the orderer a parked tx whose nonce gap the last
	// inclusion just closed, so it can re-enter the block's candidates.
	OnNonceGapResolved(queueItem txQueueItem)
}

// txOrdererSequencer is the sequencer functionality the tx orderers depend on.
type txOrdererSequencer interface {
	// drainValidatedTxs drains, validates, and nonce-prechecks the pending txs for the next
	// block, in priority order.
	drainValidatedTxs(statedb *state.StateDB, baseFee *big.Int) []txQueueItem
}

// fifoTxOrderer yields the block's candidates in the order the sequencer drained them.
type fifoTxOrderer struct {
	seq     txOrdererSequencer
	baseFee *big.Int
	// The embedded fetcher holds the not-yet-yielded candidates; set by StartBlock, consumed
	// through NextQueueItem, and emptied by TakeRemaining.
	fixedTxFetcher
}

var _ txOrderer = (*fifoTxOrderer)(nil)

func newFIFOTxOrderer(seq txOrdererSequencer, baseFee *big.Int) *fifoTxOrderer {
	return &fifoTxOrderer{seq: seq, baseFee: baseFee}
}

// StartBlock drains the sequencer's pending txs as the block's candidates.
func (o *fifoTxOrderer) StartBlock(statedb *state.StateDB) bool {
	items := o.seq.drainValidatedTxs(statedb, o.baseFee)
	o.fixedTxFetcher = fixedTxFetcher{items: items}
	return len(items) > 0
}

// OnTxInclusion is a no-op: FIFO ordering doesn't react to inclusions.
func (o *fifoTxOrderer) OnTxInclusion(queueItem txQueueItem) {}

// OnNonceGapResolved appends the revived tx to the block's candidates: its nonce is valid
// against the in-progress state, so it can follow its predecessor into the same block.
func (o *fifoTxOrderer) OnNonceGapResolved(queueItem txQueueItem) {
	o.items = append(o.items, queueItem)
}
