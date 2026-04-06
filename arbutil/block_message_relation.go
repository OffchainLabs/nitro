// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbutil

import "fmt"

// messages are 0-indexed
type MessageIndex uint64

func BlockNumberToMessageIndex(blockNum, genesis uint64) (MessageIndex, error) {
	if blockNum < genesis {
		return 0, fmt.Errorf("blockNum %d < genesis %d", blockNum, genesis)
	}
	return MessageIndex(blockNum - genesis), nil
}

func MessageIndexToBlockNumber(msgIdx MessageIndex, genesis uint64) uint64 {
	return uint64(msgIdx) + genesis
}

// BatchCountGetter provides the two methods needed by FindInboxBatchContainingMessage.
// Both InboxTracker and MessageExtractor satisfy this interface.
// Implementations must return monotonically non-decreasing MessageIndex values
// for increasing seqNum arguments to ensure correct binary search behavior.
type BatchCountGetter interface {
	GetBatchCount() (uint64, error)
	GetBatchMessageCount(seqNum uint64) (MessageIndex, error)
}

// FindInboxBatchContainingMessage performs a binary search over batch metadata
// to find the batch that contains the given message position. Returns
// (batchNum, true, nil) on success, (0, false, nil) if not yet posted in any
// batch, or (0, false, err) on unexpected errors.
func FindInboxBatchContainingMessage(reader BatchCountGetter, pos MessageIndex) (uint64, bool, error) {
	batchCount, err := reader.GetBatchCount()
	if err != nil {
		return 0, false, err
	}
	if batchCount == 0 {
		return 0, false, nil
	}
	low := uint64(0)
	high := batchCount - 1
	lastBatchMessageCount, err := reader.GetBatchMessageCount(high)
	if err != nil {
		return 0, false, err
	}
	if lastBatchMessageCount <= pos {
		return 0, false, nil
	}
	// Iteration preconditions:
	// - high >= low
	// - msgCount(low - 1) <= pos implies low <= target
	// - msgCount(high) > pos implies high >= target
	// Therefore, if low == high, then low == high == target
	const maxIter = 64
	for range maxIter {
		// Due to integer rounding, mid >= low && mid < high
		mid := (low + high) / 2
		count, err := reader.GetBatchMessageCount(mid)
		if err != nil {
			return 0, false, err
		}
		if count < pos {
			// Must narrow as mid >= low, therefore mid + 1 > low, therefore newLow > oldLow
			// Keeps low precondition as msgCount(mid) < pos
			low = mid + 1
		} else if count == pos {
			return mid + 1, true, nil
		} else if count == pos+1 || mid == low { // implied: count > pos
			return mid, true, nil
		} else {
			// implied: count > pos + 1
			// Must narrow as mid < high, therefore newHigh < oldHigh
			// Keeps high precondition as msgCount(mid) > pos
			high = mid
		}
		if high == low {
			return high, true, nil
		}
	}
	return 0, false, fmt.Errorf("FindInboxBatchContainingMessage: exceeded %d iterations searching for message %d in %d batches; possible inconsistent batch metadata", maxIter, pos, batchCount)
}
