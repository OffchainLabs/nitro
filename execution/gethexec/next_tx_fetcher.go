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
