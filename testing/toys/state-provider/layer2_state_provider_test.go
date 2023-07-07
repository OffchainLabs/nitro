// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE
package toys

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	prefixproofs "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/prefix-proofs"
	challenge_testing "github.com/OffchainLabs/challenge-protocol-v2/testing"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var _ = l2stateprovider.Provider(&L2StateBackend{})

func mockMachineAtBlock(_ context.Context, block uint64) (Machine, error) {
	blockBytes := make([]uint8, 8)
	binary.BigEndian.PutUint64(blockBytes, block)
	startState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: crypto.Keccak256Hash(blockBytes),
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	return NewSimpleMachine(startState, nil), nil
}

func TestChallengeBoundaries_DifferentiateAssertionAndExecutionStates(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	_ = ctx
	manager, err := NewWithMockedStateRoots(
		hashes,
		WithMaxWavmOpcodesPerBlock(8),
		WithNumOpcodesPerBigStep(8),
		WithMachineAtBlockProvider(mockMachineAtBlock),
		WithForceMachineBlockCompat(),
	)
	require.NoError(t, err)
	blockChalCommit, err := manager.HistoryCommitmentUpTo(ctx, 4)
	require.NoError(t, err)
	require.Equal(t, hashes[0], blockChalCommit.FirstLeaf)

	fromAssertionHeight := uint64(0)
	bigStep, err := manager.BigStepLeafCommitment(
		ctx,
		common.Hash{},
		fromAssertionHeight,
	)
	require.NoError(t, err)
	require.Equal(t, hashes[0], bigStep.FirstLeaf)
	require.NotEqual(t, bigStep.FirstLeaf, bigStep.LastLeaf)

	fromBigStep := uint64(0)
	smallStep, err := manager.SmallStepLeafCommitment(
		ctx,
		common.Hash{},
		fromAssertionHeight,
		fromBigStep,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), bigStep.Height)
	require.Equal(t, uint64(8), smallStep.Height)
	require.Equal(t, bigStep.FirstLeaf, smallStep.FirstLeaf)
}

func TestGranularCommitments_SameStartHistory(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	_ = ctx
	manager, err := NewWithMockedStateRoots(
		hashes,
		WithMaxWavmOpcodesPerBlock(56),
		WithNumOpcodesPerBigStep(8),
		WithMachineAtBlockProvider(mockMachineAtBlock),
		WithForceMachineBlockCompat(),
	)
	require.NoError(t, err)

	// Generating top-level, block challenge commitments.
	fromBlockChallengeHeight := uint64(4)
	toBlockChallengeHeight := uint64(7)
	start, err := manager.HistoryCommitmentUpTo(ctx, fromBlockChallengeHeight)
	require.NoError(t, err)
	end, err := manager.HistoryCommitmentUpTo(ctx, toBlockChallengeHeight)
	require.NoError(t, err)
	require.Equal(t, start.FirstLeaf, end.FirstLeaf)
	require.NotEqual(t, start.LastLeaf, end.LastLeaf)
	require.NotEqual(t, start.Merkle, end.Merkle)

	// Generating a big step challenge commitment
	// for all big WAVM steps between blocks 4 to 5.
	toBigStep := uint64(4)

	start, err = manager.BigStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlockChallengeHeight,
		toBigStep,
	)
	require.NoError(t, err)
	end, err = manager.BigStepLeafCommitment(
		ctx,
		common.Hash{},
		fromBlockChallengeHeight,
	)
	require.NoError(t, err)
	require.Equal(t, start.FirstLeaf, end.FirstLeaf)
	require.NotEqual(t, start.LastLeaf, end.LastLeaf)
	require.NotEqual(t, start.Merkle, end.Merkle)

	fromBigStep := uint64(0)
	toSmallStep := uint64(4)
	start, err = manager.SmallStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlockChallengeHeight,
		fromBigStep,
		toSmallStep,
	)
	require.NoError(t, err)
	end, err = manager.SmallStepLeafCommitment(
		ctx,
		common.Hash{},
		fromBlockChallengeHeight,
		fromBigStep,
	)
	require.NoError(t, err)
	require.Equal(t, start.FirstLeaf, end.FirstLeaf)
	require.NotEqual(t, start.LastLeaf, end.LastLeaf)
	require.NotEqual(t, start.Merkle, end.Merkle)
}

func TestGranularCommitments_DifferentStartPoints(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	_ = ctx
	manager, err := NewWithMockedStateRoots(
		hashes,
		WithMaxWavmOpcodesPerBlock(56),
		WithNumOpcodesPerBigStep(8),
		WithMachineAtBlockProvider(mockMachineAtBlock),
		WithForceMachineBlockCompat(),
	)
	require.NoError(t, err)

	// Generating top-level, block challenge commitments.
	fromBlockChallengeHeight := uint64(4)
	toBlockChallengeHeight := uint64(7)
	start, err := manager.HistoryCommitmentUpTo(ctx, fromBlockChallengeHeight)
	require.NoError(t, err)
	end, err := manager.HistoryCommitmentUpTo(ctx, toBlockChallengeHeight)
	require.NoError(t, err)
	require.Equal(t, start.FirstLeaf, end.FirstLeaf)
	require.NotEqual(t, start.LastLeaf, end.LastLeaf)
	require.NotEqual(t, start.Merkle, end.Merkle)

	// Generating a big step challenge commitment
	// for all big WAVM steps between blocks 4 to 5.
	fromBigStep := uint64(2)
	toBigStep := fromBigStep + 1

	start, err = manager.BigStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlockChallengeHeight,
		toBigStep,
	)
	require.NoError(t, err)
	end, err = manager.BigStepLeafCommitment(
		ctx,
		common.Hash{},
		fromBlockChallengeHeight,
	)
	require.NoError(t, err)
	require.Equal(t, start.FirstLeaf, end.FirstLeaf)
	require.NotEqual(t, start.LastLeaf, end.LastLeaf)
	require.NotEqual(t, start.Merkle, end.Merkle)

	toSmallStep := uint64(6)
	start, err = manager.SmallStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlockChallengeHeight,
		fromBigStep,
		toSmallStep,
	)
	require.NoError(t, err)
	end, err = manager.SmallStepLeafCommitment(
		ctx,
		common.Hash{},
		fromBlockChallengeHeight,
		fromBigStep,
	)
	require.NoError(t, err)
	require.Equal(t, start.FirstLeaf, end.FirstLeaf)
	require.NotEqual(t, start.LastLeaf, end.LastLeaf)
	require.NotEqual(t, start.Merkle, end.Merkle)
}

func TestAllPrefixProofs(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	manager, err := NewWithMockedStateRoots(
		hashes,
		WithMaxWavmOpcodesPerBlock(20),
		WithNumOpcodesPerBigStep(4),
		WithMachineAtBlockProvider(mockMachineAtBlock),
		WithForceMachineBlockCompat(),
	)
	require.NoError(t, err)

	from := uint64(2)
	to := uint64(3)

	loCommit, err := manager.HistoryCommitmentUpTo(ctx, from)
	require.NoError(t, err)
	hiCommit, err := manager.HistoryCommitmentUpTo(ctx, to)
	require.NoError(t, err)
	packedProof, err := manager.PrefixProofUpToBatch(ctx, 0, from, to, 1)
	require.NoError(t, err)

	data, err := ProofArgs.Unpack(packedProof)
	require.NoError(t, err)
	preExpansion := data[0].([][32]byte)
	proof := data[1].([][32]byte)

	preExpansionHashes := make([]common.Hash, len(preExpansion))
	for i := 0; i < len(preExpansion); i++ {
		preExpansionHashes[i] = preExpansion[i]
	}
	prefixProof := make([]common.Hash, len(proof))
	for i := 0; i < len(proof); i++ {
		prefixProof[i] = proof[i]
	}

	err = prefixproofs.VerifyPrefixProof(&prefixproofs.VerifyPrefixProofConfig{
		PreRoot:      loCommit.Merkle,
		PreSize:      from + 1,
		PostRoot:     hiCommit.Merkle,
		PostSize:     to + 1,
		PreExpansion: preExpansionHashes,
		PrefixProof:  prefixProof,
	})
	require.NoError(t, err)

	bigFrom := uint64(1)

	bigCommit, err := manager.BigStepLeafCommitment(ctx, common.Hash{}, from)
	require.NoError(t, err)

	bigBisectCommit, err := manager.BigStepCommitmentUpTo(ctx, common.Hash{}, from, bigFrom)
	require.NoError(t, err)
	require.Equal(t, bigFrom, bigBisectCommit.Height)
	require.Equal(t, bigCommit.FirstLeaf, bigBisectCommit.FirstLeaf)

	bigProof, err := manager.BigStepPrefixProof(ctx, common.Hash{}, from, bigFrom, bigCommit.Height)
	require.NoError(t, err)

	data, err = ProofArgs.Unpack(bigProof)
	require.NoError(t, err)
	preExpansion = data[0].([][32]byte)
	proof = data[1].([][32]byte)

	preExpansionHashes = make([]common.Hash, len(preExpansion))
	for i := 0; i < len(preExpansion); i++ {
		preExpansionHashes[i] = preExpansion[i]
	}
	prefixProof = make([]common.Hash, len(proof))
	for i := 0; i < len(proof); i++ {
		prefixProof[i] = proof[i]
	}

	computed, err := prefixproofs.Root(preExpansionHashes)
	require.NoError(t, err)
	require.Equal(t, bigBisectCommit.Merkle, computed)

	err = prefixproofs.VerifyPrefixProof(&prefixproofs.VerifyPrefixProofConfig{
		PreRoot:      bigBisectCommit.Merkle,
		PreSize:      bigFrom + 1,
		PostRoot:     bigCommit.Merkle,
		PostSize:     bigCommit.Height + 1,
		PreExpansion: preExpansionHashes,
		PrefixProof:  prefixProof,
	})
	require.NoError(t, err)

	smallCommit, err := manager.SmallStepLeafCommitment(ctx, common.Hash{}, from, bigFrom)
	require.NoError(t, err)

	smallFrom := uint64(2)

	smallBisectCommit, err := manager.SmallStepCommitmentUpTo(ctx, common.Hash{}, from, bigFrom, smallFrom)
	require.NoError(t, err)
	require.Equal(t, smallFrom, smallBisectCommit.Height)
	require.Equal(t, smallCommit.FirstLeaf, smallBisectCommit.FirstLeaf)

	smallProof, err := manager.SmallStepPrefixProof(ctx, common.Hash{}, from, bigFrom, smallFrom, smallCommit.Height)
	require.NoError(t, err)

	data, err = ProofArgs.Unpack(smallProof)
	require.NoError(t, err)
	preExpansion = data[0].([][32]byte)
	proof = data[1].([][32]byte)

	preExpansionHashes = make([]common.Hash, len(preExpansion))
	for i := 0; i < len(preExpansion); i++ {
		preExpansionHashes[i] = preExpansion[i]
	}
	prefixProof = make([]common.Hash, len(proof))
	for i := 0; i < len(proof); i++ {
		prefixProof[i] = proof[i]
	}

	computed, err = prefixproofs.Root(preExpansionHashes)
	require.NoError(t, err)
	require.Equal(t, smallBisectCommit.Merkle, computed)

	err = prefixproofs.VerifyPrefixProof(&prefixproofs.VerifyPrefixProofConfig{
		PreRoot:      smallBisectCommit.Merkle,
		PreSize:      smallFrom + 1,
		PostRoot:     smallCommit.Merkle,
		PostSize:     smallCommit.Height + 1,
		PreExpansion: preExpansionHashes,
		PrefixProof:  prefixProof,
	})
	require.NoError(t, err)
}

func TestDivergenceGranularity(t *testing.T) {
	ctx := context.Background()
	numStates := uint64(10)
	bigStepSize := uint64(10)
	maxOpcodesPerBlock := uint64(100)

	honestStates, _ := setupStates(t, numStates, 0 /* honest */)
	honestManager, err := newTestingMachine(
		honestStates,
		WithMaxWavmOpcodesPerBlock(maxOpcodesPerBlock),
		WithNumOpcodesPerBigStep(bigStepSize),
		WithMachineAtBlockProvider(mockMachineAtBlock),
		WithForceMachineBlockCompat(),
	)
	require.NoError(t, err)

	fromBlock := uint64(1)
	toBlock := uint64(2)
	honestCommit, err := honestManager.BigStepLeafCommitment(
		ctx,
		common.Hash{},
		fromBlock,
	)
	require.NoError(t, err)

	t.Log("Big step leaf commitment height", honestCommit.Height)

	divergenceHeight := toBlock
	evilStates, _ := setupStates(t, numStates, divergenceHeight)

	evilManager, err := newTestingMachine(
		evilStates,
		WithMaxWavmOpcodesPerBlock(maxOpcodesPerBlock),
		WithNumOpcodesPerBigStep(bigStepSize),
		WithBlockDivergenceHeight(toBlock),
		// Diverges at the 3rd small step, within the 3rd big step.
		WithMachineDivergenceStep(divergenceHeight+(divergenceHeight-1)*bigStepSize),
		WithMachineAtBlockProvider(mockMachineAtBlock),
		WithForceMachineBlockCompat(),
	)
	require.NoError(t, err)

	// Big step challenge granularity.
	evilCommit, err := evilManager.BigStepLeafCommitment(
		ctx,
		common.Hash{},
		fromBlock,
	)
	require.NoError(t, err)

	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.LastLeaf, evilCommit.LastLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)

	// Check if big step commitments between the validators agree before the divergence height.
	checkHeight := divergenceHeight - 1
	honestCommit, err = honestManager.BigStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlock,
		checkHeight,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.BigStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlock,
		checkHeight,
	)
	require.NoError(t, err)
	require.Equal(t, honestCommit, evilCommit)

	t.Log("Big step commitments match before divergence height")

	// Check if big step commitments between the validators disagree starting at the divergence height.
	honestCommit, err = honestManager.BigStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlock,
		divergenceHeight,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.BigStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlock,
		divergenceHeight,
	)
	require.NoError(t, err)

	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.LastLeaf, evilCommit.LastLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)

	t.Log("Big step commitments diverge at divergence height")

	// Small step challenge granularity.
	fromBigStep := divergenceHeight - 1
	honestCommit, err = honestManager.SmallStepLeafCommitment(
		ctx,
		common.Hash{},
		fromBlock,
		fromBigStep,
	)
	require.NoError(t, err)

	evilCommit, err = evilManager.SmallStepLeafCommitment(
		ctx,
		common.Hash{},
		fromBlock,
		fromBigStep,
	)
	require.NoError(t, err)

	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.LastLeaf, evilCommit.LastLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)

	t.Log("Small step commitments diverge at divergence height")

	// Check if small step commitments between the validators agree before the divergence height.
	toSmallStep := divergenceHeight - 1
	honestCommit, err = honestManager.SmallStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlock,
		fromBigStep,
		toSmallStep,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.SmallStepCommitmentUpTo(
		ctx,
		common.Hash{},
		fromBlock,
		fromBigStep,
		toSmallStep,
	)
	require.NoError(t, err)
	require.Equal(t, honestCommit, evilCommit)
}

func setupStates(t *testing.T, numStates, divergenceHeight uint64) ([]*protocol.ExecutionState, []common.Hash) {
	t.Helper()
	states := make([]*protocol.ExecutionState, numStates)
	roots := make([]common.Hash, numStates)
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
				BlockHash:  blockHash,
				Batch:      0,
				PosInBatch: i,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}
		if i+1 == numStates {
			state.GlobalState.Batch = 1
			state.GlobalState.PosInBatch = 0
		}
		states[i] = state
		roots[i] = protocol.ComputeSimpleMachineChallengeHash(state)
	}
	return states, roots
}

func TestPrefixProofs(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		lo uint64
		hi uint64
	}{
		{0, 1},
		{0, 3},
		{1, 2},
		{1, 3},
		{1, 15},
		{17, 255},
		{23, 255},
		{20, 511},
	} {
		leaves := hashesForUints(0, c.hi+1)
		manager, err := NewWithMockedStateRoots(leaves)
		require.NoError(t, err)

		packedProof, err := manager.PrefixProofUpToBatch(ctx, 0, c.lo, c.hi, 1)
		require.NoError(t, err)

		data, err := ProofArgs.Unpack(packedProof)
		require.NoError(t, err)
		preExpansion := data[0].([][32]byte)
		proof := data[1].([][32]byte)

		preExpansionHashes := make([]common.Hash, len(preExpansion))
		for i := 0; i < len(preExpansion); i++ {
			preExpansionHashes[i] = preExpansion[i]
		}
		prefixProof := make([]common.Hash, len(proof))
		for i := 0; i < len(proof); i++ {
			prefixProof[i] = proof[i]
		}

		postExpansion, err := manager.HistoryCommitmentUpTo(ctx, c.hi)
		require.NoError(t, err)

		root, err := prefixproofs.Root(preExpansionHashes)
		require.NoError(t, err)

		cfg := &prefixproofs.VerifyPrefixProofConfig{
			PreRoot:      root,
			PreSize:      c.lo + 1,
			PostRoot:     postExpansion.Merkle,
			PostSize:     c.hi + 1,
			PreExpansion: preExpansionHashes,
			PrefixProof:  prefixProof,
		}
		err = prefixproofs.VerifyPrefixProof(cfg)
		require.NoError(t, err)
	}
}

func newTestingMachine(
	assertionChainExecutionStates []*protocol.ExecutionState,
	opts ...Opt,
) (*L2StateBackend, error) {
	if len(assertionChainExecutionStates) == 0 {
		return nil, errors.New("must have execution states")
	}
	stateRoots := make([]common.Hash, len(assertionChainExecutionStates))
	var lastBatch uint64 = math.MaxUint64
	var lastPosInBatch uint64 = math.MaxUint64
	for i := 0; i < len(stateRoots); i++ {
		state := assertionChainExecutionStates[i]
		if state.GlobalState.Batch == lastBatch && state.GlobalState.PosInBatch == lastPosInBatch {
			return nil, fmt.Errorf("execution states %v and %v have the same batch %v and position in batch %v", i-1, i, lastBatch, lastPosInBatch)
		}
		lastBatch = state.GlobalState.Batch
		lastPosInBatch = state.GlobalState.PosInBatch
		stateRoots[i] = protocol.ComputeSimpleMachineChallengeHash(state)
	}
	s := &L2StateBackend{
		stateRoots:      stateRoots,
		executionStates: assertionChainExecutionStates,
		machineAtBlock: func(context.Context, uint64) (Machine, error) {
			return nil, errors.New("state manager created with NewWithAssertionStates() cannot provide machines")
		},
		levelZeroBlockEdgeHeight:     challenge_testing.LevelZeroBlockEdgeHeight,
		levelZeroBigStepEdgeHeight:   challenge_testing.LevelZeroBigStepEdgeHeight,
		levelZeroSmallStepEdgeHeight: challenge_testing.LevelZeroSmallStepEdgeHeight,
	}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}

func hashesForUints(lo, hi uint64) []common.Hash {
	var ret []common.Hash
	for i := lo; i < hi; i++ {
		ret = append(ret, hashForUint(i))
	}
	return ret
}

func hashForUint(x uint64) common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, x))
}
