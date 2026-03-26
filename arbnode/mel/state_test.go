// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package mel

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
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
