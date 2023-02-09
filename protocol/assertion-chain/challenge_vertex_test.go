package assertionchain

import (
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestChallengeVertex_Bisect(t *testing.T) {
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
	require.NoError(t, err, "FAILS HERE")

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
		// We make a challenge period pass.
		_, err = v2.Bisect(
			util.HistoryCommitment{
				Height:    4,
				Merkle:    common.BytesToHash([]byte("nyan2")),
				FirstLeaf: genesis.inner.StateHash,
			},
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "presumptive successor already confirmable")
	})
	t.Run("invalid prefix history", func(t *testing.T) {
		t.Skip("Need to add proof capabilities in solidity in order to test")
	})
	t.Run("bisection vertex already exists", func(t *testing.T) {
	})
}
