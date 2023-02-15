package solimpl

import (
	"context"
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestChallengeVertex_ConfirmPsTimer(t *testing.T) {
	chain, acc := setupAssertionChainWithChallengeManager(t)
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge := setupTopLevelFork(t, chain, height1, height2)

	genesis, err := chain.AssertionByID(common.Hash{})
	require.NoError(t, err)

	// We add two leaves to the challenge.
	v1, err := challenge.AddLeaf(
		a1,
		util.HistoryCommitment{
			Height:    height1,
			Merkle:    common.BytesToHash([]byte("nyan")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)
	_, err = challenge.AddLeaf(
		a2,
		util.HistoryCommitment{
			Height:    height2,
			Merkle:    common.BytesToHash([]byte("nyan2")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)

	t.Run("vertex ps timer has not exceeded challenge duration", func(t *testing.T) {
		require.ErrorIs(t, v1.ConfirmPsTimer(context.Background()), ErrPsTimerNotYet)
	})
	t.Run("vertex ps timer has exceeded challenge duration", func(t *testing.T) {
		require.NoError(t, acc.backend.AdjustTime(time.Second*2000))
		require.NoError(t, v1.ConfirmPsTimer(context.Background()))
	})
}

func TestChallengeVertex_HasConfirmedSibling(t *testing.T) {
	ctx := context.Background()
	chain, acc := setupAssertionChainWithChallengeManager(t)
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge := setupTopLevelFork(t, chain, height1, height2)

	genesis, err := chain.AssertionByID(common.Hash{})
	require.NoError(t, err)

	// We add two leaves to the challenge.
	v1, err := challenge.AddLeaf(
		a1,
		util.HistoryCommitment{
			Height:    height1,
			Merkle:    common.BytesToHash([]byte("nyan")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)
	v2, err := challenge.AddLeaf(
		a2,
		util.HistoryCommitment{
			Height:    height2,
			Merkle:    common.BytesToHash([]byte("nyan2")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)
	require.NoError(t, acc.backend.AdjustTime(time.Second*2000))
	require.NoError(t, v1.ConfirmPsTimer(ctx))

	manager, err := chain.ChallengeManager()
	require.NoError(t, err)
	_, err = manager.vertexById(v1.id)
	require.NoError(t, err)
	v2, err = manager.vertexById(v2.id)
	require.NoError(t, err)

	ok, err := v2.HasConfirmedSibling(ctx)
	require.NoError(t, err)
	require.Equal(t, true, ok)
}

func TestChallengeVertex_IsPresumptiveSuccessor(t *testing.T) {
	ctx := context.Background()
	chain, _ := setupAssertionChainWithChallengeManager(t)
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge := setupTopLevelFork(t, chain, height1, height2)

	genesis, err := chain.AssertionByID(common.Hash{})
	require.NoError(t, err)

	// We add two leaves to the challenge.
	v1, err := challenge.AddLeaf(
		a1,
		util.HistoryCommitment{
			Height:    height1,
			Merkle:    common.BytesToHash([]byte("nyan")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)
	v2, err := challenge.AddLeaf(
		a2,
		util.HistoryCommitment{
			Height:    height2,
			Merkle:    common.BytesToHash([]byte("nyan2")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)

	t.Run("first to act is now presumptive", func(t *testing.T) {
		isPs, err := v1.IsPresumptiveSuccessor(ctx)
		require.NoError(t, err)
		require.Equal(t, true, isPs)

		isPs, err = v2.IsPresumptiveSuccessor(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isPs)
	})
	t.Run("the newly bisected vertex is now presumptive", func(t *testing.T) {
		wantCommit := common.BytesToHash([]byte("nyan2"))
		bisectedTo, err := v2.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    wantCommit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), bisectedTo.inner.Height.Uint64())

		manager, err := chain.ChallengeManager()
		require.NoError(t, err)
		v1, err = manager.vertexById(v1.id)
		require.NoError(t, err)
		v2, err = manager.vertexById(v2.id)
		require.NoError(t, err)

		// V1 and V2 should not longer be presumptive.
		isPs, err := v1.IsPresumptiveSuccessor(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isPs)

		isPs, err = v2.IsPresumptiveSuccessor(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isPs)

		// Bisected to should be presumptive.
		isPs, err = bisectedTo.IsPresumptiveSuccessor(ctx)
		require.NoError(t, err)
		require.Equal(t, true, isPs)
	})
}

func TestChallengeVertex_ChildrenAreAtOneStepFork(t *testing.T) {
	ctx := context.Background()
	t.Run("children are one step away", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		height1 := uint64(1)
		height2 := uint64(1)
		a1, a2, challenge := setupTopLevelFork(t, chain, height1, height2)

		genesis, err := chain.AssertionByID(common.Hash{})
		require.NoError(t, err)

		// We add two leaves to the challenge.
		_, err = challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)
		_, err = challenge.AddLeaf(
			a2,
			util.HistoryCommitment{
				Height:    height2,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)

		manager, err := chain.ChallengeManager()
		require.NoError(t, err)
		rootV, err := manager.vertexById(challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err := rootV.ChildrenAreAtOneStepFork(ctx)
		require.NoError(t, err)
		require.Equal(t, true, atOSF)
	})
	t.Run("different heights", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		height1 := uint64(6)
		height2 := uint64(7)
		a1, a2, challenge := setupTopLevelFork(t, chain, height1, height2)

		genesis, err := chain.AssertionByID(common.Hash{})
		require.NoError(t, err)

		// We add two leaves to the challenge.
		_, err = challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)
		_, err = challenge.AddLeaf(
			a2,
			util.HistoryCommitment{
				Height:    height2,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)

		manager, err := chain.ChallengeManager()
		require.NoError(t, err)
		rootV, err := manager.vertexById(challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err := rootV.ChildrenAreAtOneStepFork(ctx)
		require.NoError(t, err)
		require.Equal(t, false, atOSF)
	})
	t.Run("two bisection leading to one step fork", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		height1 := uint64(2)
		height2 := uint64(2)
		a1, a2, challenge := setupTopLevelFork(t, chain, height1, height2)

		genesis, err := chain.AssertionByID(common.Hash{})
		require.NoError(t, err)

		// We add two leaves to the challenge.
		v1, err := challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)
		v2, err := challenge.AddLeaf(
			a2,
			util.HistoryCommitment{
				Height:    height2,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)

		manager, err := chain.ChallengeManager()
		require.NoError(t, err)
		rootV, err := manager.vertexById(challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err := rootV.ChildrenAreAtOneStepFork(ctx)
		require.NoError(t, err)
		require.Equal(t, false, atOSF)

		// We then bisect, and then the vertices we bisected to should
		// now be at one step forks, as they will be at height 1 while their
		// parent is at height 0.
		commit := common.BytesToHash([]byte("nyan2"))
		bisectedTo2, err := v2.Bisect(
			util.HistoryCommitment{
				Height:    1,
				Merkle:    commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), bisectedTo2.inner.Height.Uint64())

		v1, err = manager.vertexById(v1.id)
		require.NoError(t, err)
		_, err = manager.vertexById(v2.id)
		require.NoError(t, err)

		commit = common.BytesToHash([]byte("nyan2fork"))
		bisectedTo1, err := v1.Bisect(
			util.HistoryCommitment{
				Height:    1,
				Merkle:    commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), bisectedTo1.inner.Height.Uint64())

		rootV, err = manager.vertexById(challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err = rootV.ChildrenAreAtOneStepFork(ctx)
		require.NoError(t, err)
		require.Equal(t, true, atOSF)
	})
}

func TestChallengeVertex_Bisect(t *testing.T) {
	chain, acc := setupAssertionChainWithChallengeManager(t)
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge := setupTopLevelFork(t, chain, height1, height2)

	genesis, err := chain.AssertionByID(common.Hash{})
	require.NoError(t, err)

	// We add two leaves to the challenge.
	v1, err := challenge.AddLeaf(
		a1,
		util.HistoryCommitment{
			Height:    height1,
			Merkle:    common.BytesToHash([]byte("nyan")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)
	v2, err := challenge.AddLeaf(
		a2,
		util.HistoryCommitment{
			Height:    height2,
			Merkle:    common.BytesToHash([]byte("nyan2")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)

	t.Run("vertex does not exist", func(t *testing.T) {
		vertex := &ChallengeVertex{
			id:      common.BytesToHash([]byte("junk")),
			manager: challenge.manager,
		}
		_, err = vertex.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "does not exist")
	})
	t.Run("winner already declared", func(t *testing.T) {
		t.Skip("Need to add winner capabilities in order to test")
	})
	t.Run("cannot bisect presumptive successor", func(t *testing.T) {
		// V1 should be the presumptive successor here.
		_, err = v1.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "Cannot bisect presumptive")
	})
	t.Run("presumptive successor already confirmable", func(t *testing.T) {
		chalPeriod, err := chain.ChallengePeriodSeconds()
		require.NoError(t, err)
		err = acc.backend.AdjustTime(chalPeriod)
		require.NoError(t, err)
		// We make a challenge period pass.
		_, err = v2.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "cannot set lower ps")
	})
	t.Run("invalid prefix history", func(t *testing.T) {
		t.Skip("Need to add proof capabilities in solidity in order to test")
	})
	t.Run("OK", func(t *testing.T) {
		chain, _ = setupAssertionChainWithChallengeManager(t)
		height1 = uint64(6)
		height2 = uint64(7)
		a1, a2, challenge = setupTopLevelFork(t, chain, height1, height2)

		// We add two leaves to the challenge.
		v1, err := challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)
		v2, err = challenge.AddLeaf(
			a2,
			util.HistoryCommitment{
				Height:    height2,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)
		wantCommit := common.BytesToHash([]byte("nyan2"))
		bisectedTo, err := v2.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    wantCommit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), bisectedTo.inner.Height.Uint64())
		require.Equal(t, wantCommit[:], bisectedTo.inner.HistoryRoot[:])
		// Vertex must be in the protocol.
		_, err = challenge.manager.caller.GetVertex(challenge.manager.assertionChain.callOpts, bisectedTo.id)
		require.NoError(t, err)

		_, err = v1.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    wantCommit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "already exists")
	})
}
