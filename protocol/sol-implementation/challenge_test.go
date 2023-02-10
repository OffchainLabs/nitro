package solimpl

import (
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestChallenge_BlockChallenge_AddLeaf(t *testing.T) {
	chain, _ := setupAssertionChainWithChallengeManager(t)
	height1 := uint64(1)
	height2 := uint64(1)
	a1, _, challenge := setupTopLevelFork(t, chain, height1, height2)

	t.Run("claim predecessor not linked to challenge", func(t *testing.T) {
		// Pass in a junk assertion that has no predecessor.
		_, err := challenge.AddLeaf(
			&Assertion{
				chain: chain,
				id:    common.BytesToHash([]byte("junk")),
				StateCommitment: util.StateCommitment{
					Height:    height1,
					StateRoot: common.BytesToHash([]byte("foo")),
				},
				inner: outgen.Assertion{
					PredecessorId: common.BytesToHash([]byte("junk")),
				},
			},
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("bar")),
			},
		)
		require.ErrorContains(t, err, "Assertion does not exist")
	})
	t.Run("invalid height", func(t *testing.T) {
		_, err := challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height: 100,
				Merkle: common.BytesToHash([]byte("bar")),
			},
		)
		require.ErrorContains(t, err, "Invalid height")
	})
	t.Run("last state is not assertion claim block hash", func(t *testing.T) {
		t.Skip("Needs proofs implemented in solidity")
	})
	t.Run("empty history commitment", func(t *testing.T) {
		_, err := challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.Hash{},
			},
		)
		require.ErrorContains(t, err, "Empty historyCommitment")
	})
	t.Run("winner already declared", func(t *testing.T) {
		t.Skip("Needs winner declaration logic implemented in solidity")
	})
	t.Run("last state not in history", func(t *testing.T) {
		t.Skip()
	})
	t.Run("first state not in history", func(t *testing.T) {
		t.Skip()
	})
	t.Run("first state is not the challenge root", func(t *testing.T) {
		_, err := challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.ErrorContains(t, err, "First state is not the challenge root")
	})
	t.Run("OK", func(t *testing.T) {
		genesis, err := chain.AssertionByID(common.Hash{})
		require.NoError(t, err)
		_, err = challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.NoError(t, err)
	})
	t.Run("already exists", func(t *testing.T) {
		genesis, err := chain.AssertionByID(common.Hash{})
		require.NoError(t, err)
		_, err = challenge.AddLeaf(
			a1,
			util.HistoryCommitment{
				Height:    height1,
				Merkle:    common.BytesToHash([]byte("nyan")),
				FirstLeaf: genesis.inner.StateHash,
			},
		)
		require.ErrorContains(t, err, "already exists")
	})
}

func setupTopLevelFork(
	t *testing.T,
	chain *AssertionChain,
	height1,
	height2 uint64,
) (*Assertion, *Assertion, *Challenge) {
	t.Helper()
	genesisId := common.Hash{}

	// Creates a simple assertion chain fork.
	commit1 := util.StateCommitment{
		Height:    height1,
		StateRoot: common.BytesToHash([]byte{1}),
	}
	a1, err := chain.CreateAssertion(commit1, genesisId)
	require.NoError(t, err)

	commit2 := util.StateCommitment{
		Height:    height2,
		StateRoot: common.BytesToHash([]byte{2}),
	}
	a2, err := chain.CreateAssertion(commit2, genesisId)
	require.NoError(t, err)

	// Initiates a challenge on the genesis assertion.
	challenge, err := chain.CreateSuccessionChallenge(genesisId)
	require.NoError(t, err)
	return a1, a2, challenge
}
