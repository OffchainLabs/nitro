// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melreplay_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/mel-replay"
)

func createDelayedMessages(n int) []*mel.DelayedInboxMessage {
	msgs := make([]*mel.DelayedInboxMessage, n)
	for i := range n {
		reqID := common.BigToHash(big.NewInt(int64(i)))
		msgs[i] = &mel.DelayedInboxMessage{
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

// TestDelayedMessageRecordingAndReplayRoundTrip verifies the cross-mode
// invariant: preimages recorded during native mode (accumulate + pour + pop)
// are sufficient for replay mode to independently pour and read the same
// delayed messages. This catches divergence between the two pour
// implementations in state.go and delayed_message_db.go.
func TestDelayedMessageRecordingAndReplayRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name     string
		msgCount int
	}{
		{"single message", 1},
		{"multiple messages", 5},
		{"many messages", 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			msgs := createDelayedMessages(tc.msgCount)

			// ---- RECORDING PHASE (native mode) ----
			// Accumulate delayed messages with preimage recording enabled.
			preimages := make(daprovider.PreimagesMap)
			nativeState := &mel.State{}
			require.NoError(t, nativeState.RecordDelayedMsgPreimagesTo(preimages))

			for _, msg := range msgs {
				require.NoError(t, nativeState.AccumulateDelayedMessage(msg))
				nativeState.DelayedMessagesSeen++
			}

			// Snapshot state before pour — replay will start from here.
			replayState := &mel.State{
				DelayedMessagesRead:    nativeState.DelayedMessagesRead,
				DelayedMessagesSeen:    nativeState.DelayedMessagesSeen,
				DelayedMessageInboxAcc: nativeState.DelayedMessageInboxAcc,
				// OutboxAcc is zero — replay must pour on its own
			}

			// Pour and pop in native mode to record outbox preimages.
			require.NoError(t, nativeState.PourDelayedInboxToOutbox())
			for range tc.msgCount {
				_, err := nativeState.PopDelayedOutbox()
				require.NoError(t, err)
				nativeState.DelayedMessagesRead++
			}

			// ---- REPLAY PHASE (using only recorded preimages) ----
			resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)
			replayDB := melreplay.NewDelayedMessageDatabase(resolver)

			for i, expected := range msgs {
				got, err := replayDB.ReadDelayedMessage(replayState, uint64(i))
				require.NoError(t, err, "failed to read delayed message %d in replay mode", i)

				expectedBytes, err := rlp.EncodeToBytes(expected)
				require.NoError(t, err)
				gotBytes, err := rlp.EncodeToBytes(got)
				require.NoError(t, err)
				require.Equal(t, expectedBytes, gotBytes, "delayed message %d content mismatch between native and replay", i)

				replayState.DelayedMessagesRead++
			}

			// Both accumulators should be empty after reading all messages.
			require.Equal(t, common.Hash{}, replayState.DelayedMessageInboxAcc)
			require.Equal(t, common.Hash{}, replayState.DelayedMessageOutboxAcc)
		})
	}
}

// TestDelayedMessageReplayWithMixedInboxOutbox tests replay when messages
// span both inbox and outbox (some poured, then more accumulated).
func TestDelayedMessageReplayWithMixedInboxOutbox(t *testing.T) {
	t.Parallel()
	batch1 := createDelayedMessages(3)
	batch2 := createDelayedMessages(2)
	// Make batch2 distinct
	for i := range batch2 {
		reqID := common.BigToHash(big.NewInt(int64(100 + i)))
		batch2[i].Message.Header.RequestId = &reqID
		batch2[i].Message.L2msg = []byte(fmt.Sprintf("batch2-delayed-%d", i))
	}

	preimages := make(daprovider.PreimagesMap)
	nativeState := &mel.State{}
	require.NoError(t, nativeState.RecordDelayedMsgPreimagesTo(preimages))

	// Accumulate batch1
	for _, msg := range batch1 {
		require.NoError(t, nativeState.AccumulateDelayedMessage(msg))
		nativeState.DelayedMessagesSeen++
	}

	// Pour batch1 and read first message (partial pop)
	require.NoError(t, nativeState.PourDelayedInboxToOutbox())
	_, err := nativeState.PopDelayedOutbox()
	require.NoError(t, err)
	nativeState.DelayedMessagesRead++

	// Accumulate batch2 (now inbox has new messages, outbox has remaining from batch1)
	for _, msg := range batch2 {
		require.NoError(t, nativeState.AccumulateDelayedMessage(msg))
		nativeState.DelayedMessagesSeen++
	}

	// Snapshot for replay — mixed state: outbox has 2 from batch1, inbox has 2 from batch2
	replayState := &mel.State{
		DelayedMessagesRead:     nativeState.DelayedMessagesRead,
		DelayedMessagesSeen:     nativeState.DelayedMessagesSeen,
		DelayedMessageInboxAcc:  nativeState.DelayedMessageInboxAcc,
		DelayedMessageOutboxAcc: nativeState.DelayedMessageOutboxAcc,
	}

	// Continue native mode: pop remaining batch1, pour batch2, pop batch2
	for range 2 {
		_, err := nativeState.PopDelayedOutbox()
		require.NoError(t, err)
		nativeState.DelayedMessagesRead++
	}
	require.NoError(t, nativeState.PourDelayedInboxToOutbox())
	for range 2 {
		_, err := nativeState.PopDelayedOutbox()
		require.NoError(t, err)
		nativeState.DelayedMessagesRead++
	}

	// Replay: read all remaining messages
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)
	replayDB := melreplay.NewDelayedMessageDatabase(resolver)

	allExpected := append(batch1[1:], batch2...) // batch1[0] was already read
	for i, expected := range allExpected {
		msgIndex := replayState.DelayedMessagesRead
		got, err := replayDB.ReadDelayedMessage(replayState, msgIndex)
		require.NoError(t, err, "failed to read message at index %d (iteration %d)", msgIndex, i)

		expectedBytes, err := rlp.EncodeToBytes(expected)
		require.NoError(t, err)
		gotBytes, err := rlp.EncodeToBytes(got)
		require.NoError(t, err)
		require.Equal(t, expectedBytes, gotBytes, "message mismatch at index %d", msgIndex)

		replayState.DelayedMessagesRead++
	}

	require.Equal(t, common.Hash{}, replayState.DelayedMessageInboxAcc)
	require.Equal(t, common.Hash{}, replayState.DelayedMessageOutboxAcc)
}
