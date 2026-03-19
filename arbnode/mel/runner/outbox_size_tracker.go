// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import "github.com/ethereum/go-ethereum/log"

// OutboxSizeTracker maintains a sliding window of outbox sizes indexed by
// parent chain block number. This enables O(1) pivot lookup during
// RebuildDelayedMsgPreimages instead of the O(N²) findPivot search.
//
// The window is trimmed from the left when parent chain blocks finalize
// (those entries are no longer needed for reorg recovery) and trimmed
// from the right on reorgs (discarding entries for reorged blocks).
type OutboxSizeTracker struct {
	startBlock        uint64
	sizes             []int
	getFinalizedBlock func() (uint64, error)
}

func NewOutboxSizeTracker(startBlock uint64, initialOutboxSize int, getFinalizedBlock func() (uint64, error)) *OutboxSizeTracker {
	return &OutboxSizeTracker{
		startBlock:        startBlock,
		sizes:             []int{initialOutboxSize},
		getFinalizedBlock: getFinalizedBlock,
	}
}

// Record appends the outbox size for the given block number.
// Block numbers must be recorded sequentially.
func (t *OutboxSizeTracker) Record(blockNum uint64, outboxSize int) {
	expected := t.startBlock + uint64(len(t.sizes))
	if blockNum != expected {
		log.Warn("OutboxSizeTracker: non-sequential block recorded, resetting",
			"expected", expected, "got", blockNum)
		t.startBlock = blockNum
		t.sizes = []int{outboxSize}
		return
	}
	t.sizes = append(t.sizes, outboxSize)
}

// Lookup returns the outbox size at the given block number if available.
func (t *OutboxSizeTracker) Lookup(blockNum uint64) (int, bool) {
	if blockNum < t.startBlock {
		return 0, false
	}
	idx := blockNum - t.startBlock
	if idx >= uint64(len(t.sizes)) {
		return 0, false
	}
	return t.sizes[idx], true
}

// TrimLeft discards entries for blocks <= upToBlock, advancing startBlock.
func (t *OutboxSizeTracker) TrimLeft(upToBlock uint64) {
	if upToBlock < t.startBlock {
		return
	}
	trimCount := min(uint64(len(t.sizes)), upToBlock-t.startBlock+1)
	t.sizes = t.sizes[trimCount:]
	t.startBlock = upToBlock + 1
}

// TrimRight discards entries for blocks >= fromBlock.
func (t *OutboxSizeTracker) TrimRight(fromBlock uint64) {
	if fromBlock <= t.startBlock {
		t.startBlock = fromBlock
		t.sizes = nil
		return
	}
	keepCount := fromBlock - t.startBlock
	if keepCount < uint64(len(t.sizes)) {
		t.sizes = t.sizes[:keepCount]
	}
}

// Reset discards all tracked state and reinitializes from the given block.
// Called when corruption is detected (e.g., negative outbox size).
func (t *OutboxSizeTracker) Reset(startBlock uint64, outboxSize int) {
	t.startBlock = startBlock
	t.sizes = []int{outboxSize}
}

// TrimToFinalized calls the stored finalization callback to get the current
// finalized block number, then trims entries for finalized blocks.
func (t *OutboxSizeTracker) TrimToFinalized() {
	if t.getFinalizedBlock == nil {
		return
	}
	finalizedBlock, err := t.getFinalizedBlock()
	if err != nil {
		log.Debug("OutboxSizeTracker: failed to get finalized block", "err", err)
		return
	}
	if finalizedBlock > 0 {
		t.TrimLeft(finalizedBlock)
	}
}
