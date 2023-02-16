package solimpl

import (
	"context"
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestChallengeVertex_ConfirmPsTimer(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	a1, _, challenge, chain1, _ := setupTopLevelFork(t, ctx, height1, height2)

	genesis, err := chain1.AssertionByID(0)
	require.NoError(t, err)

	v1, err := challenge.AddLeaf(
		ctx,
		a1,
		util.HistoryCommitment{
			Height:    height1,
			Merkle:    common.BytesToHash([]byte("nyan")),
			FirstLeaf: genesis.inner.StateHash,
		},
	)
	require.NoError(t, err)

	t.Run("vertex ps timer has not exceeded challenge duration", func(t *testing.T) {
		t.Skip("Needs investigation")
		require.ErrorIs(t, v1.ConfirmPsTimer(ctx), ErrPsTimerNotYet)
	})
	t.Run("vertex ps timer has exceeded challenge duration", func(t *testing.T) {
		t.Skip("Needs investigation")
		backend, ok := chain1.backend.(*backends.SimulatedBackend)
		require.Equal(t, true, ok)
		require.NoError(t, backend.AdjustTime(time.Second*2000))
		require.NoError(t, v1.ConfirmPsTimer(ctx))
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
	v1, err := challenge.AddLeaf(
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
	v2, err := challenge.AddLeaf(
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
		v1, err := challenge.AddLeaf(
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
		v2, err := challenge.AddLeaf(
			ctx,
			a2,
			util.HistoryCommitment{
				Height: height2,
				Merkle: common.BytesToHash([]byte("nyan2")),
			},
		)
		require.NoError(t, err)

		curr, err := manager.caller.GetCurrentPsTimer(&bind.CallOpts{}, v1.id)
		require.NoError(t, err)
		t.Logf("V1 ps is %d", curr.Uint64())
		curr, err = manager.caller.GetCurrentPsTimer(&bind.CallOpts{}, v2.id)
		require.NoError(t, err)
		t.Logf("V2 ps is %d", curr.Uint64())

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
		err := vertex.CreateSubChallenge(ctx)
		require.ErrorContains(t, err, "execution reverted: Fork candidate vertex does not exist")
	})
	t.Run("Error: leaf can never be a fork candidate", func(t *testing.T) {
		a1, _, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		v1, err := challenge.AddLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)
		err = v1.CreateSubChallenge(ctx)
		require.ErrorContains(t, err, "execution reverted: Leaf can never be a fork candidate")
	})
	t.Run("Error: lowest height not one above the current height", func(t *testing.T) {
		t.Skip("Failing due to ps check, need investigation")
		a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		_, err := challenge.AddLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)
		v2, err := challenge.AddLeaf(
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
		require.ErrorContains(t, bisectedTo.CreateSubChallenge(context.Background()), "execution reverted: Lowest height not one above the current height")
	})
	t.Run("Error: has presumptive successor", func(t *testing.T) {
		t.Skip("Failing due to ps check, need investigation")
		a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		v1, err := challenge.AddLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)

		v2, err := challenge.AddLeaf(
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

		require.ErrorContains(t, v1Height1.CreateSubChallenge(context.Background()), "execution reverted: Has presumptive successor")
	})
	t.Run("Can create succession challenge", func(t *testing.T) {
		t.Skip("Failing due to ps check, need investigation")
		a1, a2, challenge, _, _ := setupTopLevelFork(t, ctx, height1, height2)

		// We add two leaves to the challenge.
		v1, err := challenge.AddLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)

		v2, err := challenge.AddLeaf(
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
		require.NoError(t, genesis.CreateSubChallenge(context.Background()))
	})
}
