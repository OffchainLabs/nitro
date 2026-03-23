// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melreplay_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/mel-replay"
)

func TestRecordingMessagePreimagesAndReadingMessages(t *testing.T) {
	ctx := context.Background()
	var messages []*arbostypes.MessageWithMetadata
	numMsgs := uint64(10)
	for i := range numMsgs {
		messages = append(messages, &arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					BlockNumber: i,
					RequestId:   &common.Hash{},
					L1BaseFee:   common.Big0,
				},
			},
			DelayedMessagesRead: i,
		})
	}
	// Set up preimage recording before accumulating
	preimages := make(daprovider.PreimagesMap)
	state := &mel.State{}
	require.NoError(t, state.RecordMsgPreimagesTo(preimages))
	for i := range numMsgs {
		require.NoError(t, state.AccumulateMessage(messages[i]))
		state.MsgCount++
	}

	// Test reading in wasm mode
	msgReader := melreplay.NewMessageReader(
		melreplay.NewTypeBasedPreimageResolver(
			arbutil.Keccak256PreimageType,
			preimages,
		),
	)
	for i := range numMsgs {
		msg, err := msgReader.Read(ctx, state, i)
		require.NoError(t, err)
		require.Equal(t, msg.Hash(), messages[i].Hash())
	}
}

// testItem is a simple RLP-encodable struct for testing PeekFromAccumulator.
type testItem struct {
	Value uint64
}

// buildAccumulator pushes items into an accumulator hash chain using the same
// scheme as mel.State.AccumulateMessage: acc_new = H(acc_old || H(rlp(item))),
// recording preimages into the provided map.
func buildAccumulator(t *testing.T, items []testItem, preimages daprovider.PreimagesMap) common.Hash {
	t.Helper()
	keccakMap := preimages[arbutil.Keccak256PreimageType]
	acc := common.Hash{} // starts at zero
	for _, item := range items {
		itemBytes, err := rlp.EncodeToBytes(item)
		require.NoError(t, err)
		itemHash := crypto.Keccak256Hash(itemBytes)
		preimage := append(acc.Bytes(), itemHash.Bytes()...)
		newAcc := crypto.Keccak256Hash(preimage)
		keccakMap[newAcc] = preimage
		keccakMap[itemHash] = itemBytes
		acc = newAcc
	}
	return acc
}

func makePreimagesMap() daprovider.PreimagesMap {
	m := make(daprovider.PreimagesMap)
	m[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	return m
}

func TestPeekFromAccumulatorSingleItem(t *testing.T) {
	preimages := makePreimagesMap()
	items := []testItem{{Value: 42}}
	acc := buildAccumulator(t, items, preimages)

	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)
	result, err := melreplay.PeekFromAccumulator[testItem](context.Background(), resolver, acc, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(42), result.Value)
}

func TestPeekFromAccumulatorMultipleItems(t *testing.T) {
	preimages := makePreimagesMap()
	n := uint64(10)
	items := make([]testItem, n)
	for i := range uint64(n) {
		items[i] = testItem{Value: i * 100}
	}
	acc := buildAccumulator(t, items, preimages)
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)

	// Read every item by lookback position.
	// lookback=1 is the last pushed item (index n-1), lookback=n is the first (index 0).
	for i := range uint64(n) {
		lookback := n - i
		result, err := melreplay.PeekFromAccumulator[testItem](context.Background(), resolver, acc, lookback)
		require.NoError(t, err)
		require.Equal(t, i*100, result.Value, "mismatch at lookback %d (item index %d)", lookback, i)
	}
}

func TestPeekFromAccumulatorLastItemIsLookback1(t *testing.T) {
	preimages := makePreimagesMap()
	items := []testItem{{Value: 1}, {Value: 2}, {Value: 3}}
	acc := buildAccumulator(t, items, preimages)
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)

	// lookback=1 should return the most recently pushed item
	result, err := melreplay.PeekFromAccumulator[testItem](context.Background(), resolver, acc, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(3), result.Value)
}

func TestPeekFromAccumulatorFirstItemIsLookbackN(t *testing.T) {
	preimages := makePreimagesMap()
	items := []testItem{{Value: 1}, {Value: 2}, {Value: 3}}
	acc := buildAccumulator(t, items, preimages)
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)

	// lookback=3 (n) should return the first pushed item
	result, err := melreplay.PeekFromAccumulator[testItem](context.Background(), resolver, acc, 3)
	require.NoError(t, err)
	require.Equal(t, uint64(1), result.Value)
}
