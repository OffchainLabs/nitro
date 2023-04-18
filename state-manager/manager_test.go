package statemanager

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util/prefix-proofs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

var _ = Manager(&Simulated{})

func TestChallengeBoundaries_DifferentiateAssertionAndExecutionStates(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	_ = ctx
	manager, err := New(
		hashes,
		WithMaxWavmOpcodesPerBlock(8),
		WithNumOpcodesPerBigStep(8),
	)
	require.NoError(t, err)
	blockChalCommit, err := manager.HistoryCommitmentUpTo(ctx, 4)
	require.NoError(t, err)
	require.Equal(t, hashes[0], blockChalCommit.FirstLeaf)

	fromAssertionHeight := uint64(0)
	toAssertionHeight := fromAssertionHeight + 1
	bigStep, err := manager.BigStepLeafCommitment(
		ctx,
		fromAssertionHeight,
		toAssertionHeight,
	)
	require.NoError(t, err)
	require.NotEqual(t, hashes[0], bigStep.FirstLeaf)

	fromBigStep := uint64(0)
	toBigStep := fromBigStep + 1
	smallStep, err := manager.SmallStepLeafCommitment(
		ctx,
		fromAssertionHeight,
		toAssertionHeight,
		fromBigStep,
		toBigStep,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), bigStep.Height)
	require.Equal(t, uint64(7), smallStep.Height)
	require.Equal(t, bigStep.FirstLeaf, smallStep.FirstLeaf)
}

func TestGranularCommitments_SameStartHistory(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	_ = ctx
	manager, err := New(
		hashes,
		WithMaxWavmOpcodesPerBlock(56),
		WithNumOpcodesPerBigStep(8),
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
	toBlockChallengeHeight = fromBlockChallengeHeight + 1
	toBigStep := uint64(4)

	start, err = manager.BigStepCommitmentUpTo(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		toBigStep,
	)
	require.NoError(t, err)
	end, err = manager.BigStepLeafCommitment(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
	)
	require.NoError(t, err)
	require.Equal(t, start.FirstLeaf, end.FirstLeaf)
	require.NotEqual(t, start.LastLeaf, end.LastLeaf)
	require.NotEqual(t, start.Merkle, end.Merkle)

	fromBigStep := uint64(0)
	toBigStep = fromBigStep + 1
	toSmallStep := uint64(4)
	start, err = manager.SmallStepCommitmentUpTo(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep,
		toSmallStep,
	)
	require.NoError(t, err)
	end, err = manager.SmallStepLeafCommitment(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep,
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
	manager, err := New(
		hashes,
		WithMaxWavmOpcodesPerBlock(56),
		WithNumOpcodesPerBigStep(8),
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
	toBlockChallengeHeight = fromBlockChallengeHeight + 1
	fromBigStep := uint64(2)
	toBigStep := fromBigStep + 1

	start, err = manager.BigStepCommitmentUpTo(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		toBigStep,
	)
	require.NoError(t, err)
	end, err = manager.BigStepLeafCommitment(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
	)
	require.NoError(t, err)
	require.Equal(t, start.FirstLeaf, end.FirstLeaf)
	require.NotEqual(t, start.LastLeaf, end.LastLeaf)
	require.NotEqual(t, start.Merkle, end.Merkle)

	toSmallStep := uint64(6)
	start, err = manager.SmallStepCommitmentUpTo(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep,
		toSmallStep,
	)
	require.NoError(t, err)
	end, err = manager.SmallStepLeafCommitment(
		ctx,
		fromBlockChallengeHeight,
		toBlockChallengeHeight,
		fromBigStep,
		toBigStep,
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
	manager, err := New(
		hashes,
		WithMaxWavmOpcodesPerBlock(20),
		WithNumOpcodesPerBigStep(4),
	)
	require.NoError(t, err)

	from := uint64(2)
	to := uint64(3)

	loCommit, err := manager.HistoryCommitmentUpTo(ctx, from)
	require.NoError(t, err)
	hiCommit, err := manager.HistoryCommitmentUpTo(ctx, to)
	require.NoError(t, err)
	packedProof, err := manager.PrefixProof(ctx, from, to)
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

	bigCommit, err := manager.BigStepLeafCommitment(ctx, from, to)
	require.NoError(t, err)

	bigBisectCommit, err := manager.BigStepCommitmentUpTo(ctx, from, to, bigFrom)
	require.NoError(t, err)
	require.Equal(t, bigFrom, bigBisectCommit.Height)
	require.Equal(t, bigCommit.FirstLeaf, bigBisectCommit.FirstLeaf)

	bigProof, err := manager.BigStepPrefixProof(ctx, from, to, bigFrom, bigCommit.Height)
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

	smallCommit, err := manager.SmallStepLeafCommitment(ctx, from, to, bigFrom, bigFrom+1)
	require.NoError(t, err)

	smallFrom := uint64(2)

	smallBisectCommit, err := manager.SmallStepCommitmentUpTo(ctx, from, to, bigFrom, bigFrom+1, smallFrom)
	require.NoError(t, err)
	require.Equal(t, smallFrom, smallBisectCommit.Height)
	require.Equal(t, smallCommit.FirstLeaf, smallBisectCommit.FirstLeaf)

	smallProof, err := manager.SmallStepPrefixProof(ctx, from, to, bigFrom, bigFrom+1, smallFrom, smallCommit.Height)
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

	honestStates, _, honestCounts := setupStates(t, numStates, 0 /* honest */)
	honestManager, err := NewWithAssertionStates(
		honestStates,
		honestCounts,
		WithMaxWavmOpcodesPerBlock(maxOpcodesPerBlock),
		WithNumOpcodesPerBigStep(bigStepSize),
	)
	require.NoError(t, err)

	fromBlock := uint64(1)
	toBlock := uint64(2)
	honestCommit, err := honestManager.BigStepLeafCommitment(
		ctx,
		fromBlock,
		toBlock,
	)
	require.NoError(t, err)

	t.Log("Big step leaf commitment height", honestCommit.Height)

	divergenceHeight := uint64(3)
	evilStates, _, evilCounts := setupStates(t, numStates, divergenceHeight)

	evilManager, err := NewWithAssertionStates(
		evilStates,
		evilCounts,
		WithMaxWavmOpcodesPerBlock(maxOpcodesPerBlock),
		WithNumOpcodesPerBigStep(bigStepSize),
		WithBigStepStateDivergenceHeight(divergenceHeight),   // Diverges at the 3rd big step.
		WithSmallStepStateDivergenceHeight(divergenceHeight), // Diverges at the 3rd small step, within the 3rd big step.
	)
	require.NoError(t, err)

	// Big step challenge granularity.
	evilCommit, err := evilManager.BigStepLeafCommitment(
		ctx,
		fromBlock,
		toBlock,
	)
	require.NoError(t, err)

	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)

	// Check if big step commitments between the validators agree before the divergence height.
	checkHeight := divergenceHeight - 1
	honestCommit, err = honestManager.BigStepCommitmentUpTo(
		ctx,
		fromBlock,
		toBlock,
		checkHeight,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.BigStepCommitmentUpTo(
		ctx,
		fromBlock,
		toBlock,
		checkHeight,
	)
	require.NoError(t, err)
	require.Equal(t, honestCommit, evilCommit)

	t.Log("Big step commitments match before divergence height")

	// Check if big step commitments between the validators disagree starting at the divergence height.
	honestCommit, err = honestManager.BigStepCommitmentUpTo(
		ctx,
		fromBlock,
		toBlock,
		divergenceHeight,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.BigStepCommitmentUpTo(
		ctx,
		fromBlock,
		toBlock,
		divergenceHeight,
	)
	require.NoError(t, err)

	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)

	t.Log("Big step commitments diverge at divergence height")

	// Small step challenge granularity.
	fromBigStep := divergenceHeight - 1
	toBigStep := divergenceHeight
	honestCommit, err = honestManager.SmallStepLeafCommitment(
		ctx,
		fromBlock,
		toBlock,
		fromBigStep,
		toBigStep,
	)
	require.NoError(t, err)

	evilCommit, err = evilManager.SmallStepLeafCommitment(
		ctx,
		fromBlock,
		toBlock,
		fromBigStep,
		toBigStep,
	)
	require.NoError(t, err)

	require.Equal(t, honestCommit.Height, evilCommit.Height)
	require.Equal(t, honestCommit.FirstLeaf, evilCommit.FirstLeaf)
	require.NotEqual(t, honestCommit.Merkle, evilCommit.Merkle)

	t.Log("Small step commitments diverge at divergence height")

	// Check if small step commitments between the validators agree before the divergence height.
	toSmallStep := divergenceHeight - 1
	honestCommit, err = honestManager.SmallStepCommitmentUpTo(
		ctx,
		fromBlock,
		toBlock,
		fromBigStep,
		toBigStep,
		toSmallStep,
	)
	require.NoError(t, err)
	evilCommit, err = evilManager.SmallStepCommitmentUpTo(
		ctx,
		fromBlock,
		toBlock,
		fromBigStep,
		toBigStep,
		toSmallStep,
	)
	require.NoError(t, err)
	require.Equal(t, honestCommit, evilCommit)
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
		manager, err := New(leaves)
		require.NoError(t, err)

		packedProof, err := manager.PrefixProof(ctx, c.lo, c.hi)
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
