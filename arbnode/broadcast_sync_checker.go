// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"github.com/offchainlabs/nitro/arbutil"
)

// SyncStatusProvider provides sync status information
type SyncStatusProvider interface {
	Synced() bool
}

// BatchInfoProvider provides batch-related information for backlog calculation
type BatchInfoProvider interface {
	GetBatchCount() (uint64, error)
	GetBatchMetadata(seqNum uint64) (BatchMetadata, error)
}

// BroadcastSyncChecker determines whether messages should be broadcast based on sync status
type BroadcastSyncChecker struct {
	syncStatus SyncStatusProvider
	batchInfo  BatchInfoProvider
}

func NewBroadcastSyncChecker(syncStatus SyncStatusProvider, batchInfo BatchInfoProvider) *BroadcastSyncChecker {
	return &BroadcastSyncChecker{
		syncStatus: syncStatus,
		batchInfo:  batchInfo,
	}
}

// ShouldBroadcast determines if messages should be broadcast based on sync state and message position
func (b *BroadcastSyncChecker) ShouldBroadcast(firstMsgIdx arbutil.MessageIndex, msgCount int) bool {
	if msgCount == 0 {
		return false
	}

	if b.syncStatus.Synced() {
		return true
	}

	// We're not synced, so check if these messages are within the 2 batch backlog threshold.
	// Check the LAST message in the batch - if it should be broadcast, send the whole batch.
	// If any errors stop us from determining the last 2 batch threshold, then fail open.
	batchCount, err := b.batchInfo.GetBatchCount()
	if err != nil || batchCount < 2 {
		return true
	}

	batchMeta, err := b.batchInfo.GetBatchMetadata(batchCount - 2)
	if err != nil {
		return true
	}

	// Check if the LAST message in this batch is within the backlog threshold
	// #nosec G115
	lastMsgIdx := firstMsgIdx + arbutil.MessageIndex(msgCount) - 1
	return lastMsgIdx >= batchMeta.MessageCount
}
