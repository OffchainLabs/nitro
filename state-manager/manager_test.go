package statemanager

import (
	"context"
	"math/big"
	"testing"

	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"math/rand"
)

func TestDivergenceGranularity(t *testing.T) {
	ctx := context.Background()
	numStates := uint64(10)
	bigStepSize := uint64(10)
	maxOpcodesPerBlock := uint64(100)

	honestStates, honestRoots, honestCounts := setupStates(t, numStates, 0 /* honest */)
	honestManager, err := NewWithAssertionStates(
		honestStates,
		honestCounts,
		WithMaxWavmOpcodesPerBlock(maxOpcodesPerBlock),
		WithNumOpcodesPerBigStep(bigStepSize),
	)
	require.NoError(t, err)

	blockNum := uint64(1)
	fromBlock := uint64(1)
	toBlock := uint64(2)
	honestCommit, err := honestManager.BigStepLeafCommitment(
		ctx,
		blockNum,
		fromBlock,
		toBlock,
		honestRoots[fromBlock],
		honestRoots[toBlock],
	)
	require.NoError(t, err)

	t.Log("Big step leaf commitment height", honestCommit.Height)

	divergenceHeight := uint64(4)
	evilStates, evilRoots, evilCounts := setupStates(t, numStates, divergenceHeight)

	evilManager, err := NewWithAssertionStates(
		evilStates,
		evilCounts,
		WithBigStepStateDivergenceHeight(divergenceHeight),   // Diverges at the 4th big step.
		WithSmallStepStateDivergenceHeight(divergenceHeight), // Diverges at the 4th small step, within the 4th big step.
		WithMaxWavmOpcodesPerBlock(maxOpcodesPerBlock),
		WithNumOpcodesPerBigStep(bigStepSize),
	)
	require.NoError(t, err)

	// Big step challenge granularity.
	evilCommit, err := evilManager.BigStepLeafCommitment(
		ctx,
		blockNum,
		fromBlock,
		toBlock,
		evilRoots[fromBlock],
		evilRoots[toBlock],
	)
	require.NoError(t, err)

	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)

	// Check if big step commitments between the validators agree before the divergence height.
	checkHeight := divergenceHeight - 1
	honestCommit, err = honestManager.BigStepCommitmentUpTo(
		ctx,
		blockNum,
		honestRoots[fromBlock],
		honestRoots[toBlock],
		checkHeight,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.BigStepCommitmentUpTo(
		ctx,
		blockNum,
		evilRoots[fromBlock],
		evilRoots[toBlock],
		checkHeight,
	)
	require.NoError(t, err)
	require.Equal(t, honestCommit, evilCommit)

	// Check if big step commitments between the validators disagree starting at the divergence height.
	honestCommit, err = honestManager.BigStepCommitmentUpTo(
		ctx,
		blockNum,
		honestRoots[fromBlock],
		honestRoots[toBlock],
		divergenceHeight,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.BigStepCommitmentUpTo(
		ctx,
		blockNum,
		evilRoots[fromBlock],
		evilRoots[toBlock],
		divergenceHeight,
	)
	require.NoError(t, err)

	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)

	// Small step challenge granularity.
	fromBigStep := divergenceHeight - 1
	toBigStep := divergenceHeight
	honestCommit, err = honestManager.SmallStepLeafCommitment(
		ctx,
		blockNum,
		fromBigStep,
		toBigStep,
		honestRoots[fromBlock],
		honestCommit.LastLeaf,
	)
	require.NoError(t, err)

	evilCommit, err = evilManager.SmallStepLeafCommitment(
		ctx,
		blockNum,
		fromBigStep,
		toBigStep,
		evilRoots[fromBlock],
		evilCommit.LastLeaf,
	)
	require.NoError(t, err)

	t.Log("Small step leaf commitment height", honestCommit.Height)
	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)

	// Check if small step commitments between the validators agree before the divergence height.
	checkHeight = divergenceHeight - 1
	honestCommit, err = honestManager.SmallStepCommitmentUpTo(
		ctx,
		blockNum,
		honestRoots[fromBlock],
		honestCommit.LastLeaf,
		checkHeight,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.SmallStepCommitmentUpTo(
		ctx,
		blockNum,
		evilRoots[fromBlock],
		evilCommit.LastLeaf,
		checkHeight,
	)
	require.NoError(t, err)
	require.Equal(t, honestCommit, evilCommit)

	// Check if small step commitments between the validators disagree starting at the divergence height.
	honestCommit, err = honestManager.SmallStepCommitmentUpTo(
		ctx,
		blockNum,
		honestRoots[fromBlock],
		honestCommit.LastLeaf,
		divergenceHeight,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.SmallStepCommitmentUpTo(
		ctx,
		blockNum,
		evilRoots[fromBlock],
		evilCommit.LastLeaf,
		divergenceHeight,
	)
	require.NoError(t, err)

	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)
}

func setupStates(t *testing.T, numStates, divergenceHeight uint64) ([]*protocol.ExecutionState, []common.Hash, []*big.Int) {
	t.Helper()
	states := make([]*protocol.ExecutionState, numStates)
	roots := make([]common.Hash, numStates)
	inboxCounts := make([]*big.Int, numStates)
	for i := uint64(0); i < numStates; i++ {
		var blockHash common.Hash
		if divergenceHeight == 0 || i < divergenceHeight {
			blockHash = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
		} else {
			junkRoot := make([]byte, 32)
			_, err := rand.Read(junkRoot)
			require.NoError(t, err)
			blockHash = crypto.Keccak256Hash(junkRoot)
		}
		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: blockHash,
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}
		states[i] = state
		roots[i] = protocol.ComputeStateHash(state, big.NewInt(1))
		inboxCounts[i] = big.NewInt(1)
	}
	return states, roots, inboxCounts
}
