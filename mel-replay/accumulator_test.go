// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melreplay_test

import (
	"context"
	"errors"
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
	ctx := context.Background()
	preimages := makePreimagesMap()
	items := []testItem{{Value: 42}}
	acc := buildAccumulator(t, items, preimages)

	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)
	result, err := melreplay.PeekFromAccumulator[testItem](ctx, resolver, acc, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(42), result.Value)
}

func TestPeekFromAccumulatorMultipleItems(t *testing.T) {
	ctx := context.Background()
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
		result, err := melreplay.PeekFromAccumulator[testItem](ctx, resolver, acc, lookback)
		require.NoError(t, err)
		require.Equal(t, i*100, result.Value, "mismatch at lookback %d (item index %d)", lookback, i)
	}
}

func TestPeekFromAccumulatorLastItemIsLookback1(t *testing.T) {
	ctx := context.Background()
	preimages := makePreimagesMap()
	items := []testItem{{Value: 1}, {Value: 2}, {Value: 3}}
	acc := buildAccumulator(t, items, preimages)
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)

	// lookback=1 should return the most recently pushed item
	result, err := melreplay.PeekFromAccumulator[testItem](ctx, resolver, acc, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(3), result.Value)
}

func TestPeekFromAccumulatorFirstItemIsLookbackN(t *testing.T) {
	ctx := context.Background()
	preimages := makePreimagesMap()
	items := []testItem{{Value: 1}, {Value: 2}, {Value: 3}}
	acc := buildAccumulator(t, items, preimages)
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)

	// lookback=3 (n) should return the first pushed item
	result, err := melreplay.PeekFromAccumulator[testItem](ctx, resolver, acc, 3)
	require.NoError(t, err)
	require.Equal(t, uint64(1), result.Value)
}

func TestCrossValidateBuildAccumulatorAndAccumulateMessage(t *testing.T) {
	n := uint64(5)
	messages := make([]*arbostypes.MessageWithMetadata, n)
	for i := range n {
		messages[i] = &arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					BlockNumber: i,
					RequestId:   &common.Hash{},
					L1BaseFee:   common.Big0,
				},
			},
			DelayedMessagesRead: i,
		}
	}

	// Path A: production AccumulateMessage
	prodPreimages := make(daprovider.PreimagesMap)
	state := &mel.State{}
	require.NoError(t, state.RecordMsgPreimagesTo(prodPreimages))
	for _, msg := range messages {
		require.NoError(t, state.AccumulateMessage(msg))
		state.MsgCount++
	}
	prodAcc := state.LocalMsgAccumulator

	// Path B: independent manual computation
	manualAcc := common.Hash{}
	for _, msg := range messages {
		msgBytes, err := rlp.EncodeToBytes(msg)
		require.NoError(t, err)
		msgHash := crypto.Keccak256Hash(msgBytes)
		manualAcc = crypto.Keccak256Hash(append(manualAcc.Bytes(), msgHash.Bytes()...))
	}

	require.Equal(t, manualAcc, prodAcc, "production AccumulateMessage and manual computation should match")

	// Verify messages can be read back from production preimages
	ctx := context.Background()
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, prodPreimages)
	for i := range n {
		result, err := melreplay.PeekFromAccumulator[arbostypes.MessageWithMetadata](ctx, resolver, prodAcc, n-i)
		require.NoError(t, err)
		require.Equal(t, messages[i].Hash(), result.Hash(), "message hash mismatch at index %d", i)
	}
}

func TestPeekFromAccumulatorLookbackZeroError(t *testing.T) {
	ctx := context.Background()
	preimages := makePreimagesMap()
	items := []testItem{{Value: 42}}
	acc := buildAccumulator(t, items, preimages)
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)

	_, err := melreplay.PeekFromAccumulator[testItem](ctx, resolver, acc, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "lookbacks must be >= 1")
}

func TestPeekFromAccumulatorLookbackExceedsChainLength(t *testing.T) {
	ctx := context.Background()
	preimages := makePreimagesMap()
	items := []testItem{{Value: 1}, {Value: 2}, {Value: 3}}
	acc := buildAccumulator(t, items, preimages)
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)

	_, err := melreplay.PeekFromAccumulator[testItem](ctx, resolver, acc, 4)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve preimage")
}

func TestPeekFromAccumulatorContextCancellation(t *testing.T) {
	preimages := makePreimagesMap()
	items := []testItem{{Value: 1}, {Value: 2}, {Value: 3}}
	acc := buildAccumulator(t, items, preimages)
	resolver := melreplay.NewTypeBasedPreimageResolver(arbutil.Keccak256PreimageType, preimages)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := melreplay.PeekFromAccumulator[testItem](ctx, resolver, acc, 2)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
}
