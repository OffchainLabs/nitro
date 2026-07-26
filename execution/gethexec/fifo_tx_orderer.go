// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

// txOrdererSequencer is the sequencer functionality the tx orderers depend on.
type txOrdererSequencer interface {
	// drainValidatedTxs drains, validates, and nonce-prechecks the pending txs for the next
	// block, in priority order.
	drainValidatedTxs() []txQueueItem
}

// fifoTxOrderer yields the block's candidates in the order the sequencer drained them and
// never re-yields a requeued tx in the same block.
type fifoTxOrderer struct {
	seq txOrdererSequencer

	// The embedded fetcher holds the not-yet-yielded candidates; set by StartBlock, consumed
	// through NextQueueItem.
	fixedTxFetcher

	// requeued holds the txs given back via Requeue until TakeRemaining returns them.
	requeued []txQueueItem
}

var _ txOrderer = (*fifoTxOrderer)(nil)

func newFIFOTxOrderer(seq txOrdererSequencer) *fifoTxOrderer {
	return &fifoTxOrderer{seq: seq}
}

// StartBlock drains the sequencer's pending txs as the block's candidates.
func (o *fifoTxOrderer) StartBlock() bool {
	items := o.seq.drainValidatedTxs()
	o.fixedTxFetcher = fixedTxFetcher{items: items}
	return len(items) > 0
}

// Requeue holds the tx for TakeRemaining; it is never yielded again in the current block.
func (o *fifoTxOrderer) Requeue(item txQueueItem) {
	o.requeued = append(o.requeued, item)
}

// TakeRemaining returns the requeued txs first: they were yielded from the head of the drain,
// so the result preserves the drain order.
func (o *fifoTxOrderer) TakeRemaining() []txQueueItem {
	items := o.takeAll()
	remaining := make([]txQueueItem, 0, len(o.requeued)+len(items))
	remaining = append(remaining, o.requeued...)
	remaining = append(remaining, items...)
	o.requeued = nil
	return remaining
}
