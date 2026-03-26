// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package mel

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/containers"
)

func makeTestMessage(blockNumber uint64) *arbostypes.MessageWithMetadata {
	return &arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				BlockNumber: blockNumber,
				RequestId:   &common.Hash{},
				L1BaseFee:   common.Big0,
			},
		},
		DelayedMessagesRead: blockNumber,
	}
}

func TestAccumulateMessageHashComputation(t *testing.T) {
	state := &State{}

	msg1 := makeTestMessage(1)
	require.NoError(t, state.AccumulateMessage(msg1))

	// Independently compute expected accumulator after first message
	msgBytes1, err := rlp.EncodeToBytes(msg1)
	require.NoError(t, err)
	msgHash1 := crypto.Keccak256Hash(msgBytes1)
	expected1 := crypto.Keccak256Hash(append(common.Hash{}.Bytes(), msgHash1.Bytes()...))
	require.Equal(t, expected1, state.LocalMsgAccumulator, "accumulator mismatch after first message")

	// Accumulate a second message and verify chaining
	msg2 := makeTestMessage(2)
	require.NoError(t, state.AccumulateMessage(msg2))

	msgBytes2, err := rlp.EncodeToBytes(msg2)
	require.NoError(t, err)
	msgHash2 := crypto.Keccak256Hash(msgBytes2)
	expected2 := crypto.Keccak256Hash(append(expected1.Bytes(), msgHash2.Bytes()...))
	require.Equal(t, expected2, state.LocalMsgAccumulator, "accumulator mismatch after second message")
}

func TestCloneZerosLocalMsgAccumulator(t *testing.T) {
	state := &State{}
	preimages := make(daprovider.PreimagesMap)
	require.NoError(t, state.RecordMsgPreimagesTo(preimages))

	for i := range 3 {
		require.NoError(t, state.AccumulateMessage(makeTestMessage(uint64(i))))
	}
	require.NotEqual(t, common.Hash{}, state.LocalMsgAccumulator, "accumulator should be non-zero after messages")

	originalAcc := state.LocalMsgAccumulator
	cloned := state.Clone()

	require.Equal(t, common.Hash{}, cloned.LocalMsgAccumulator, "cloned state should have zero LocalMsgAccumulator")
	require.NotNil(t, cloned.msgPreimagesDest, "cloned state should preserve msgPreimagesDest")
	require.Equal(t, originalAcc, state.LocalMsgAccumulator, "original accumulator should be unchanged after clone")
}

func TestPreimageRecordingCounts(t *testing.T) {
	t.Run("recording enabled", func(t *testing.T) {
		state := &State{}
		preimages := make(daprovider.PreimagesMap)
		require.NoError(t, state.RecordMsgPreimagesTo(preimages))

		n := 5
		for i := range n {
			require.NoError(t, state.AccumulateMessage(makeTestMessage(uint64(i))))
		}
		require.Equal(t, 2*n, len(preimages[arbutil.Keccak256PreimageType]), "expected exactly 2 preimages per message")
	})

	t.Run("recording disabled", func(t *testing.T) {
		state := &State{}
		n := 5
		for i := range n {
			require.NoError(t, state.AccumulateMessage(makeTestMessage(uint64(i))))
		}
		require.NotEqual(t, common.Hash{}, state.LocalMsgAccumulator, "accumulator should still work without recording")
	})
}

func createTestDelayedMessages(n int) []*DelayedInboxMessage {
	msgs := make([]*DelayedInboxMessage, n)
	for i := range n {
		reqID := common.BigToHash(big.NewInt(int64(i)))
		msgs[i] = &DelayedInboxMessage{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:      arbostypes.L1MessageType_EndOfBlock,
					RequestId: &reqID,
					L1BaseFee: common.Big0,
				},
				L2msg: []byte(fmt.Sprintf("delayed-msg-%d", i)),
			},
		}
	}
	return msgs
}

func accumulateN(t *testing.T, s *State, msgs []*DelayedInboxMessage) {
	t.Helper()
	for _, msg := range msgs {
		require.NoError(t, s.AccumulateDelayedMessage(msg))
		s.DelayedMessagesSeen++
	}
}

func makeFetcher(msgs []*DelayedInboxMessage, startIndex uint64) func(uint64) (*DelayedInboxMessage, error) {
	return func(index uint64) (*DelayedInboxMessage, error) {
		offset := index - startIndex
		if offset >= uint64(len(msgs)) {
			return nil, fmt.Errorf("message index %d out of range", index)
		}
		return msgs[offset], nil
	}
}

func TestRebuildDelayedMsgPreimages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		totalMsgs int
		pourFirst bool
		addAfter  int // messages to accumulate after pour (creates mixed state)
	}{
		{
			name:      "all messages in inbox only",
			totalMsgs: 5,
			pourFirst: false,
		},
		{
			name:      "all messages in outbox only",
			totalMsgs: 5,
			pourFirst: true,
		},
		{
			name:      "mixed outbox and inbox",
			totalMsgs: 3,
			pourFirst: true,
			addAfter:  2,
		},
		{
			name:      "empty queue is no-op",
			totalMsgs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Build the golden state with a populated cache.
			s := &State{}
			var allMsgs []*DelayedInboxMessage

			if tc.totalMsgs > 0 {
				msgs := createTestDelayedMessages(tc.totalMsgs)
				accumulateN(t, s, msgs)
				allMsgs = append(allMsgs, msgs...)
			}

			if tc.pourFirst {
				require.NoError(t, s.PourDelayedInboxToOutbox())
			}

			if tc.addAfter > 0 {
				extra := createTestDelayedMessages(tc.addAfter)
				// Make distinct from the first batch
				for i := range extra {
					reqID := common.BigToHash(big.NewInt(int64(tc.totalMsgs + i)))
					extra[i].Message.Header.RequestId = &reqID
					extra[i].Message.L2msg = []byte(fmt.Sprintf("delayed-msg-%d", tc.totalMsgs+i))
				}
				accumulateN(t, s, extra)
				allMsgs = append(allMsgs, extra...)
			}

			// Create a fresh state with same accumulators but nil cache
			rebuilt := &State{
				DelayedMessagesRead:     s.DelayedMessagesRead,
				DelayedMessagesSeen:     s.DelayedMessagesSeen,
				DelayedMessageInboxAcc:  s.DelayedMessageInboxAcc,
				DelayedMessageOutboxAcc: s.DelayedMessageOutboxAcc,
			}

			unreadMsgs := allMsgs[rebuilt.DelayedMessagesRead:]
			fetcher := makeFetcher(unreadMsgs, rebuilt.DelayedMessagesRead)

			require.NoError(t, rebuilt.RebuildDelayedMsgPreimages(fetcher))

			if tc.totalMsgs == 0 && tc.addAfter == 0 {
				return // empty queue, nothing more to verify
			}

			// Verify the rebuilt cache allows pop to work (PopDelayedOutbox
			// auto-pours when the outbox is empty, so no manual pour needed).
			for rebuilt.DelayedMessagesRead < rebuilt.DelayedMessagesSeen {
				msgHash, err := rebuilt.PopDelayedOutbox()
				require.NoError(t, err)
				require.NotEqual(t, common.Hash{}, msgHash)
				rebuilt.DelayedMessagesRead++
			}
			require.Equal(t, common.Hash{}, rebuilt.DelayedMessageOutboxAcc)
		})
	}
}

func TestAccumulateDelayedMessage_CacheResize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		initialCacheSize  int
		msgsToAdd         int
		expectedSizeAfter int
	}{
		{
			name:             "zero capacity grows to totalUnread+1",
			initialCacheSize: 0,
			msgsToAdd:        3,
			// Each accumulate: totalUnread=0→resize(1), totalUnread=1→resize(2), totalUnread=2→resize(3)
			expectedSizeAfter: 3,
		},
		{
			name:              "no resize when cache already large enough",
			initialCacheSize:  100,
			msgsToAdd:         5,
			expectedSizeAfter: 100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &State{}
			if tc.initialCacheSize > 0 {
				s.delayedMsgPreimages = containers.NewLruCache[common.Hash, []byte](tc.initialCacheSize)
			}

			msgs := createTestDelayedMessages(tc.msgsToAdd)
			accumulateN(t, s, msgs)

			require.NotNil(t, s.delayedMsgPreimages)
			require.Equal(t, tc.expectedSizeAfter, s.delayedMsgPreimages.Size())
		})
	}
}

func TestPourDelayedInboxToOutbox_CacheResize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		msgCount   int
		expectSize int
		expectLen  int
	}{
		{
			name:       "pour 1 message",
			msgCount:   1,
			expectSize: 3, // 3 * 1
			// With 1 message, inbox key = Keccak256(zero || msgHash) = outbox key,
			// so the outbox Add overwrites the inbox entry. Len = 1.
			expectLen: 1,
		},
		{
			name:       "pour 5 messages",
			msgCount:   5,
			expectSize: 15, // 3 * 5
			expectLen:  10, // 5 inbox + 5 outbox
		},
		{
			name:     "pour 0 messages is no-op",
			msgCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &State{}

			if tc.msgCount == 0 {
				require.NoError(t, s.PourDelayedInboxToOutbox())
				return
			}

			msgs := createTestDelayedMessages(tc.msgCount)
			accumulateN(t, s, msgs)

			inboxAcc := s.DelayedMessageInboxAcc
			require.NotEqual(t, common.Hash{}, inboxAcc)

			require.NoError(t, s.PourDelayedInboxToOutbox())

			require.Equal(t, tc.expectSize, s.delayedMsgPreimages.Size())
			require.Equal(t, tc.expectLen, s.delayedMsgPreimages.Len())

			// Inbox should be zeroed, outbox non-zero
			require.Equal(t, common.Hash{}, s.DelayedMessageInboxAcc)
			require.NotEqual(t, common.Hash{}, s.DelayedMessageOutboxAcc)

			// Old inbox entry still accessible (stale but not evicted due to 3x capacity)
			_, found := s.delayedMsgPreimages.Peek(inboxAcc)
			require.True(t, found, "stale inbox preimage should still be in cache after pour")

			// Outbox entry accessible
			_, found = s.delayedMsgPreimages.Peek(s.DelayedMessageOutboxAcc)
			require.True(t, found, "outbox preimage should be in cache after pour")

			// All messages can be popped
			for range tc.msgCount {
				_, err := s.PopDelayedOutbox()
				require.NoError(t, err)
				s.DelayedMessagesRead++
			}
			require.Equal(t, common.Hash{}, s.DelayedMessageOutboxAcc)
		})
	}
}

func TestRebuildThenPourAndPop_MatchesOriginal(t *testing.T) {
	t.Parallel()

	// Build a mixed state: accumulate 5, pour, pop 1, accumulate 2 more
	s := &State{}
	batch1 := createTestDelayedMessages(5)
	accumulateN(t, s, batch1)
	require.NoError(t, s.PourDelayedInboxToOutbox())

	hash0, err := s.PopDelayedOutbox()
	require.NoError(t, err)
	s.DelayedMessagesRead++

	extra := createTestDelayedMessages(2)
	for i := range extra {
		reqID := common.BigToHash(big.NewInt(int64(100 + i)))
		extra[i].Message.Header.RequestId = &reqID
		extra[i].Message.L2msg = []byte(fmt.Sprintf("extra-delayed-%d", i))
	}
	accumulateN(t, s, extra)

	// Now: Read=1, Seen=7, outbox has msgs 1-4, inbox has 2 extra
	// Collect all remaining hashes from the original state
	originalHashes := []common.Hash{hash0}
	for s.DelayedMessagesRead < s.DelayedMessagesSeen {
		h, err := s.PopDelayedOutbox()
		require.NoError(t, err)
		originalHashes = append(originalHashes, h)
		s.DelayedMessagesRead++
	}
	require.Len(t, originalHashes, 7)

	// Re-create the mixed state (after pop 1 + accumulate 2)
	s2 := &State{}
	accumulateN(t, s2, batch1)
	require.NoError(t, s2.PourDelayedInboxToOutbox())
	_, err = s2.PopDelayedOutbox()
	require.NoError(t, err)
	s2.DelayedMessagesRead++
	accumulateN(t, s2, extra)

	// Rebuild cache from scratch on a new state with same accumulators
	var allMsgs []*DelayedInboxMessage
	allMsgs = append(allMsgs, batch1...)
	allMsgs = append(allMsgs, extra...)
	rebuilt := &State{
		DelayedMessagesRead:     s2.DelayedMessagesRead,
		DelayedMessagesSeen:     s2.DelayedMessagesSeen,
		DelayedMessageInboxAcc:  s2.DelayedMessageInboxAcc,
		DelayedMessageOutboxAcc: s2.DelayedMessageOutboxAcc,
	}
	fetcher := makeFetcher(allMsgs[rebuilt.DelayedMessagesRead:], rebuilt.DelayedMessagesRead)
	require.NoError(t, rebuilt.RebuildDelayedMsgPreimages(fetcher))

	// Pop all from rebuilt state and verify hashes match originals
	var rebuiltHashes []common.Hash
	for rebuilt.DelayedMessagesRead < rebuilt.DelayedMessagesSeen {
		h, err := rebuilt.PopDelayedOutbox()
		require.NoError(t, err)
		rebuiltHashes = append(rebuiltHashes, h)
		rebuilt.DelayedMessagesRead++
	}

	require.Equal(t, originalHashes[1:], rebuiltHashes)
}
