package solimpl

import (
	"context"
	"math/big"
	"testing"
	"time"

	//"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCreateAssertion(t *testing.T) {
	chain, _ := setupAssertionChainWithChallengeManager(t)

	commit := util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{1}),
	}
	t.Run("OK", func(t *testing.T) {
		created, err2 := chain.CreateAssertion(commit, 0)
		require.NoError(t, err2)
		require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
	})
	// t.Run("already exists", func(t *testing.T) {
	// 	_, err = chain.CreateAssertion(commit, 0)
	// 	require.ErrorIs(t, err, ErrAlreadyExists)
	// })
	// t.Run("previous assertion does not exist", func(t *testing.T) {
	// 	commit := util.StateCommitment{
	// 		Height:    2,
	// 		StateRoot: common.BytesToHash([]byte{2}),
	// 	}
	// 	_, err = chain.CreateAssertion(commit, 1)
	// 	require.ErrorIs(t, err, ErrPrevDoesNotExist)
	// })
	// t.Run("invalid height", func(t *testing.T) {
	// 	commit := util.StateCommitment{
	// 		Height:    0,
	// 		StateRoot: common.BytesToHash([]byte{3}),
	// 	}
	// 	_, err = chain.CreateAssertion(commit, 0)
	// 	require.ErrorIs(t, err, ErrInvalidHeight)
	// })
	// t.Run("too late to create sibling", func(t *testing.T) {
	// 	Adds two challenge periods to the chain timestamp.
	// 	err = acc.backend.AdjustTime(time.Second * 20)
	// 	require.NoError(t, err)
	// 	commit := util.StateCommitment{
	// 		Height:    1,
	// 		StateRoot: common.BytesToHash([]byte("forked")),
	// 	}
	// 	_, err = chain.CreateAssertion(commit, 0)
	// 	require.ErrorIs(t, err, ErrTooLate)
	// })
}

func TestAssertionByID(t *testing.T) {
	chain, _ := setupAssertionChainWithChallengeManager(t)

	resp, err := chain.AssertionByID(0)
	require.NoError(t, err)

	require.Equal(t, true, resp.inner.StateHash != [32]byte{})

	_, err = chain.AssertionByID(1)
	require.ErrorIs(t, err, ErrNotFound)
}

// func TestAssertion_Confirm(t *testing.T) {
// 	ctx := context.Background()
// 	acc, err := setupAccount()
// 	require.NoError(t, err)

// 	genesisStateRoot := common.BytesToHash([]byte("foo"))
// 	addr, _, _, err := challengeV2gen.DeployAssertionChain(
// 		acc.txOpts,
// 		acc.backend,
// 		genesisStateRoot,
// 		big.NewInt(10), // 10 second challenge period.
// 	)
// 	require.NoError(t, err)
// 	acc.backend.Commit()

// 	chain, err := NewAssertionChain(
// 		ctx, addr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
// 	)
// 	require.NoError(t, err)

// 	commit := util.StateCommitment{
// 		Height:    1,
// 		StateRoot: common.BytesToHash([]byte{1}),
// 	}
// 	genesisId := common.Hash{}

// 	created, err := chain.CreateAssertion(commit, genesisId)
// 	require.NoError(t, err)
// 	require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
// 	acc.backend.Commit()

// 	t.Run("Can confirm assertion", func(t *testing.T) {
// 		require.Equal(t, uint8(0), created.inner.Status) // Pending.
// 		require.NoError(t, created.Confirm())
// 		acc.backend.Commit()
// 		created, err = chain.AssertionByID(created.id)
// 		require.NoError(t, err)
// 		require.Equal(t, uint8(1), created.inner.Status) // Confirmed.
// 	})

// 	t.Run("Unknown assertion", func(t *testing.T) {
// 		created.id = common.BytesToHash([]byte("meow"))
// 		require.ErrorIs(t, created.Confirm(), ErrNotFound)
// 	})
// }

func TestAssertion_Reject(t *testing.T) {
	acc, err := setupAccount()
	require.NoError(t, err)

	chain, _ := setupAssertionChainWithChallengeManager(t)
	commit := util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{1}),
	}
	created, err := chain.CreateAssertion(commit, 0)
	require.NoError(t, err)
	require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
	acc.backend.Commit()

	commit = util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{2}),
	}
	created, err = chain.CreateAssertion(commit, 0)
	require.NoError(t, err)
	require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
	acc.backend.Commit()

	_, err = chain.CreateSuccessionChallenge(0)
	require.NoError(t, err)

	// t.Run("Can reject assertion", func(t *testing.T) {
	// 	t.Skip("TODO: Can't reject assertion. Blocked by one step proof")
	// 	require.Equal(t, uint8(0), created.inner.Status) // Pending.
	// 	require.NoError(t, created.Reject())
	// 	acc.backend.Commit()
	// 	created, err = chain.AssertionByID(created.id)
	// 	require.NoError(t, err)
	// 	require.Equal(t, uint8(2), created.inner.Status) // Confirmed.
	// })

	t.Run("Unknown assertion", func(t *testing.T) {
		created.id = 1
		require.ErrorIs(t, created.Reject(), ErrNotFound)
	})

	t.Run("Already confirmed assertion", func(t *testing.T) {
		ga, err := chain.AssertionByID(0)
		require.NoError(t, err)
		require.ErrorIs(t, ga.Reject(), ErrNonPendingAssertion)
	})
}

func TestChallengePeriodSeconds(t *testing.T) {
	chain, _ := setupAssertionChainWithChallengeManager(t)
	chalPeriod, err := chain.ChallengePeriodSeconds()
	require.NoError(t, err)
	require.Equal(t, time.Second, chalPeriod)
}

func TestCreateSuccessionChallenge(t *testing.T) {
	t.Run("assertion does not exist", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		_, err := chain.CreateSuccessionChallenge(2)
		require.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("assertion already rejected", func(t *testing.T) {
		t.Skip(
			"Needs a challenge manager to provide a winning claim first",
		)
	})
	t.Run("at least two children required", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		_, err := chain.CreateSuccessionChallenge(0)
		require.ErrorIs(t, err, ErrInvalidChildren)

		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		_, err = chain.CreateAssertion(commit1, 0)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(0)
		require.ErrorIs(t, err, ErrInvalidChildren)
	})

	t.Run("too late to challenge", func(t *testing.T) {
		chain, acc := setupAssertionChainWithChallengeManager(t)
		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		_, err := chain.CreateAssertion(commit1, 0)
		require.NoError(t, err)

		commit2 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{2}),
		}

		_, err = chain.CreateAssertion(commit2, 0)
		require.NoError(t, err)

		challengePeriod, err := chain.ChallengePeriodSeconds()
		require.NoError(t, err)

		// Adds two challenge periods to the chain timestamp.
		err = acc.backend.AdjustTime(challengePeriod * 2)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(0)
		require.ErrorIs(t, err, ErrTooLate)
	})
	t.Run("OK", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		_, err := chain.CreateAssertion(commit1, 0)
		require.NoError(t, err)

		commit2 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{2}),
		}

		_, err = chain.CreateAssertion(commit2, 0)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(0)
		require.NoError(t, err)
	})
	t.Run("challenge already exists", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		_, err := chain.CreateAssertion(commit1, 0)
		require.NoError(t, err)

		commit2 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{2}),
		}

		_, err = chain.CreateAssertion(commit2, 0)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(0)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(0)
		require.ErrorIs(t, err, ErrAlreadyExists)
	})
}

func setupAssertionChainWithChallengeManager(t *testing.T) (*AssertionChain, *testAccount) {
	t.Helper()
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)
	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := acc.accountAddr
	chainId := big.NewInt(1337)
	loserStakeEscrow := common.Address{}
	cfg := generateRollupConfig(prod, wasmModuleRoot, rollupOwner, chainId, loserStakeEscrow)
	numValidators := uint64(10)
	addresses := deployFullRollupStack(
		t,
		ctx,
		acc.backend,
		acc.txOpts,
		common.Address{},
		numValidators,
		cfg,
	)
	chain, err := NewAssertionChain(
		ctx,
		addresses.Rollup,
		addresses.RollupUserLogic,
		acc.txOpts,
		&bind.CallOpts{},
		acc.accountAddr,
		acc.backend,
	)
	require.NoError(t, err)
	acc.backend.Commit()

	return chain, acc
}
