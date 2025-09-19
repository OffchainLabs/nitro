// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"errors"
	"testing"

	"github.com/offchainlabs/nitro/arbutil"
)

// Mock SyncStatusProvider for testing
type mockSyncStatusProvider struct {
	synced bool
}

func (m *mockSyncStatusProvider) Synced() bool {
	return m.synced
}

// Mock BatchInfoProvider for testing
type mockBatchInfoProvider struct {
	batchCount uint64
	batchMetas map[uint64]BatchMetadata
	err        error
}

func (m *mockBatchInfoProvider) GetBatchCount() (uint64, error) {
	return m.batchCount, m.err
}

func (m *mockBatchInfoProvider) GetBatchMetadata(seqNum uint64) (BatchMetadata, error) {
	if m.err != nil {
		return BatchMetadata{}, m.err
	}
	meta, exists := m.batchMetas[seqNum]
	if !exists {
		return BatchMetadata{MessageCount: 0}, nil
	}
	return meta, nil
}

func TestBroadcastSyncChecker(t *testing.T) {
	tests := []struct {
		name              string
		synced            bool
		firstMsgIdx       arbutil.MessageIndex
		msgCount          int
		batchCount        uint64
		thresholdMsgCount arbutil.MessageIndex
		batchInfoError    bool
		expected          bool
		description       string
	}{
		{
			name:        "empty batch - never broadcast",
			msgCount:    0,
			expected:    false,
			description: "Empty message batches should never be broadcast",
		},
		{
			name:        "synced state - always broadcast",
			synced:      true,
			msgCount:    5,
			expected:    true,
			description: "When node is synced, should always broadcast",
		},
		{
			name:           "batch info error - always broadcast",
			synced:         false,
			msgCount:       5,
			batchInfoError: true,
			expected:       true,
			description:    "When batch info provider has errors, fail open and broadcast",
		},
		{
			name:        "insufficient batches - always broadcast",
			synced:      false,
			msgCount:    5,
			batchCount:  1,
			expected:    true,
			description: "When less than 2 batches exist, fail open and broadcast",
		},
		{
			name:              "not synced - messages before threshold - no broadcast",
			synced:            false,
			firstMsgIdx:       100,
			msgCount:          10,
			batchCount:        10,
			thresholdMsgCount: 200, // Messages 100-109 are all before threshold of 200
			expected:          false,
			description:       "When not synced and all messages are before backlog threshold, should not broadcast",
		},
		{
			name:              "not synced - last message at threshold - broadcast",
			synced:            false,
			firstMsgIdx:       195,
			msgCount:          5,
			batchCount:        10,
			thresholdMsgCount: 199, // Messages 195-199, last message (199) is at threshold
			expected:          true,
			description:       "When not synced but last message reaches threshold, should broadcast",
		},
		{
			name:              "not synced - last message after threshold - broadcast",
			synced:            false,
			firstMsgIdx:       200,
			msgCount:          5,
			batchCount:        10,
			thresholdMsgCount: 199, // Messages 200-204, last message (204) is after threshold
			expected:          true,
			description:       "When not synced but last message is after threshold, should broadcast",
		},
		{
			name:              "not synced - boundary case - first message before, last after threshold",
			synced:            false,
			firstMsgIdx:       195,
			msgCount:          10,
			batchCount:        10,
			thresholdMsgCount: 199, // Messages 195-204, crosses threshold at message 199
			expected:          true,
			description:       "When batch spans threshold boundary, should broadcast entire batch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup sync status provider
			syncStatus := &mockSyncStatusProvider{
				synced: tt.synced,
			}

			// Setup batch info provider
			batchInfo := &mockBatchInfoProvider{
				batchCount: tt.batchCount,
				batchMetas: make(map[uint64]BatchMetadata),
			}

			if tt.batchInfoError {
				batchInfo.err = errors.New("mock error")
			} else if tt.batchCount >= 2 {
				// Set up the batch metadata for batch (batchCount - 2)
				batchInfo.batchMetas[tt.batchCount-2] = BatchMetadata{
					MessageCount: tt.thresholdMsgCount,
				}
			}

			// Create the checker
			checker := NewBroadcastSyncChecker(syncStatus, batchInfo)

			// Test the ShouldBroadcast method
			result := checker.ShouldBroadcast(tt.firstMsgIdx, tt.msgCount)

			if result != tt.expected {
				t.Errorf("ShouldBroadcast() = %v, expected %v. %s", result, tt.expected, tt.description)
				t.Logf("Test parameters: firstMsgIdx=%d, msgCount=%d, batchCount=%d, threshold=%d",
					tt.firstMsgIdx, tt.msgCount, tt.batchCount, tt.thresholdMsgCount)
			}
		})
	}
}

func TestBroadcastSyncCheckerEdgeCases(t *testing.T) {
	// Test single message at exact threshold
	syncStatus := &mockSyncStatusProvider{synced: false}
	batchInfo := &mockBatchInfoProvider{
		batchCount: 5,
		batchMetas: map[uint64]BatchMetadata{
			3: {MessageCount: 100}, // Threshold is at message 100
		},
	}

	checker := NewBroadcastSyncChecker(syncStatus, batchInfo)

	testCases := []struct {
		name        string
		firstMsgIdx arbutil.MessageIndex
		msgCount    int
		expected    bool
	}{
		{
			name:        "single message exactly at threshold",
			firstMsgIdx: 100,
			msgCount:    1,
			expected:    true,
		},
		{
			name:        "single message just before threshold",
			firstMsgIdx: 99,
			msgCount:    1,
			expected:    false,
		},
		{
			name:        "single message just after threshold",
			firstMsgIdx: 101,
			msgCount:    1,
			expected:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.ShouldBroadcast(tc.firstMsgIdx, tc.msgCount)
			if result != tc.expected {
				t.Errorf("ShouldBroadcast(%d, %d) = %v, expected %v",
					tc.firstMsgIdx, tc.msgCount, result, tc.expected)
			}
		})
	}
}
