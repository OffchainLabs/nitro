package solimpl

import (
	"context"
	"testing"

	"fmt"
	"math"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"math/rand"
)

var _ = protocol.ChallengeVertex(&ChallengeVertex{})

func TestChallengeVertex_ConfirmPsTimer(t *testing.T) {
	ctx := context.Background()
	tx := &activeTx{readWriteTx: true}
	height1 := uint64(7)
	height2 := uint64(3)
	a1, a2, challenge, chain1, _ := setupTopLevelFork(t, ctx, height1, height2)

	// We add two leaves to the challenge.
	honestHashes := honestHashesUpTo(10)
	evilHashes := evilHashesUpTo(10)
	honestManager := statemanager.New(honestHashes)
	evilManager := statemanager.New(evilHashes)
	honestCommit, err := honestManager.HistoryCommitmentUpTo(ctx, height1)
	require.NoError(t, err)
	evilCommit, err := evilManager.HistoryCommitmentUpTo(ctx, height2)
	require.NoError(t, err)

	// We add two leaves to the challenge.
	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a1,
		honestCommit,
	)
	require.NoError(t, err)
	_, err = challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a2,
		evilCommit,
	)
	require.NoError(t, err)

	t.Run("vertex ps timer has not exceeded challenge duration", func(t *testing.T) {
		require.ErrorIs(t, v1.ConfirmForPsTimer(ctx, tx), ErrPsTimerNotYet)
	})
	t.Run("vertex ps timer has exceeded challenge duration", func(t *testing.T) {
		t.Skip("TODO(RJ): Add customizable challenge period")
		backend, ok := chain1.backend.(*backends.SimulatedBackend)
		require.Equal(t, true, ok)
		for i := 0; i < 1000; i++ {
			backend.Commit()
		}
		require.NoError(t, v1.ConfirmForPsTimer(ctx, tx))
	})
}

func honestHashesUpTo(n uint64) []common.Hash {
	hashes := make([]common.Hash, n)
	for i := uint64(0); i < n; i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	return hashes
}

func evilHashesUpTo(n uint64) []common.Hash {
	hashes := make([]common.Hash, n)
	for i := uint64(0); i < n; i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", math.MaxUint64-i)))
	}
	return hashes
}

func divergingHashesStartingAt(t *testing.T, n uint64, hashes []common.Hash) []common.Hash {
	t.Helper()
	divergingHashes := make([]common.Hash, len(hashes))
	for i := uint64(0); i < n; i++ {
		divergingHashes[i] = hashes[i]
	}
	for i := n; i < uint64(len(divergingHashes)); i++ {
		junk := make([]byte, 32)
		_, err := rand.Read(junk)
		require.NoError(t, err)
		divergingHashes[i] = common.BytesToHash(junk)
	}
	return divergingHashes
}

func TestChallengeVertex_HasConfirmedSibling(t *testing.T) {
	ctx := context.Background()
	tx := &activeTx{readWriteTx: true}
	height1 := uint64(3)
	height2 := uint64(7)
	a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

	// We add two leaves to the challenge.
	honestHashes := honestHashesUpTo(10)
	evilHashes := evilHashesUpTo(10)
	honestManager := statemanager.New(honestHashes)
	evilManager := statemanager.New(evilHashes)
	honestCommit, err := honestManager.HistoryCommitmentUpTo(ctx, height1)
	require.NoError(t, err)
	evilCommit, err := evilManager.HistoryCommitmentUpTo(ctx, height2)
	require.NoError(t, err)

	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a1,
		honestCommit,
	)
	require.NoError(t, err)
	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a2,
		evilCommit,
	)
	require.NoError(t, err)

	backend, ok := chain.backend.(*backends.SimulatedBackend)
	require.Equal(t, true, ok)
	for i := 0; i < 1000; i++ {
		backend.Commit()
	}
	require.NoError(t, v1.ConfirmForPsTimer(ctx, tx))

	ok, err = v2.HasConfirmedSibling(ctx, tx)
	require.NoError(t, err)
	require.Equal(t, true, ok)
}

func TestChallengeVertex_IsPresumptiveSuccessor(t *testing.T) {
	ctx := context.Background()
	tx := &activeTx{readWriteTx: true}
	height1 := uint64(7)
	height2 := uint64(7)
	a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

	honestHashes := honestHashesUpTo(10)
	evilHashes := evilHashesUpTo(10)

	honestManager := statemanager.New(honestHashes)
	evilManager := statemanager.New(evilHashes)
	honestCommit, err := honestManager.HistoryCommitmentUpTo(ctx, height1)
	require.NoError(t, err)
	evilCommit, err := evilManager.HistoryCommitmentUpTo(ctx, height2)
	require.NoError(t, err)

	// We add two leaves to the challenge.
	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a1,
		honestCommit,
	)
	require.NoError(t, err)
	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a2,
		evilCommit,
	)
	require.NoError(t, err)

	t.Run("both are rivals, so no one is presumptive", func(t *testing.T) {
		isPs, err := v1.IsPresumptiveSuccessor(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, false, isPs)

		isPs, err = v2.IsPresumptiveSuccessor(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, false, isPs)
	})
	t.Run("the newly bisected vertex is now presumptive", func(t *testing.T) {
		preCommit, err := evilManager.HistoryCommitmentUpTo(ctx, 3)
		require.NoError(t, err)
		proof, err := evilManager.PrefixProof(ctx, 3, 7)
		require.NoError(t, err)

		bisectedToV, err := v2.Bisect(
			ctx,
			tx,
			preCommit,
			proof,
		)
		require.NoError(t, err)
		bisectedTo := bisectedToV.(*ChallengeVertex)

		require.Equal(t, uint64(3), bisectedTo.inner.Height.Uint64())

		// V1 should no longer be presumptive.
		isPs, err := v1.IsPresumptiveSuccessor(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, false, isPs)

		// Bisected to should be presumptive.
		isPs, err = bisectedTo.IsPresumptiveSuccessor(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, true, isPs)
	})
}

func TestChallengeVertex_ChildrenAreAtOneStepFork(t *testing.T) {
	ctx := context.Background()
	tx := &activeTx{readWriteTx: true}
	t.Run("children are one step away", func(t *testing.T) {
		height1 := uint64(1)
		height2 := uint64(1)
		a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

		honestHashes := honestHashesUpTo(10)
		evilHashes := evilHashesUpTo(10)
		honestManager := statemanager.New(honestHashes)
		evilManager := statemanager.New(evilHashes)
		honestCommit, err := honestManager.HistoryCommitmentUpTo(ctx, height1)
		require.NoError(t, err)
		evilCommit, err := evilManager.HistoryCommitmentUpTo(ctx, height2)
		require.NoError(t, err)

		// We add two leaves to the challenge.
		_, err = challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			honestCommit,
		)
		require.NoError(t, err)
		_, err = challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a2,
			evilCommit,
		)
		require.NoError(t, err)

		manager, err := chain.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		rootV, err := manager.GetVertex(ctx, tx, challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err := rootV.Unwrap().ChildrenAreAtOneStepFork(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, true, atOSF)
	})
	t.Run("different heights", func(t *testing.T) {
		height1 := uint64(3)
		height2 := uint64(7)
		a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

		honestHashes := honestHashesUpTo(10)
		evilHashes := evilHashesUpTo(10)
		honestManager := statemanager.New(honestHashes)
		evilManager := statemanager.New(evilHashes)
		honestCommit, err := honestManager.HistoryCommitmentUpTo(ctx, height1)
		require.NoError(t, err)
		evilCommit, err := evilManager.HistoryCommitmentUpTo(ctx, height2)
		require.NoError(t, err)

		// We add two leaves to the challenge.
		_, err = challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			honestCommit,
		)
		require.NoError(t, err)
		_, err = challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a2,
			evilCommit,
		)
		require.NoError(t, err)

		manager, err := chain.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		rootV, err := manager.GetVertex(ctx, tx, challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err := rootV.Unwrap().ChildrenAreAtOneStepFork(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, false, atOSF)
	})
	t.Run("two bisection leading to one step fork", func(t *testing.T) {
		t.Skip()
		height1 := uint64(2)
		height2 := uint64(2)
		a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

		honestHashes := honestHashesUpTo(height1)
		evilHashes := evilHashesUpTo(height2)
		honestCommit, err := util.NewHistoryCommitment(height1, honestHashes)
		require.NoError(t, err)
		evilCommit, err := util.NewHistoryCommitment(height2, evilHashes)
		require.NoError(t, err)

		// We add two leaves to the challenge.
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			honestCommit,
		)
		require.NoError(t, err)
		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a2,
			evilCommit,
		)
		require.NoError(t, err)

		manager, err := chain.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		rootV, err := manager.GetVertex(ctx, tx, challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err := rootV.Unwrap().ChildrenAreAtOneStepFork(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, false, atOSF)

		// We then bisect, and then the vertices we bisected to should
		// now be at one step forks, as they will be at height 1 while their
		// parent is at height 0.
		bisectedTo2V, err := v2.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{},
			make([]byte, 0),
		)
		require.NoError(t, err)
		bisectedTo2 := bisectedTo2V.(*ChallengeVertex)
		require.Equal(t, uint64(1), bisectedTo2.inner.Height.Uint64())

		bisectedTo1V, err := v1.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{},
			make([]byte, 0),
		)
		require.NoError(t, err)
		bisectedTo1 := bisectedTo1V.(*ChallengeVertex)
		require.Equal(t, uint64(1), bisectedTo1.inner.Height.Uint64())

		rootV, err = manager.GetVertex(ctx, tx, challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err = rootV.Unwrap().ChildrenAreAtOneStepFork(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, true, atOSF)
	})
}

func TestChallengeVertex_Bisect(t *testing.T) {
	ctx := context.Background()
	tx := &activeTx{readWriteTx: true}
	height1 := uint64(3)
	height2 := uint64(7)
	a1, a2, challenge, chain1, chain2 := setupTopLevelFork(t, ctx, height1, height2)

	honestHashes := honestHashesUpTo(10)
	evilHashes := evilHashesUpTo(10)
	honestManager := statemanager.New(honestHashes)
	evilManager := statemanager.New(evilHashes)
	honestCommit, err := honestManager.HistoryCommitmentUpTo(ctx, height1)
	require.NoError(t, err)
	evilCommit, err := evilManager.HistoryCommitmentUpTo(ctx, height2)
	require.NoError(t, err)

	// We add two leaves to the challenge.
	manager, err := chain1.CurrentChallengeManager(ctx, tx)
	require.NoError(t, err)
	challenge.manager = manager.(*ChallengeManager)
	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a1,
		honestCommit,
	)
	require.NoError(t, err)

	manager, err = chain2.CurrentChallengeManager(ctx, tx)
	require.NoError(t, err)
	challenge.manager = manager.(*ChallengeManager)
	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a2,
		evilCommit,
	)
	require.NoError(t, err)

	t.Run("vertex does not exist", func(t *testing.T) {
		vertex := &ChallengeVertex{
			id:      common.BytesToHash([]byte("junk")),
			manager: challenge.manager,
		}
		_, err = vertex.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: common.BytesToHash([]byte("nyan4")),
			},
			make([]byte, 0),
		)
		require.ErrorContains(t, err, "does not exist")
	})
	t.Run("winner already declared", func(t *testing.T) {
		t.Skip("Need to add winner capabilities in order to test")
	})
	t.Run("cannot bisect presumptive successor", func(t *testing.T) {
		// V1 should be the presumptive successor here.
		_, err = v1.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: common.BytesToHash([]byte("nyan4")),
			},
			make([]byte, 0),
		)
		require.ErrorContains(t, err, "Cannot bisect presumptive")
	})
	t.Run("presumptive successor already confirmable", func(t *testing.T) {
		manager, err = chain1.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		chalPeriod, err := manager.ChallengePeriodSeconds(ctx, tx)
		require.NoError(t, err)
		backend, ok := chain1.backend.(*backends.SimulatedBackend)
		require.Equal(t, true, ok)
		err = backend.AdjustTime(chalPeriod)
		require.NoError(t, err)

		preCommit, err := evilManager.HistoryCommitmentUpTo(ctx, 3)
		require.NoError(t, err)
		prefixProof, err := evilManager.PrefixProof(ctx, 3, 7)
		require.NoError(t, err)

		// We make a challenge period pass.
		_, err = v2.Bisect(
			ctx,
			tx,
			preCommit,
			prefixProof,
		)
		require.ErrorContains(t, err, "cannot set same height ps")
	})
	t.Run("invalid prefix history", func(t *testing.T) {
		t.Skip("Need to add proof capabilities in solidity in order to test")
	})
	t.Run("OK", func(t *testing.T) {
		height1 := uint64(3)
		height2 := uint64(7)
		a1, a2, challenge, chain1, chain2 := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		manager, err := chain1.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		challenge.manager = manager.(*ChallengeManager)
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			honestCommit,
		)
		require.NoError(t, err)

		manager, err = chain2.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		challenge.manager = manager.(*ChallengeManager)
		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a2,
			evilCommit,
		)
		require.NoError(t, err)

		preCommit, err := evilManager.HistoryCommitmentUpTo(ctx, 3)
		require.NoError(t, err)
		prefixProof, err := evilManager.PrefixProof(ctx, 3, 7)
		require.NoError(t, err)

		bisectedToV, err := v2.Bisect(
			ctx,
			tx,
			preCommit,
			prefixProof,
		)
		require.NoError(t, err)
		bisectedTo := bisectedToV.(*ChallengeVertex)
		require.Equal(t, uint64(3), bisectedTo.inner.Height.Uint64())

		bisectTo, err := util.BisectionPoint(0, 3)
		require.NoError(t, err)

		preCommit, err = honestManager.HistoryCommitmentUpTo(ctx, bisectTo)
		require.NoError(t, err)
		prefixProof, err = honestManager.PrefixProof(ctx, bisectTo, 3)
		require.NoError(t, err)

		bisectedToV, err = v1.Bisect(
			ctx,
			tx,
			preCommit,
			prefixProof,
		)
		require.NoError(t, err)
		bisectedTo = bisectedToV.(*ChallengeVertex)
		require.Equal(t, uint64(1), bisectedTo.inner.Height.Uint64())
	})
}

func TestChallengeVertex_Merge(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(7)
	height2 := uint64(7)
	a1, a2, challenge, chain1, chain2 := setupTopLevelFork(t, ctx, height1, height2)
	tx := &activeTx{readWriteTx: true}

	honestHashes := honestHashesUpTo(10)
	evilHashes := divergingHashesStartingAt(t, 5, honestHashes)
	honestManager := statemanager.New(honestHashes)
	evilManager := statemanager.New(evilHashes)
	honestCommit, err := honestManager.HistoryCommitmentUpTo(ctx, height1)
	require.NoError(t, err)
	evilCommit, err := evilManager.HistoryCommitmentUpTo(ctx, height2)
	require.NoError(t, err)

	// We add two leaves to the challenge.
	manager, err := chain1.CurrentChallengeManager(ctx, tx)
	require.NoError(t, err)
	challenge.manager = manager.(*ChallengeManager)
	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a1,
		honestCommit,
	)
	require.NoError(t, err)

	manager, err = chain2.CurrentChallengeManager(ctx, tx)
	require.NoError(t, err)
	challenge.manager = manager.(*ChallengeManager)
	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a2,
		evilCommit,
	)
	require.NoError(t, err)

	t.Run("vertex does not exist", func(t *testing.T) {
		vertex := &ChallengeVertex{
			id:      common.BytesToHash([]byte("junk")),
			manager: challenge.manager,
		}
		_, err = vertex.Merge(
			ctx,
			tx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: common.BytesToHash([]byte("nyan4")),
			},
			make([]byte, 0),
		)
		require.ErrorContains(t, err, "does not exist")
	})
	t.Run("winner already declared", func(t *testing.T) {
		t.Skip("Need to add winner capabilities in order to test")
	})
	t.Run("presumptive successor already confirmable", func(t *testing.T) {
		backend, ok := chain1.backend.(*backends.SimulatedBackend)
		require.Equal(t, true, ok)

		preCommit, err := evilManager.HistoryCommitmentUpTo(ctx, 3)
		require.NoError(t, err)
		prefixProof, err := evilManager.PrefixProof(ctx, 3, 7)
		require.NoError(t, err)

		_, err = v2.Bisect(
			ctx,
			tx,
			preCommit,
			prefixProof,
		)
		require.NoError(t, err)

		for i := 0; i < 1000; i++ {
			backend.Commit()
		}

		preCommit, err = honestManager.HistoryCommitmentUpTo(ctx, 3)
		require.NoError(t, err)
		prefixProof, err = honestManager.PrefixProof(ctx, 3, 7)
		require.NoError(t, err)
		_, err = v1.Merge(
			ctx,
			tx,
			preCommit,
			prefixProof,
		)
		require.ErrorContains(t, err, "cannot set same height ps")
	})
	t.Run("invalid prefix history", func(t *testing.T) {
		t.Skip("Need to add proof capabilities in solidity in order to test")
	})
	t.Run("OK", func(t *testing.T) {
		height1 := uint64(7)
		height2 := uint64(7)
		a1, a2, challenge, chain1, chain2 := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		manager, err := chain1.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		challenge.manager = manager.(*ChallengeManager)
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			honestCommit,
		)
		require.NoError(t, err)

		manager, err = chain2.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		challenge.manager = manager.(*ChallengeManager)
		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a2,
			evilCommit,
		)
		require.NoError(t, err)

		preCommit, err := evilManager.HistoryCommitmentUpTo(ctx, 3)
		require.NoError(t, err)
		prefixProof, err := evilManager.PrefixProof(ctx, 3, 7)
		require.NoError(t, err)
		bisectedToV, err := v2.Bisect(
			ctx,
			tx,
			preCommit,
			prefixProof,
		)
		require.NoError(t, err)
		bisectedTo := bisectedToV.(*ChallengeVertex)
		require.Equal(t, uint64(3), bisectedTo.inner.Height.Uint64())

		preCommit, err = honestManager.HistoryCommitmentUpTo(ctx, 3)
		require.NoError(t, err)
		prefixProof, err = honestManager.PrefixProof(ctx, 3, 7)
		require.NoError(t, err)
		mergedToV, err := v1.Merge(
			ctx,
			tx,
			preCommit,
			prefixProof,
		)
		require.NoError(t, err)

		mergedTo := mergedToV.(*ChallengeVertex)
		require.Equal(t, bisectedTo.inner.HistoryRoot, mergedTo.inner.HistoryRoot)
	})
}

func TestChallengeVertex_CreateSubChallenge(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(7)
	height2 := uint64(7)
	tx := &activeTx{readWriteTx: true}

	t.Run("Error: vertex does not exist", func(t *testing.T) {
		_, _, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		vertex := &ChallengeVertex{
			id:      common.BytesToHash([]byte("junk")),
			manager: challenge.manager,
		}
		_, err := vertex.CreateSubChallenge(ctx, tx)
		require.ErrorContains(t, err, "execution reverted: Challenge does not exist")
	})

	honestHashes := honestHashesUpTo(10)
	evilHashes := divergingHashesStartingAt(t, 1, honestHashes)
	honestManager := statemanager.New(honestHashes)
	evilManager := statemanager.New(evilHashes)
	honestCommit, err := honestManager.HistoryCommitmentUpTo(ctx, height1)
	require.NoError(t, err)
	evilCommit, err := evilManager.HistoryCommitmentUpTo(ctx, height2)
	require.NoError(t, err)

	t.Run("Error: leaf can never be a fork candidate", func(t *testing.T) {
		a1, _, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			honestCommit,
		)
		require.NoError(t, err)
		_, err = v1.CreateSubChallenge(ctx, tx)
		require.ErrorContains(t, err, "execution reverted: Leaf can never be a fork candidate")
	})
	t.Run("Error: lowest height not one above the current height", func(t *testing.T) {
		a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			honestCommit,
		)
		require.NoError(t, err)
		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a2,
			evilCommit,
		)
		require.NoError(t, err)

		preCommit, err := evilManager.HistoryCommitmentUpTo(ctx, 3)
		require.NoError(t, err)
		prefixProof, err := evilManager.PrefixProof(ctx, 3, 7)
		require.NoError(t, err)
		bisectedToV, err := v2.Bisect(
			ctx,
			tx,
			preCommit,
			prefixProof,
		)
		require.NoError(t, err)
		bisectedTo := bisectedToV.(*ChallengeVertex)
		require.Equal(t, uint64(3), bisectedTo.inner.Height.Uint64())

		// Vertex must be in the protocol.
		_, err = challenge.manager.caller.GetVertex(challenge.manager.assertionChain.callOpts, bisectedTo.id)
		require.NoError(t, err)
		_, err = bisectedTo.CreateSubChallenge(ctx, tx)
		require.ErrorContains(t, err, "execution reverted: Lowest height not one above the current height")
	})
	t.Run("Error: has presumptive successor", func(t *testing.T) {
		a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

		height1 = uint64(1)
		height2 = uint64(3)
		honestHashes := honestHashesUpTo(10)
		evilHashes := divergingHashesStartingAt(t, 1, honestHashes)
		honestManager := statemanager.New(honestHashes)
		evilManager := statemanager.New(evilHashes)
		honestCommit, err := honestManager.HistoryCommitmentUpTo(ctx, height1)
		require.NoError(t, err)
		evilCommit, err := evilManager.HistoryCommitmentUpTo(ctx, height2)
		require.NoError(t, err)

		// We add two leaves to the challenge.
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			honestCommit,
		)
		require.NoError(t, err)

		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a2,
			evilCommit,
		)
		require.NoError(t, err)

		preCommit, err := evilManager.HistoryCommitmentUpTo(ctx, 1)
		require.NoError(t, err)
		prefixProof, err := evilManager.PrefixProof(ctx, 0, 1)
		require.NoError(t, err)

		_, err = v2.Bisect(
			ctx,
			tx,
			preCommit,
			prefixProof,
		)
		require.NoError(t, err)

		err = chain.Call(func(tx protocol.ActiveTx) error {
			rootVertex, err := v1.Prev(ctx, tx)
			require.NoError(t, err)
			_, err = rootVertex.Unwrap().CreateSubChallenge(ctx, tx)
			return err
		})
		require.ErrorContains(t, err, "execution reverted: Has presumptive successor")
	})
	t.Run("Can create succession challenge", func(t *testing.T) {
		a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			util.HistoryCommitment{
				Height:        height1,
				Merkle:        common.BytesToHash([]byte("nyan")),
				LastLeaf:      a1.inner.StateHash,
				LastLeafProof: []common.Hash{a1.inner.StateHash},
			},
		)
		require.NoError(t, err)

		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a2,
			util.HistoryCommitment{
				Height:        height2,
				Merkle:        common.BytesToHash([]byte("nyan2")),
				LastLeaf:      a2.inner.StateHash,
				LastLeafProof: []common.Hash{a2.inner.StateHash},
			},
		)
		require.NoError(t, err)

		v1Commit := common.BytesToHash([]byte("nyan"))
		v2Commit := common.BytesToHash([]byte("nyan2"))
		v2Height4V, err := v2.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: v2Commit,
			},
			make([]byte, 0),
		)
		require.NoError(t, err)
		v2Height4 := v2Height4V.(*ChallengeVertex)
		require.Equal(t, uint64(4), v2Height4.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height4.inner.HistoryRoot[:])

		v1Height4V, err := v1.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: v1Commit,
			},
			make([]byte, 0),
		)
		require.NoError(t, err)
		v1Height4 := v1Height4V.(*ChallengeVertex)
		require.Equal(t, uint64(4), v1Height4.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height4.inner.HistoryRoot[:])

		v2Height2V, err := v2Height4.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{
				Height: 2,
				Merkle: v2Commit,
			},
			make([]byte, 0),
		)
		require.NoError(t, err)
		v2Height2 := v2Height2V.(*ChallengeVertex)
		require.Equal(t, uint64(2), v2Height2.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height2.inner.HistoryRoot[:])

		v1Height2V, err := v1Height4.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{
				Height: 2,
				Merkle: v1Commit,
			},
			make([]byte, 0),
		)
		require.NoError(t, err)
		v1Height2 := v1Height2V.(*ChallengeVertex)
		require.Equal(t, uint64(2), v1Height2.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height2.inner.HistoryRoot[:])

		v1Height1V, err := v1Height2.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{
				Height: 1,
				Merkle: v1Commit,
			},
			make([]byte, 0),
		)
		require.NoError(t, err)
		v1Height1 := v1Height1V.(*ChallengeVertex)
		require.Equal(t, uint64(1), v1Height1.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height1.inner.HistoryRoot[:])

		v2Height1V, err := v2Height2.Bisect(
			ctx,
			tx,
			util.HistoryCommitment{
				Height: 1,
				Merkle: v2Commit,
			},
			make([]byte, 0),
		)
		require.NoError(t, err)
		v2Height1 := v2Height1V.(*ChallengeVertex)
		require.Equal(t, uint64(1), v2Height1.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height1.inner.HistoryRoot[:])

		genesisVertex, err := challenge.manager.caller.GetVertex(challenge.manager.assertionChain.callOpts, v2Height1.inner.PredecessorId)
		require.NoError(t, err)
		genesis := &ChallengeVertex{
			inner:   genesisVertex,
			id:      v2Height1.inner.PredecessorId,
			manager: challenge.manager,
		}
		_, err = genesis.CreateSubChallenge(context.Background(), tx)
		require.NoError(t, err)
	})
}

func TestChallengeVertex_AddSubChallengeLeaf(t *testing.T) {
	ctx := context.Background()
	tx := &activeTx{readWriteTx: true}
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a1,
		util.HistoryCommitment{
			Height:        height1,
			Merkle:        common.BytesToHash([]byte("nyan")),
			LastLeaf:      a1.inner.StateHash,
			LastLeafProof: []common.Hash{a1.inner.StateHash},
		},
	)
	require.NoError(t, err)

	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a2,
		util.HistoryCommitment{
			Height:        height2,
			Merkle:        common.BytesToHash([]byte("nyan2")),
			LastLeaf:      a2.inner.StateHash,
			LastLeafProof: []common.Hash{a2.inner.StateHash},
		},
	)
	require.NoError(t, err)

	v1Commit := common.BytesToHash([]byte("nyan"))
	v2Commit := common.BytesToHash([]byte("nyan2"))
	v2Height4V, err := v2.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 4,
			Merkle: v2Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v2Height4 := v2Height4V.(*ChallengeVertex)
	require.Equal(t, uint64(4), v2Height4.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height4.inner.HistoryRoot[:])

	v1Height4V, err := v1.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 4,
			Merkle: v1Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v1Height4 := v1Height4V.(*ChallengeVertex)
	require.Equal(t, uint64(4), v1Height4.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height4.inner.HistoryRoot[:])

	v2Height2V, err := v2Height4.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 2,
			Merkle: v2Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v2Height2 := v2Height2V.(*ChallengeVertex)
	require.Equal(t, uint64(2), v2Height2.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height2.inner.HistoryRoot[:])

	v1Height2V, err := v1Height4.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 2,
			Merkle: v1Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v1Height2 := v1Height2V.(*ChallengeVertex)
	require.Equal(t, uint64(2), v1Height2.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height2.inner.HistoryRoot[:])

	v1Height1V, err := v1Height2.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 1,
			Merkle: v1Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v1Height1 := v1Height1V.(*ChallengeVertex)
	require.Equal(t, uint64(1), v1Height1.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height1.inner.HistoryRoot[:])

	v2Height1V, err := v2Height2.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 1,
			Merkle: v2Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v2Height1 := v2Height1V.(*ChallengeVertex)
	require.Equal(t, uint64(1), v2Height1.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height1.inner.HistoryRoot[:])

	genesisVertex, err := challenge.manager.caller.GetVertex(challenge.manager.assertionChain.callOpts, v2Height1.inner.PredecessorId)
	require.NoError(t, err)
	genesis := &ChallengeVertex{
		inner:   genesisVertex,
		id:      v2Height1.inner.PredecessorId,
		manager: challenge.manager,
	}
	bigStepChal, err := genesis.CreateSubChallenge(context.Background(), tx)
	require.NoError(t, err)

	t.Run("empty history root", func(t *testing.T) {
		_, err = bigStepChal.AddSubChallengeLeaf(ctx, tx, v1Height1, util.HistoryCommitment{})
		require.ErrorContains(t, err, "execution reverted: Empty historyRoot")
	})
	t.Run("vertex does not exist", func(t *testing.T) {
		_, err = bigStepChal.AddSubChallengeLeaf(ctx, tx, &ChallengeVertex{
			id:      [32]byte{},
			manager: challenge.manager,
		}, util.HistoryCommitment{
			Height: 2,
			Merkle: v1Commit,
		})
		require.ErrorContains(t, err, "execution reverted: Vertex does not exist")
	})
	t.Run("claim has invalid succession challenge", func(t *testing.T) {
		_, err = bigStepChal.AddSubChallengeLeaf(ctx, tx, v1Height2, util.HistoryCommitment{
			Height: 2,
			Merkle: v1Commit,
		})
		require.ErrorContains(t, err, "execution reverted: Claim has invalid succession challenge")
	})
	t.Run("create sub challenge leaf rival 1", func(t *testing.T) {
		v, err := bigStepChal.AddSubChallengeLeaf(ctx, tx, v1Height1, util.HistoryCommitment{
			Height: 1,
			Merkle: v1Commit,
		})
		require.NoError(t, err)
		require.False(t, v.Id() == [32]byte{}) // Should have a non-empty ID
	})
	t.Run("create sub challenge leaf rival 2", func(t *testing.T) {
		v, err := bigStepChal.AddSubChallengeLeaf(ctx, tx, v2Height1, util.HistoryCommitment{
			Height: 1,
			Merkle: v2Commit,
		})
		require.NoError(t, err)
		require.False(t, v.Id() == [32]byte{}) // Should have a non-empty ID
	})
}

func TestChallengeVertex_CanConfirmSubChallenge(t *testing.T) {
	ctx := context.Background()
	tx := &activeTx{readWriteTx: true}
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a1,
		util.HistoryCommitment{
			Height:        height1,
			Merkle:        common.BytesToHash([]byte("nyan")),
			LastLeaf:      a1.inner.StateHash,
			LastLeafProof: []common.Hash{a1.inner.StateHash},
		},
	)
	require.NoError(t, err)

	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		tx,
		a2,
		util.HistoryCommitment{
			Height:        height2,
			Merkle:        common.BytesToHash([]byte("nyan2")),
			LastLeaf:      a2.inner.StateHash,
			LastLeafProof: []common.Hash{a2.inner.StateHash},
		},
	)
	require.NoError(t, err)

	v1Commit := common.BytesToHash([]byte("nyan"))
	v2Commit := common.BytesToHash([]byte("nyan2"))
	v2Height4V, err := v2.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 4,
			Merkle: v2Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v2Height4 := v2Height4V.(*ChallengeVertex)
	require.Equal(t, uint64(4), v2Height4.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height4.inner.HistoryRoot[:])

	v1Height4V, err := v1.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 4,
			Merkle: v1Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v1Height4 := v1Height4V.(*ChallengeVertex)
	require.Equal(t, uint64(4), v1Height4.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height4.inner.HistoryRoot[:])

	v2Height2V, err := v2Height4.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 2,
			Merkle: v2Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v2Height2 := v2Height2V.(*ChallengeVertex)
	require.Equal(t, uint64(2), v2Height2.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height2.inner.HistoryRoot[:])

	v1Height2V, err := v1Height4.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 2,
			Merkle: v1Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v1Height2 := v1Height2V.(*ChallengeVertex)
	require.Equal(t, uint64(2), v1Height2.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height2.inner.HistoryRoot[:])

	v1Height1V, err := v1Height2.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 1,
			Merkle: v1Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v1Height1 := v1Height1V.(*ChallengeVertex)
	require.Equal(t, uint64(1), v1Height1.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height1.inner.HistoryRoot[:])

	v2Height1V, err := v2Height2.Bisect(
		ctx,
		tx,
		util.HistoryCommitment{
			Height: 1,
			Merkle: v2Commit,
		},
		make([]byte, 0),
	)
	require.NoError(t, err)
	v2Height1 := v2Height1V.(*ChallengeVertex)
	require.Equal(t, uint64(1), v2Height1.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height1.inner.HistoryRoot[:])

	genesisVertex, err := challenge.manager.caller.GetVertex(challenge.manager.assertionChain.callOpts, v2Height1.inner.PredecessorId)
	require.NoError(t, err)
	genesis := &ChallengeVertex{
		inner:   genesisVertex,
		id:      v2Height1.inner.PredecessorId,
		manager: challenge.manager,
	}
	bigStepChal, err := genesis.CreateSubChallenge(context.Background(), tx)
	require.NoError(t, err)

	v, err := bigStepChal.AddSubChallengeLeaf(ctx, tx, v1Height1, util.HistoryCommitment{
		Height: 1,
		Merkle: v1Commit,
	})
	require.NoError(t, err)

	t.Run("can't confirm sub challenge", func(t *testing.T) {
		require.ErrorContains(t, v.ConfirmForPsTimer(ctx, tx), "ps timer has not exceeded challenge period")
	})
	t.Run("can confirm sub challenge", func(t *testing.T) {
		backend, ok := chain.backend.(*backends.SimulatedBackend)
		require.Equal(t, true, ok)
		for i := 0; i < 1000; i++ {
			backend.Commit()
		}
		require.NoError(t, v.ConfirmForPsTimer(ctx, tx))
	})
}
