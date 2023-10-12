package stateprovider

import (
	"context"
	"testing"

	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	_ l2stateprovider.L2MessageStateCollector = (*L2StateBackend)(nil)
	_ l2stateprovider.MachineHashCollector    = (*L2StateBackend)(nil)
)

func TestHistoryCommitment(t *testing.T) {
	ctx := context.Background()
	wasmModuleRoot := common.Hash{}
	challengeLeafHeights := []l2stateprovider.Height{
		4,
		8,
		16,
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

	provider := l2stateprovider.NewHistoryCommitmentProvider(
		stateBackend,
		stateBackend,
		stateBackend,
		challengeLeafHeights,
		stateBackend,
	)
	t.Run("produces a block challenge commitment with height equal to leaf height const", func(t *testing.T) {
		got, err := provider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              wasmModuleRoot,
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.None[l2stateprovider.Height](),
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(challengeLeafHeights[0]), got.Height)
	})
	t.Run("produces a block challenge commitment with height up to", func(t *testing.T) {
		got, err := provider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              wasmModuleRoot,
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(l2stateprovider.Height(2)),
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), got.Height)
	})
	t.Run("produces a subchallenge history commitment with claims matching higher level start end leaves", func(t *testing.T) {
		blockChallengeCommit, err := provider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              wasmModuleRoot,
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(l2stateprovider.Height(1)),
			},
		)
		require.NoError(t, err)

		subChallengeCommit, err := provider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              wasmModuleRoot,
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{0},
				FromHeight:                  0,
				UpToHeight:                  option.None[l2stateprovider.Height](),
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
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              wasmModuleRoot,
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(l2stateprovider.Height(1)),
			},
		)
		require.NoError(t, err)

		smallStepSubchallengeCommit, err := provider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              wasmModuleRoot,
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{0, 0},
				FromHeight:                  0,
				UpToHeight:                  option.None[l2stateprovider.Height](),
			},
		)
		require.NoError(t, err)

		require.Equal(t, uint64(challengeLeafHeights[2]), smallStepSubchallengeCommit.Height)
		require.Equal(t, blockChallengeCommit.FirstLeaf, smallStepSubchallengeCommit.FirstLeaf)
	})
}
