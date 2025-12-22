// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"testing"

	"github.com/offchainlabs/nitro/arbutil"
)

func TestShouldBroadcastDuringSync(t *testing.T) {
	tests := []struct {
		name                   string
		synced                 bool
		firstMsgIdx            arbutil.MessageIndex
		msgCount               int
		batchCount             uint64
		batchThresholdMsgCount arbutil.MessageIndex
		haveBatchMetadata      bool
		expectedBroadcast      bool
		description            string
	}{
		{
			name:              "empty batch - never broadcast",
			msgCount:          0,
			expectedBroadcast: false,
			description:       "Empty message batches should never be broadcast",
		},
		{
			name:              "synced state - always broadcast",
			synced:            true,
			msgCount:          5,
			expectedBroadcast: true,
			description:       "When node is synced, should always broadcast",
		},
		{
			name:              "no batch metadata - fail open and broadcast",
			synced:            false,
			msgCount:          5,
			haveBatchMetadata: false,
			expectedBroadcast: true,
			description:       "When batch metadata is unavailable, fail open and broadcast",
		},
		{
			name:                   "batch info error with valid batch count - fail open and broadcast",
			synced:                 false,
			firstMsgIdx:            195,
			msgCount:               5,
			batchCount:             10,
			batchThresholdMsgCount: 199,
			haveBatchMetadata:      false, // This simulates the error getting metadata
			expectedBroadcast:      true,
			description:            "When batch metadata fetch fails (despite valid batch count), fail open and broadcast",
		},
		{
			name:              "insufficient batches - fail open and broadcast",
			synced:            false,
			msgCount:          5,
			batchCount:        1,
			haveBatchMetadata: true,
			expectedBroadcast: true,
			description:       "When less than 2 batches exist, fail open and broadcast",
		},
		{
			name:                   "not synced - messages before threshold - no broadcast",
			synced:                 false,
			firstMsgIdx:            100,
			msgCount:               10,
			batchCount:             10,
			batchThresholdMsgCount: 200, // Messages 100-109 are all before threshold of 200
			haveBatchMetadata:      true,
			expectedBroadcast:      false,
			description:            "When not synced and all messages are before backlog threshold, should not broadcast",
		},
		{
			name:                   "not synced - last message at threshold - broadcast",
			synced:                 false,
			firstMsgIdx:            195,
			msgCount:               5,
			batchCount:             10,
			batchThresholdMsgCount: 199, // Messages 195-199, last message (199) is at threshold
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "When not synced but last message reaches threshold, should broadcast",
		},
		{
			name:                   "not synced - last message after threshold - broadcast",
			synced:                 false,
			firstMsgIdx:            200,
			msgCount:               5,
			batchCount:             10,
			batchThresholdMsgCount: 199, // Messages 200-204, last message (204) is after threshold
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "When not synced but last message is after threshold, should broadcast",
		},
		{
			name:                   "not synced - boundary case - first message before, last after threshold",
			synced:                 false,
			firstMsgIdx:            195,
			msgCount:               10,
			batchCount:             10,
			batchThresholdMsgCount: 199, // Messages 195-204, crosses threshold at message 199
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "When batch spans threshold boundary, should broadcast entire batch",
		},
		{
			name:                   "edge case - single message exactly at threshold",
			synced:                 false,
			firstMsgIdx:            100,
			msgCount:               1,
			batchCount:             5,
			batchThresholdMsgCount: 100,
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "Single message exactly at threshold should broadcast",
		},
		{
			name:                   "edge case - single message just before threshold",
			synced:                 false,
			firstMsgIdx:            99,
			msgCount:               1,
			batchCount:             5,
			batchThresholdMsgCount: 100,
			haveBatchMetadata:      true,
			expectedBroadcast:      false,
			description:            "Single message just before threshold should not broadcast",
		},
		{
			name:                   "edge case - single message just after threshold",
			synced:                 false,
			firstMsgIdx:            101,
			msgCount:               1,
			batchCount:             5,
			batchThresholdMsgCount: 100,
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "Single message just after threshold should broadcast",
		},
		{
			name:                   "batch count exactly 2 - should work normally",
			synced:                 false,
			firstMsgIdx:            50,
			msgCount:               5,
			batchCount:             2,
			batchThresholdMsgCount: 45, // batch 0 ends at message 45
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "With exactly 2 batches, threshold calculation should work",
		},
		{
			name:                   "large batch spanning multiple old batches",
			synced:                 false,
			firstMsgIdx:            1000,
			msgCount:               500,
			batchCount:             100,
			batchThresholdMsgCount: 1200, // Messages 1000-1499, threshold at 1200
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "Large batch that includes messages after threshold should broadcast",
		},
		{
			name:                   "metadata available but batch count is 0",
			synced:                 false,
			firstMsgIdx:            10,
			msgCount:               5,
			batchCount:             0,
			batchThresholdMsgCount: 0,
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "Zero batch count should fail open and broadcast",
		},
		// Threshold calculation test cases
		{
			name:                   "threshold calc - all messages before threshold",
			synced:                 false,
			firstMsgIdx:            90,
			msgCount:               5, // messages 90-94
			batchCount:             10,
			batchThresholdMsgCount: 100,
			haveBatchMetadata:      true,
			expectedBroadcast:      false,
			description:            "All messages before threshold should not broadcast",
		},
		{
			name:                   "threshold calc - first messages before, last at threshold",
			synced:                 false,
			firstMsgIdx:            96,
			msgCount:               5, // messages 96-100
			batchCount:             10,
			batchThresholdMsgCount: 100,
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "Messages reaching threshold should broadcast",
		},
		{
			name:                   "threshold calc - all messages after threshold",
			synced:                 false,
			firstMsgIdx:            101,
			msgCount:               5, // messages 101-105
			batchCount:             10,
			batchThresholdMsgCount: 100,
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "All messages after threshold should broadcast",
		},
		{
			name:                   "threshold calc - single message at threshold boundary",
			synced:                 false,
			firstMsgIdx:            100,
			msgCount:               1, // message 100
			batchCount:             10,
			batchThresholdMsgCount: 100,
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "Single message at threshold boundary should broadcast",
		},
		{
			name:                   "threshold calc - large batch spanning threshold",
			synced:                 false,
			firstMsgIdx:            50,
			msgCount:               100, // messages 50-149
			batchCount:             10,
			batchThresholdMsgCount: 100,
			haveBatchMetadata:      true,
			expectedBroadcast:      true,
			description:            "Large batch spanning threshold should broadcast",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldBroadcastDuringSync(
				tt.synced,
				tt.firstMsgIdx,
				tt.msgCount,
				tt.batchCount,
				tt.batchThresholdMsgCount,
				tt.haveBatchMetadata,
			)

			if result != tt.expectedBroadcast {
				t.Errorf("ShouldBroadcastDuringSync() = %v, expected %v. %s", result, tt.expectedBroadcast, tt.description)
				t.Logf("Test parameters: synced=%v, firstMsgIdx=%d, msgCount=%d, batchCount=%d, threshold=%d, haveBatchMetadata=%v",
					tt.synced, tt.firstMsgIdx, tt.msgCount, tt.batchCount, tt.batchThresholdMsgCount, tt.haveBatchMetadata)
			}
		})
	}
}
