// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

// txOrdererSequencer is the sequencer functionality the tx orderers depend on.
type txOrdererSequencer interface {
	// drainValidatedTxs drains, validates, and nonce-prechecks the pending txs for the next
	// block, in priority order.
	drainValidatedTxs() []txQueueItem
}

// fifoTxOrderer yields the block's candidates in the order the sequencer drained them.
type fifoTxOrderer struct {
	seq txOrdererSequencer

	// The embedded fetcher holds the not-yet-yielded candidates; set by StartBlock, consumed
	// through NextQueueItem, and emptied by TakeRemaining.
	fixedTxFetcher
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
