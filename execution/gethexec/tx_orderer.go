// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

// nextTxFetcher supplies a block's tx candidates to the sequencing hooks one at a time.
type nextTxFetcher interface {
	// NextQueueItem yields the next block candidate, reporting false on exhaustion.
	NextQueueItem() (txQueueItem, bool)
}

// fixedTxFetcher yields a pre-set candidate list.
type fixedTxFetcher struct {
	items []txQueueItem
}

var _ nextTxFetcher = (*fixedTxFetcher)(nil)

func (f *fixedTxFetcher) NextQueueItem() (txQueueItem, bool) {
	if len(f.items) == 0 {
		return txQueueItem{}, false
	}
	item := f.items[0]
	f.items = f.items[1:]
	return item, true
}

// TakeRemaining removes and returns the not-yet-yielded candidates.
func (f *fixedTxFetcher) TakeRemaining() []txQueueItem {
	items := f.items
	f.items = nil
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
	StartBlock() (hasWork bool)

	// TakeRemaining removes and returns the never-yielded candidates for the caller to
	// dispose of.
	TakeRemaining() []txQueueItem

	// OnTxInclusion notifies the orderer that the last yielded tx made it into the block.
	OnTxInclusion()
}

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

// OnTxInclusion is a no-op: FIFO ordering doesn't react to inclusions.
func (o *fifoTxOrderer) OnTxInclusion() {}
