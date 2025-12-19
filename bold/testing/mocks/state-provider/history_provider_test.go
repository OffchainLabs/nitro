// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package stateprovider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/protocol"
	"github.com/offchainlabs/nitro/bold/state"
)

var (
	_ state.L2MessageStateCollector = (*L2StateBackend)(nil)
	_ state.MachineHashCollector    = (*L2StateBackend)(nil)
)

func simpleAssertionMetadata() *state.AssociatedAssertionMetadata {
	return &state.AssociatedAssertionMetadata{
		WasmModuleRoot: common.Hash{},
		FromState: protocol.GoGlobalState{
			Batch:      0,
			PosInBatch: 0,
		},
		BatchLimit: 1,
	}
}

func TestHistoryCommitment(t *testing.T) {
	ctx := context.Background()
	challengeLeafHeights := []state.Height{
		1 << 2,
		1 << 3,
		1 << 4,
	}
	numStates := uint64(10)
	states, _ := setupStates(t, numStates, 0 /* honest */)
	stateBackend, err := newTestingMachine(
		states,
		WithMaxWavmOpcodesPerBlock(uint64(challengeLeafHeights[1]*challengeLeafHeights[2])),
		WithMachineAtBlockProvider(mockMachineAtBlock),
		WithForceMachineBlockCompat(),
	)
	require.NoError(t, err)
	stateBackend.challengeLeafHeights = challengeLeafHeights

	provider := state.NewHistoryCommitmentProvider(
		stateBackend,
		stateBackend,
		stateBackend,
		challengeLeafHeights,
		stateBackend,
		nil,
	)
	t.Run("produces a block challenge commitment with height equal to leaf height const", func(t *testing.T) {
		got, err := provider.HistoryCommitment(
			ctx,
			&state.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []state.Height{},
				UpToHeight:                  option.None[state.Height](),
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(challengeLeafHeights[0]), got.Height)
	})
	t.Run("produces a block challenge commitment with height up to", func(t *testing.T) {
		got, err := provider.HistoryCommitment(
			ctx,
			&state.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []state.Height{},
				UpToHeight:                  option.Some(state.Height(2)),
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), got.Height)
	})
	t.Run("produces a subchallenge history commitment with claims matching higher level start end leaves", func(t *testing.T) {
		blockChallengeCommit, err := provider.HistoryCommitment(
			ctx,
			&state.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []state.Height{},
				UpToHeight:                  option.Some(state.Height(1)),
			},
		)
		require.NoError(t, err)

		subChallengeCommit, err := provider.HistoryCommitment(
			ctx,
			&state.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []state.Height{0},
				UpToHeight:                  option.None[state.Height](),
			},
		)
		require.NoError(t, err)

		require.Equal(t, uint64(challengeLeafHeights[1]), subChallengeCommit.Height)
		require.Equal(t, blockChallengeCommit.FirstLeaf, subChallengeCommit.FirstLeaf)
		require.Equal(t, blockChallengeCommit.LastLeaf, subChallengeCommit.LastLeaf)
	})
	t.Run("produces a small step challenge commit", func(t *testing.T) {
		blockChallengeCommit, err := provider.HistoryCommitment(
			ctx,
			&state.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []state.Height{},
				UpToHeight:                  option.Some(state.Height(1)),
			},
		)
		require.NoError(t, err)

		smallStepSubchallengeCommit, err := provider.HistoryCommitment(
			ctx,
			&state.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []state.Height{0, 0},
				UpToHeight:                  option.None[state.Height](),
			},
		)
		require.NoError(t, err)

		require.Equal(t, uint64(challengeLeafHeights[2]), smallStepSubchallengeCommit.Height)
		require.Equal(t, blockChallengeCommit.FirstLeaf, smallStepSubchallengeCommit.FirstLeaf)
	})
}
