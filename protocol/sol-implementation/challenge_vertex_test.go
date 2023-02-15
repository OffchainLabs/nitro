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
	a1, _, challenge := setupTopLevelFork(t, chain, height1, height2)

	genesis, err := chain.AssertionByID(common.Hash{})
	require.NoError(t, err)

	v1, err := challenge.AddLeaf(
		a1,
		util.HistoryCommitment{
			Height:    height1,
			Merkle:    common.BytesToHash([]byte("nyan")),
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

func TestChallengeVertex_CreateSubChallenge(t *testing.T) {
	ctx := context.Background()
	chain, _ := setupAssertionChainWithChallengeManager(t)
	height1 := uint64(6)
	height2 := uint64(7)
	a1, a2, challenge := setupTopLevelFork(t, chain, height1, height2)

	genesis, err := chain.AssertionByID(common.Hash{})
	require.NoError(t, err)

	t.Run("Error: vertex does not exist", func(t *testing.T) {
		vertex := &ChallengeVertex{
			id:      common.BytesToHash([]byte("junk")),
			manager: challenge.manager,
		}
		err = vertex.CreateSubChallenge(ctx)
		require.ErrorContains(t, err, "execution reverted: Fork candidate vertex does not exist")
	})
	t.Run("Error: leaf can never be a fork candidate", func(t *testing.T) {
		chain, _ = setupAssertionChainWithChallengeManager(t)
		height1 = uint64(6)
		height2 = uint64(7)
		a1, a2, challenge = setupTopLevelFork(t, chain, height1, height2)

		v1, err := challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)

		err = v1.CreateSubChallenge(ctx)
		require.ErrorContains(t, err, "execution reverted: Leaf can never be a fork candidate")
	})
	t.Run("Error: lowest height not one above the current height", func(t *testing.T) {
		chain, _ = setupAssertionChainWithChallengeManager(t)
		height1 = uint64(6)
		height2 = uint64(7)
		a1, a2, challenge = setupTopLevelFork(t, chain, height1, height2)

		// We add two leaves to the challenge.
		_, err := challenge.AddLeaf(
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
		require.ErrorContains(t, bisectedTo.CreateSubChallenge(context.Background()), "execution reverted: Lowest height not one above the current height")
	})
	t.Run("Error: has presumptive successor", func(t *testing.T) {
		chain, _ = setupAssertionChainWithChallengeManager(t)
		height1 = uint64(8)
		height2 = uint64(8)
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

		v2, err := challenge.AddLeaf(
			a2,
			util.HistoryCommitment{
				Height:    height2,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)

		v1Commit := common.BytesToHash([]byte("nyan"))
		v2Commit := common.BytesToHash([]byte("nyan2"))
		v2Height4, err := v2.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    v2Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), v2Height4.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height4.inner.HistoryRoot[:])

		v1Commit = common.BytesToHash([]byte("nyan"))
		v1Height4, err := v1.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    v1Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), v1Height4.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height4.inner.HistoryRoot[:])

		v2Height2, err := v2Height4.Bisect(
			util.HistoryCommitment{
				Height:    2,
				Merkle:    v2Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), v2Height2.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height2.inner.HistoryRoot[:])

		v1Height2, err := v1Height4.Bisect(
			util.HistoryCommitment{
				Height:    2,
				Merkle:    v1Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), v1Height2.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height2.inner.HistoryRoot[:])

		v1Height1, err := v1Height2.Bisect(
			util.HistoryCommitment{
				Height:    1,
				Merkle:    v1Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), v1Height1.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height1.inner.HistoryRoot[:])

		require.ErrorContains(t, v1Height1.CreateSubChallenge(context.Background()), "execution reverted: Has presumptive successor")
	})
	t.Run("Can create succession challenge", func(t *testing.T) {
		chain, _ = setupAssertionChainWithChallengeManager(t)
		height1 = uint64(8)
		height2 = uint64(8)
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

		v2, err := challenge.AddLeaf(
			a2,
			util.HistoryCommitment{
				Height:    height2,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)

		v1Commit := common.BytesToHash([]byte("nyan"))
		v2Commit := common.BytesToHash([]byte("nyan2"))
		v2Height4, err := v2.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    v2Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), v2Height4.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height4.inner.HistoryRoot[:])

		v1Height4, err := v1.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    v1Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(4), v1Height4.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height4.inner.HistoryRoot[:])

		v2Height2, err := v2Height4.Bisect(
			util.HistoryCommitment{
				Height:    2,
				Merkle:    v2Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), v2Height2.inner.Height.Uint64())
		require.Equal(t, v2Commit[:], v2Height2.inner.HistoryRoot[:])

		v1Height2, err := v1Height4.Bisect(
			util.HistoryCommitment{
				Height:    2,
				Merkle:    v1Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), v1Height2.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height2.inner.HistoryRoot[:])

		v1Height1, err := v1Height2.Bisect(
			util.HistoryCommitment{
				Height:    1,
				Merkle:    v1Commit,
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), v1Height1.inner.Height.Uint64())
		require.Equal(t, v1Commit[:], v1Height1.inner.HistoryRoot[:])

		v2Height1, err := v2Height2.Bisect(
			util.HistoryCommitment{
				Height:    1,
				Merkle:    v2Commit,
				FirstLeaf: genesis.inner.StateHash,
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
