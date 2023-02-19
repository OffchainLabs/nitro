package solimpl

import (
	"context"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestChallengeVertex_ConfirmPsTimer(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge, chain1, _ := setupTopLevelFork(t, ctx, height1, height2)

	genesis, err := chain1.AssertionByID(0)
	require.NoError(t, err)

	// We add two leaves to the challenge.
	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a1,
		util.HistoryCommitment{
			Height:    height1,
			Merkle:    common.BytesToHash([]byte("nyan")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)
	_, err = challenge.AddBlockChallengeLeaf(
		ctx,
		a2,
		util.HistoryCommitment{
			Height:    height2,
			Merkle:    common.BytesToHash([]byte("nyan2")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)

	t.Run("vertex ps timer has not exceeded challenge duration", func(t *testing.T) {
		require.ErrorIs(t, v1.ConfirmPsTimer(ctx), ErrPsTimerNotYet)
	})
	t.Run("vertex ps timer has exceeded challenge duration", func(t *testing.T) {
		backend, ok := chain1.backend.(*backends.SimulatedBackend)
		require.Equal(t, true, ok)
		for i := 0; i < 1000; i++ {
			backend.Commit()
		}
		require.NoError(t, v1.ConfirmPsTimer(ctx))
	})
}

func TestChallengeVertex_HasConfirmedSibling(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

	genesis, err := chain.AssertionByID(0)
	require.NoError(t, err)

	// We add two leaves to the challenge.
	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a1,
		util.HistoryCommitment{
			Height:    height1,
			Merkle:    common.BytesToHash([]byte("nyan")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)
	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a2,
		util.HistoryCommitment{
			Height:    height2,
			Merkle:    common.BytesToHash([]byte("nyan2")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)

	backend, ok := chain.backend.(*backends.SimulatedBackend)
	require.Equal(t, true, ok)
	for i := 0; i < 1000; i++ {
		backend.Commit()
	}
	require.NoError(t, v1.ConfirmPsTimer(ctx))

	ok, err = v2.HasConfirmedSibling(ctx)
	require.NoError(t, err)
	require.Equal(t, true, ok)
}

func TestChallengeVertex_IsPresumptiveSuccessor(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

	genesis, err := chain.AssertionByID(0)
	require.NoError(t, err)

	// We add two leaves to the challenge.
	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a1,
		util.HistoryCommitment{
			Height:    height1,
			Merkle:    common.BytesToHash([]byte("nyan")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)
	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
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
			ctx,
			util.HistoryCommitment{
				Height:    4,
				Merkle:    wantCommit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), bisectedTo.inner.Height.Uint64())

		// V1 should no longer be presumptive.
		isPs, err := v1.IsPresumptiveSuccessor(ctx)
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
		height1 := uint64(1)
		height2 := uint64(1)
		a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

		genesis, err := chain.AssertionByID(0)
		require.NoError(t, err)

		// We add two leaves to the challenge.
		_, err = challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)
		_, err = challenge.AddBlockChallengeLeaf(
			ctx,
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
		rootV, err := manager.GetVertex(challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err := rootV.ChildrenAreAtOneStepFork(ctx)
		require.NoError(t, err)
		require.Equal(t, true, atOSF)
	})
	t.Run("different heights", func(t *testing.T) {
		height1 := uint64(6)
		height2 := uint64(7)
		a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

		genesis, err := chain.AssertionByID(0)
		require.NoError(t, err)

		// We add two leaves to the challenge.
		_, err = challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)
		_, err = challenge.AddBlockChallengeLeaf(
			ctx,
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
		rootV, err := manager.GetVertex(challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err := rootV.ChildrenAreAtOneStepFork(ctx)
		require.NoError(t, err)
		require.Equal(t, false, atOSF)
	})
	t.Run("two bisection leading to one step fork", func(t *testing.T) {
		height1 := uint64(2)
		height2 := uint64(2)
		a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

		genesis, err := chain.AssertionByID(0)
		require.NoError(t, err)

		// We add two leaves to the challenge.
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)
		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
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
		rootV, err := manager.GetVertex(challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err := rootV.ChildrenAreAtOneStepFork(ctx)
		require.NoError(t, err)
		require.Equal(t, false, atOSF)

		// We then bisect, and then the vertices we bisected to should
		// now be at one step forks, as they will be at height 1 while their
		// parent is at height 0.
		commit := common.BytesToHash([]byte("nyan2"))
		bisectedTo2, err := v2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height:    1,
				Merkle:    commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), bisectedTo2.inner.Height.Uint64())

		commit = common.BytesToHash([]byte("nyan2fork"))
		bisectedTo1, err := v1.Bisect(
			ctx,
			util.HistoryCommitment{
				Height:    1,
				Merkle:    commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), bisectedTo1.inner.Height.Uint64())

		rootV, err = manager.GetVertex(challenge.inner.RootId)
		require.NoError(t, err)

		atOSF, err = rootV.ChildrenAreAtOneStepFork(ctx)
		require.NoError(t, err)
		require.Equal(t, true, atOSF)
	})
}

func TestChallengeVertex_Bisect(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge, chain1, chain2 := setupTopLevelFork(t, ctx, height1, height2)

	// We add two leaves to the challenge.
	manager, err := chain1.ChallengeManager()
	require.NoError(t, err)
	challenge.manager = manager
	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a1,
		util.HistoryCommitment{
			Height: height1,
			Merkle: common.BytesToHash([]byte("nyan")),
		},
	)
	require.NoError(t, err)

	manager, err = chain2.ChallengeManager()
	require.NoError(t, err)
	challenge.manager = manager
	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a2,
		util.HistoryCommitment{
			Height: height2,
			Merkle: common.BytesToHash([]byte("nyan2")),
		},
	)
	require.NoError(t, err)

	t.Run("vertex does not exist", func(t *testing.T) {
		vertex := &ChallengeVertex{
			id:      common.BytesToHash([]byte("junk")),
			manager: challenge.manager,
		}
		_, err = vertex.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: common.BytesToHash([]byte("nyan4")),
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
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: common.BytesToHash([]byte("nyan4")),
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "Cannot bisect presumptive")
	})
	t.Run("presumptive successor already confirmable", func(t *testing.T) {
		chalPeriod, err := chain1.ChallengePeriodSeconds()
		require.NoError(t, err)
		backend, ok := chain1.backend.(*backends.SimulatedBackend)
		require.Equal(t, true, ok)
		err = backend.AdjustTime(chalPeriod)
		require.NoError(t, err)

		// We make a challenge period pass.
		_, err = v2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: common.BytesToHash([]byte("nyan4")),
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "cannot set lower ps")
	})
	t.Run("invalid prefix history", func(t *testing.T) {
		t.Skip("Need to add proof capabilities in solidity in order to test")
	})
	t.Run("OK", func(t *testing.T) {
		height1 := uint64(6)
		height2 := uint64(7)
		a1, a2, challenge, chain1, chain2 := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		manager, err := chain1.ChallengeManager()
		require.NoError(t, err)
		challenge.manager = manager
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)

		manager, err = chain2.ChallengeManager()
		require.NoError(t, err)
		challenge.manager = manager
		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a2,
			util.HistoryCommitment{
				Height: height2,
				Merkle: common.BytesToHash([]byte("nyan2")),
			},
		)
		require.NoError(t, err)

		wantCommit := common.BytesToHash([]byte("nyan4"))
		bisectedTo, err := v2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: wantCommit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), bisectedTo.inner.Height.Uint64())
		require.Equal(t, wantCommit[:], bisectedTo.inner.HistoryRoot[:])

		_, err = v1.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: wantCommit,
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "already exists")
	})
}

func TestChallengeVertex_Merge(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge, chain1, chain2 := setupTopLevelFork(t, ctx, height1, height2)

	// We add two leaves to the challenge.
	manager, err := chain1.ChallengeManager()
	require.NoError(t, err)
	challenge.manager = manager
	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a1,
		util.HistoryCommitment{
			Height: height1,
			Merkle: common.BytesToHash([]byte("nyan")),
		},
	)
	require.NoError(t, err)

	manager, err = chain2.ChallengeManager()
	require.NoError(t, err)
	challenge.manager = manager
	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a2,
		util.HistoryCommitment{
			Height: height2,
			Merkle: common.BytesToHash([]byte("nyan2")),
		},
	)
	require.NoError(t, err)

	t.Run("vertex does not exist", func(t *testing.T) {
		vertex := &ChallengeVertex{
			id:      common.BytesToHash([]byte("junk")),
			manager: challenge.manager,
		}
		_, err = vertex.Merge(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: common.BytesToHash([]byte("nyan4")),
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "does not exist")
	})
	t.Run("winner already declared", func(t *testing.T) {
		t.Skip("Need to add winner capabilities in order to test")
	})
	t.Run("cannot merge presumptive successor", func(t *testing.T) {
		// V1 should be the presumptive successor here.
		_, err = v1.Merge(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: common.BytesToHash([]byte("nyan4")),
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "Cannot bisect presumptive")
	})
	t.Run("presumptive successor already confirmable", func(t *testing.T) {
		backend, ok := chain1.backend.(*backends.SimulatedBackend)
		require.Equal(t, true, ok)

		wantCommit := common.BytesToHash([]byte("nyan4"))
		_, err = v2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: wantCommit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)

		for i := 0; i < 1000; i++ {
			backend.Commit()
		}

		_, err = v1.Merge(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: wantCommit,
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "cannot set lower ps")
	})
	t.Run("invalid prefix history", func(t *testing.T) {
		t.Skip("Need to add proof capabilities in solidity in order to test")
	})
	t.Run("OK", func(t *testing.T) {
		height1 := uint64(6)
		height2 := uint64(7)
		a1, a2, challenge, chain1, chain2 := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		manager, err := chain1.ChallengeManager()
		require.NoError(t, err)
		challenge.manager = manager
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)

		manager, err = chain2.ChallengeManager()
		require.NoError(t, err)
		challenge.manager = manager
		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a2,
			util.HistoryCommitment{
				Height: height2,
				Merkle: common.BytesToHash([]byte("nyan2")),
			},
		)
		require.NoError(t, err)

		wantCommit := common.BytesToHash([]byte("nyan4"))
		bisectedTo, err := v2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: wantCommit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), bisectedTo.inner.Height.Uint64())
		require.Equal(t, wantCommit[:], bisectedTo.inner.HistoryRoot[:])

		mergedTo, err := v1.Merge(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: wantCommit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)

		require.Equal(t, bisectedTo.inner.HistoryRoot, mergedTo.inner.HistoryRoot)
	})
}

func TestChallengeVertex_CreateSubChallenge(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)

	t.Run("Error: vertex does not exist", func(t *testing.T) {
		_, _, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		vertex := &ChallengeVertex{
			id:      common.BytesToHash([]byte("junk")),
			manager: challenge.manager,
		}
		_, err := vertex.CreateSubChallenge(ctx)
		require.ErrorContains(t, err, "execution reverted: Fork candidate vertex does not exist")
	})
	t.Run("Error: leaf can never be a fork candidate", func(t *testing.T) {
		a1, _, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)
		_, err = v1.CreateSubChallenge(ctx)
		require.ErrorContains(t, err, "execution reverted: Leaf can never be a fork candidate")
	})
	t.Run("Error: lowest height not one above the current height", func(t *testing.T) {
		a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)
		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a2,
			util.HistoryCommitment{
				Height: height2,
				Merkle: common.BytesToHash([]byte("nyan2")),
			},
		)
		require.NoError(t, err)
		wantCommit := common.BytesToHash([]byte("nyan2"))
		bisectedTo, err := v2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: wantCommit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), bisectedTo.inner.Height.Uint64())
		require.Equal(t, wantCommit[:], bisectedTo.inner.HistoryRoot[:])
		// Vertex must be in the protocol.
		_, err = challenge.manager.caller.GetVertex(challenge.manager.assertionChain.callOpts, bisectedTo.id)
		require.NoError(t, err)
		_, err = bisectedTo.CreateSubChallenge(ctx)
		require.ErrorContains(t, err, "execution reverted: Lowest height not one above the current height")
	})
	t.Run("Error: has presumptive successor", func(t *testing.T) {
		a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)

		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a2,
			util.HistoryCommitment{
				Height: height2,
				Merkle: common.BytesToHash([]byte("nyan2")),
			},
		)
		require.NoError(t, err)

		v1Commit := common.BytesToHash([]byte("nyan"))
		v2Commit := common.BytesToHash([]byte("nyan2"))
		v2Height4, err := v2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: v2Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), v2Height4.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height4.inner.HistoryRoot[:])

		v1Height4, err := v1.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: v1Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), v1Height4.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height4.inner.HistoryRoot[:])

		v2Height2, err := v2Height4.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 2,
				Merkle: v2Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), v2Height2.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height2.inner.HistoryRoot[:])

		v1Height2, err := v1Height4.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 2,
				Merkle: v1Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), v1Height2.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height2.inner.HistoryRoot[:])

		v1Height1, err := v1Height2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 1,
				Merkle: v1Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), v1Height1.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height1.inner.HistoryRoot[:])

		_, err = v1Height1.CreateSubChallenge(ctx)
		require.ErrorContains(t, err, "execution reverted: Has presumptive successor")
	})
	t.Run("Can create succession challenge", func(t *testing.T) {
		a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		v1, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)

		v2, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a2,
			util.HistoryCommitment{
				Height: height2,
				Merkle: common.BytesToHash([]byte("nyan2")),
			},
		)
		require.NoError(t, err)

		v1Commit := common.BytesToHash([]byte("nyan"))
		v2Commit := common.BytesToHash([]byte("nyan2"))
		v2Height4, err := v2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: v2Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), v2Height4.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height4.inner.HistoryRoot[:])

		v1Height4, err := v1.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 4,
				Merkle: v1Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), v1Height4.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height4.inner.HistoryRoot[:])

		v2Height2, err := v2Height4.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 2,
				Merkle: v2Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), v2Height2.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height2.inner.HistoryRoot[:])

		v1Height2, err := v1Height4.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 2,
				Merkle: v1Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), v1Height2.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height2.inner.HistoryRoot[:])

		v1Height1, err := v1Height2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 1,
				Merkle: v1Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), v1Height1.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height1.inner.HistoryRoot[:])

		v2Height1, err := v2Height2.Bisect(
			ctx,
			util.HistoryCommitment{
				Height: 1,
				Merkle: v2Commit,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), v2Height1.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height1.inner.HistoryRoot[:])

		genesisVertex, err := challenge.manager.caller.GetVertex(challenge.manager.assertionChain.callOpts, v2Height1.inner.PredecessorId)
		require.NoError(t, err)
		genesis := &ChallengeVertex{
			inner:   genesisVertex,
			id:      v2Height1.inner.PredecessorId,
			manager: challenge.manager,
		}
		_, err = genesis.CreateSubChallenge(context.Background())
		require.NoError(t, err)
	})
}

func TestChallengeVertex_CreateSubChallengeLeaf(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a1,
		util.HistoryCommitment{
			Height: height1,
			Merkle: common.BytesToHash([]byte("nyan")),
		},
	)
	require.NoError(t, err)

	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a2,
		util.HistoryCommitment{
			Height: height2,
			Merkle: common.BytesToHash([]byte("nyan2")),
		},
	)
	require.NoError(t, err)

	v1Commit := common.BytesToHash([]byte("nyan"))
	v2Commit := common.BytesToHash([]byte("nyan2"))
	v2Height4, err := v2.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 4,
			Merkle: v2Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(4), v2Height4.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height4.inner.HistoryRoot[:])

	v1Height4, err := v1.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 4,
			Merkle: v1Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(4), v1Height4.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height4.inner.HistoryRoot[:])

	v2Height2, err := v2Height4.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 2,
			Merkle: v2Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(2), v2Height2.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height2.inner.HistoryRoot[:])

	v1Height2, err := v1Height4.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 2,
			Merkle: v1Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(2), v1Height2.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height2.inner.HistoryRoot[:])

	v1Height1, err := v1Height2.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 1,
			Merkle: v1Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), v1Height1.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height1.inner.HistoryRoot[:])

	v2Height1, err := v2Height2.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 1,
			Merkle: v2Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), v2Height1.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height1.inner.HistoryRoot[:])

	genesisVertex, err := challenge.manager.caller.GetVertex(challenge.manager.assertionChain.callOpts, v2Height1.inner.PredecessorId)
	require.NoError(t, err)
	genesis := &ChallengeVertex{
		inner:   genesisVertex,
		id:      v2Height1.inner.PredecessorId,
		manager: challenge.manager,
	}
	bigStepChal, err := genesis.CreateSubChallenge(context.Background())
	require.NoError(t, err)

	t.Run("empty history root", func(t *testing.T) {
		_, err = bigStepChal.AddBigStepChallengeLeaf(ctx, v1Height1, util.HistoryCommitment{})
		require.ErrorContains(t, err, "execution reverted: Empty historyRoot")
	})
	t.Run("vertex does not exist", func(t *testing.T) {
		_, err = bigStepChal.AddBigStepChallengeLeaf(ctx, &ChallengeVertex{
			id: [32]byte{},
		}, util.HistoryCommitment{
			Height: 2,
			Merkle: v1Commit,
		})
		require.ErrorContains(t, err, "execution reverted: Vertex does not exist")
	})
	t.Run("claim has invalid succession challenge", func(t *testing.T) {
		_, err = bigStepChal.AddBigStepChallengeLeaf(ctx, v1Height2, util.HistoryCommitment{
			Height: 2,
			Merkle: v1Commit,
		})
		require.ErrorContains(t, err, "execution reverted: Claim has invalid succession challenge")
	})
	t.Run("create sub challenge leaf rival 1", func(t *testing.T) {
		v, err := bigStepChal.AddBigStepChallengeLeaf(ctx, v1Height1, util.HistoryCommitment{
			Height: 1,
			Merkle: v1Commit,
		})
		require.NoError(t, err)
		require.False(t, v.id == [32]byte{}) // Should have a non-empty ID
	})
	t.Run("create sub challenge leaf rival 2", func(t *testing.T) {
		v, err := bigStepChal.AddBigStepChallengeLeaf(ctx, v2Height1, util.HistoryCommitment{
			Height: 1,
			Merkle: v2Commit,
		})
		require.NoError(t, err)
		require.False(t, v.id == [32]byte{}) // Should have a non-empty ID
	})
}

func TestChallengeVertex_CanConfirmSubChallenge(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

	v1, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a1,
		util.HistoryCommitment{
			Height: height1,
			Merkle: common.BytesToHash([]byte("nyan")),
		},
	)
	require.NoError(t, err)

	v2, err := challenge.AddBlockChallengeLeaf(
		ctx,
		a2,
		util.HistoryCommitment{
			Height: height2,
			Merkle: common.BytesToHash([]byte("nyan2")),
		},
	)
	require.NoError(t, err)

	v1Commit := common.BytesToHash([]byte("nyan"))
	v2Commit := common.BytesToHash([]byte("nyan2"))
	v2Height4, err := v2.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 4,
			Merkle: v2Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(4), v2Height4.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height4.inner.HistoryRoot[:])

	v1Height4, err := v1.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 4,
			Merkle: v1Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(4), v1Height4.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height4.inner.HistoryRoot[:])

	v2Height2, err := v2Height4.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 2,
			Merkle: v2Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(2), v2Height2.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height2.inner.HistoryRoot[:])

	v1Height2, err := v1Height4.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 2,
			Merkle: v1Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(2), v1Height2.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height2.inner.HistoryRoot[:])

	v1Height1, err := v1Height2.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 1,
			Merkle: v1Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), v1Height1.inner.Height.Uint64())
	require.Equal(t, v1Commit[:], v1Height1.inner.HistoryRoot[:])

	v2Height1, err := v2Height2.Bisect(
		ctx,
		util.HistoryCommitment{
			Height: 1,
			Merkle: v2Commit,
		},
		make([]common.Hash, 0),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), v2Height1.inner.Height.Uint64())
	require.Equal(t, v2Commit[:], v2Height1.inner.HistoryRoot[:])

	genesisVertex, err := challenge.manager.caller.GetVertex(challenge.manager.assertionChain.callOpts, v2Height1.inner.PredecessorId)
	require.NoError(t, err)
	genesis := &ChallengeVertex{
		inner:   genesisVertex,
		id:      v2Height1.inner.PredecessorId,
		manager: challenge.manager,
	}
	bigStepChal, err := genesis.CreateSubChallenge(context.Background())
	require.NoError(t, err)

	v, err := bigStepChal.AddBigStepChallengeLeaf(ctx, v1Height1, util.HistoryCommitment{
		Height: 1,
		Merkle: v1Commit,
	})
	require.NoError(t, err)

	t.Run("can't confirm sub challenge", func(t *testing.T) {
		require.ErrorContains(t, v.ConfirmPsTimer(ctx), "ps timer has not exceeded challenge period")
	})
	t.Run("can confirm sub challenge", func(t *testing.T) {
		backend, ok := chain.backend.(*backends.SimulatedBackend)
		require.Equal(t, true, ok)
		for i := 0; i < 1000; i++ {
			backend.Commit()
		}
		require.NoError(t, v.ConfirmPsTimer(ctx))
	})
}
