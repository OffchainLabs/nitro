package bold

import (
	"errors"
	"testing"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/bold/protocol"
)

type MockInboxTracker struct {
	batchMessageCounts               map[uint64]arbutil.MessageIndex
	messageToBatch                   map[arbutil.MessageIndex]uint64
	GetBatchMessageCountCalls        []uint64
	FindInboxBatchContainingMsgCalls []arbutil.MessageIndex
}

func NewMockInboxTracker() *MockInboxTracker {
	return &MockInboxTracker{
		batchMessageCounts: make(map[uint64]arbutil.MessageIndex),
		messageToBatch:     make(map[arbutil.MessageIndex]uint64),
	}
}

func (m *MockInboxTracker) SetBatchMessageCount(batchIndex uint64, count arbutil.MessageIndex) {
	m.batchMessageCounts[batchIndex] = count
}

func (m *MockInboxTracker) SetMessageToBatch(messageIndex arbutil.MessageIndex, batchIndex uint64) {
	m.messageToBatch[messageIndex] = batchIndex
}

func (m *MockInboxTracker) GetBatchMessageCount(batchIndex uint64) (arbutil.MessageIndex, error) {
	m.GetBatchMessageCountCalls = append(m.GetBatchMessageCountCalls, batchIndex)

	count, ok := m.batchMessageCounts[batchIndex]
	if !ok {
		return 0, errors.New("batch not found")
	}
	return count, nil
}

func (m *MockInboxTracker) FindInboxBatchContainingMessage(messageIndex arbutil.MessageIndex) (uint64, bool, error) {
	m.FindInboxBatchContainingMsgCalls = append(m.FindInboxBatchContainingMsgCalls, messageIndex)

	batch, ok := m.messageToBatch[messageIndex]
	if !ok {
		return uint64(messageIndex / 1000), true, nil
	}
	return batch, true, nil
}

func TestExecutionStateAfterPreviousState_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name             string
		maxSeqInboxCount uint64
		previousState    protocol.GoGlobalState
		leafHeight       uint64
		setupMock        func(*MockInboxTracker)
		expectCapped     bool
		expectedMsgCount arbutil.MessageIndex // The message count that should be used
	}{
		{
			name:             "should cap with large number of previous messages",
			maxSeqInboxCount: 1001,
			previousState: protocol.GoGlobalState{
				Batch:      1000,
				PosInBatch: 0,
			},
			leafHeight: 100,
			setupMock: func(m *MockInboxTracker) {
				// Batch 999 ends at message 1,000,000
				m.SetBatchMessageCount(999, 1_000_000)
				// Batch 1000 ends at message 1,000,400 (400 more messages)
				m.SetBatchMessageCount(1000, 1_000_400)
				// The capped message count should map to batch 1000
				m.SetMessageToBatch(1_000_100, 1000)
			},
			expectCapped:     true,
			expectedMsgCount: 1_000_100, // Should be capped to prev + 100.
		},
		{
			name:             "no cap as the number of messages is within the limit",
			maxSeqInboxCount: 1001,
			previousState: protocol.GoGlobalState{
				Batch:      1000,
				PosInBatch: 0,
			},
			leafHeight: 100,
			setupMock: func(m *MockInboxTracker) {
				m.SetBatchMessageCount(999, 1_000_000)
				// Only 50 messages in the batch
				m.SetBatchMessageCount(1000, 1_000_050)
			},
			expectCapped:     false,
			expectedMsgCount: 1_000_050,
		},
		{
			name:             "capped after looking at position in batch",
			maxSeqInboxCount: 1001,
			previousState: protocol.GoGlobalState{
				Batch:      1000,
				PosInBatch: 50, // Already 50 messages into the batch
			},
			leafHeight: 100,
			setupMock: func(m *MockInboxTracker) {
				// Batch 999 ends at 1,000,000
				m.SetBatchMessageCount(999, 1_000_000)
				// Previous is at 1,000,050 (batch 999 end + posInBatch)
				// Batch 1000 ends at 1,000,300 (250 more from previous)
				m.SetBatchMessageCount(1000, 1_000_300)
				m.SetMessageToBatch(1_000_150, 1000) // 1,000,050 + 100
			},
			expectCapped:     true,
			expectedMsgCount: 1_000_150, // prev (1,000,050) + max (100)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockInboxTracker()
			tt.setupMock(mock)

			_, _, err := computeNextMessageCountAndBatchIndex(
				tt.maxSeqInboxCount,
				tt.previousState,
				mock,
				tt.leafHeight,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Verify capping behavior by checking if FindInboxBatchContainingMessage was called.
			wasCapped := len(mock.FindInboxBatchContainingMsgCalls) > 0
			if wasCapped != tt.expectCapped {
				t.Errorf("expected capped=%v, got capped=%v", tt.expectCapped, wasCapped)
			}
			if tt.expectCapped {
				if len(mock.FindInboxBatchContainingMsgCalls) == 0 {
					t.Error("expected FindInboxBatchContainingMessage to be called")
				} else {
					calledWith := mock.FindInboxBatchContainingMsgCalls[0]
					if calledWith != tt.expectedMsgCount {
						t.Errorf("FindInboxBatchContainingMessage called with %d, expected %d",
							calledWith, tt.expectedMsgCount)
					}
				}
			}
		})
	}
}
