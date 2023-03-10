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

	blockNum := uint64(3)
	fromBlock := uint64(3)
	toBlock := uint64(4)
	honestCommit, err := honestManager.BigStepLeafCommitment(
		ctx,
		blockNum,
		fromBlock,
		toBlock,
		honestRoots[fromBlock],
		honestRoots[toBlock],
	)
	require.NoError(t, err)
	t.Logf("%+v", honestCommit)

	divergenceHeight := uint64(4)
	evilStates, evilRoots, evilCounts := setupStates(t, numStates, divergenceHeight)
	evilManager, err := NewWithAssertionStates(
		evilStates,
		evilCounts,
		WithBigStepStateDivergenceHeight(divergenceHeight),
		WithMaxWavmOpcodesPerBlock(maxOpcodesPerBlock),
		WithNumOpcodesPerBigStep(bigStepSize),
	)
	require.NoError(t, err)
	evilCommit, err := evilManager.BigStepLeafCommitment(
		ctx,
		blockNum,
		fromBlock,
		toBlock,
		evilRoots[fromBlock],
		evilRoots[toBlock],
	)
	require.NoError(t, err)
	t.Logf("%+v", evilCommit)
}

func setupStates(t *testing.T, numStates, divergenceHeight uint64) ([]*protocol.ExecutionState, []common.Hash, []*big.Int) {
	t.Helper()
	states := make([]*protocol.ExecutionState, numStates)
	roots := make([]common.Hash, numStates)
	inboxCounts := make([]*big.Int, numStates)
	for i := uint64(0); i < numStates; i++ {
		blockHash := crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
		if divergenceHeight > 0 && divergenceHeight >= i {
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
