package assertionchain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestCreateAssertion(t *testing.T) {
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)

	genesisStateRoot := common.BytesToHash([]byte("foo"))
	addr, _, _, err := outgen.DeployAssertionChain(
		acc.txOpts,
		acc.backend,
		genesisStateRoot,
		big.NewInt(10), // 10 second challenge period.
	)
	require.NoError(t, err)

	acc.backend.Commit()

	chain, err := NewAssertionChain(
		ctx, addr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
	)
	require.NoError(t, err)

	commit := util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{1}),
	}
	genesisId := common.Hash{}

	t.Run("OK", func(t *testing.T) {
		created, err2 := chain.CreateAssertion(commit, genesisId)
		require.NoError(t, err2)
		require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
	})
	t.Run("already exists", func(t *testing.T) {
		_, err = chain.CreateAssertion(commit, genesisId)
		require.ErrorIs(t, err, ErrAlreadyExists)
	})
	t.Run("previous assertion does not exist", func(t *testing.T) {
		commit := util.StateCommitment{
			Height:    2,
			StateRoot: common.BytesToHash([]byte{2}),
		}
		_, err = chain.CreateAssertion(commit, common.BytesToHash([]byte("nyan")))
		require.ErrorIs(t, err, ErrPrevDoesNotExist)
	})
	t.Run("invalid height", func(t *testing.T) {
		commit := util.StateCommitment{
			Height:    0,
			StateRoot: common.BytesToHash([]byte{3}),
		}
		_, err = chain.CreateAssertion(commit, genesisId)
		require.ErrorIs(t, err, ErrInvalidHeight)
	})
	t.Run("too late to create sibling", func(t *testing.T) {
		// Adds two challenge periods to the chain timestamp.
		err = acc.backend.AdjustTime(time.Second * 20)
		require.NoError(t, err)
		commit := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte("forked")),
		}
		_, err = chain.CreateAssertion(commit, genesisId)
		require.ErrorIs(t, err, ErrTooLate)
	})
}

func TestAssertionByID(t *testing.T) {
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)
	genesisStateRoot := common.BytesToHash([]byte("foo"))
	addr, _, _, err := outgen.DeployAssertionChain(
		acc.txOpts,
		acc.backend,
		genesisStateRoot,
		big.NewInt(1), // 1 second challenge period.
	)
	require.NoError(t, err)

	acc.backend.Commit()

	chain, err := NewAssertionChain(
		ctx, addr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
	)
	require.NoError(t, err)

	genesisId := common.Hash{}
	resp, err := chain.AssertionByID(genesisId)
	require.NoError(t, err)

	require.Equal(t, genesisStateRoot[:], resp.inner.StateHash[:])

	_, err = chain.AssertionByID(common.BytesToHash([]byte("bar")))
	require.ErrorIs(t, err, ErrNotFound)
}

func TestAssertion_Confirm(t *testing.T) {
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)

	genesisStateRoot := common.BytesToHash([]byte("foo"))
	addr, _, _, err := outgen.DeployAssertionChain(
		acc.txOpts,
		acc.backend,
		genesisStateRoot,
		big.NewInt(10), // 10 second challenge period.
	)
	require.NoError(t, err)
	acc.backend.Commit()

	chain, err := NewAssertionChain(
		ctx, addr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
	)
	require.NoError(t, err)

	commit := util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{1}),
	}
	genesisId := common.Hash{}

	created, err := chain.CreateAssertion(commit, genesisId)
	require.NoError(t, err)
	require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
	acc.backend.Commit()

	t.Run("Can confirm assertion", func(t *testing.T) {
		require.Equal(t, uint8(0), created.inner.Status) // Pending.
		require.NoError(t, created.Confirm())
		acc.backend.Commit()
		created, err = chain.AssertionByID(created.id)
		require.NoError(t, err)
		require.Equal(t, uint8(1), created.inner.Status) // Confirmed.
	})

	t.Run("Unknown assertion", func(t *testing.T) {
		created.id = common.BytesToHash([]byte("meow"))
		require.ErrorIs(t, created.Confirm(), ErrNotFound)
	})
}

func TestAssertion_Reject(t *testing.T) {
	acc, err := setupAccount()
	require.NoError(t, err)

	chain, _ := setupAssertionChainWithChallengeManager(t)
	commit := util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{1}),
	}
	genesisId := common.Hash{}
	created, err := chain.CreateAssertion(commit, genesisId)
	require.NoError(t, err)
	require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
	acc.backend.Commit()

	commit = util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{2}),
	}
	created, err = chain.CreateAssertion(commit, genesisId)
	require.NoError(t, err)
	require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
	acc.backend.Commit()

	_, err = chain.CreateSuccessionChallenge(genesisId)
	require.NoError(t, err)

	ga, err := chain.AssertionByID(genesisId)
	require.NoError(t, err)
	t.Log(ga.inner.SuccessionChallenge)

	t.Run("Can reject assertion", func(t *testing.T) {
		require.Equal(t, uint8(0), created.inner.Status) // Pending.
		require.NoError(t, created.Reject())
		acc.backend.Commit()
		created, err = chain.AssertionByID(created.id)
		require.NoError(t, err)
		require.Equal(t, uint8(2), created.inner.Status) // Confirmed.
	})

	t.Run("Unknown assertion", func(t *testing.T) {
		created.id = common.BytesToHash([]byte("meow"))
		require.ErrorIs(t, created.Reject(), ErrNotFound)
	})
}

func TestChallengePeriodSeconds(t *testing.T) {
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)
	genesisStateRoot := common.BytesToHash([]byte("foo"))
	addr, _, _, err := outgen.DeployAssertionChain(
		acc.txOpts,
		acc.backend,
		genesisStateRoot,
		big.NewInt(1), // 1 second challenge period.
	)
	require.NoError(t, err)

	acc.backend.Commit()

	chain, err := NewAssertionChain(
		ctx, addr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
	)
	require.NoError(t, err)
	chalPeriod, err := chain.ChallengePeriodSeconds()
	require.NoError(t, err)
	require.Equal(t, time.Second, chalPeriod)
}

func TestCreateSuccessionChallenge(t *testing.T) {
	genesisId := common.Hash{}

	t.Run("assertion does not exist", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		_, err := chain.CreateSuccessionChallenge([32]byte{9})
		require.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("assertion already rejected", func(t *testing.T) {
		t.Skip(
			"Needs a challenge manager to provide a winning claim first",
		)
	})
	t.Run("at least two children required", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		_, err := chain.CreateSuccessionChallenge(genesisId)
		require.ErrorIs(t, err, ErrInvalidChildren)

		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		_, err = chain.CreateAssertion(commit1, genesisId)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(genesisId)
		require.ErrorIs(t, err, ErrInvalidChildren)
	})

	t.Run("too late to challenge", func(t *testing.T) {
		chain, acc := setupAssertionChainWithChallengeManager(t)
		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		_, err := chain.CreateAssertion(commit1, genesisId)
		require.NoError(t, err)

		commit2 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{2}),
		}

		_, err = chain.CreateAssertion(commit2, genesisId)
		require.NoError(t, err)

		challengePeriod, err := chain.ChallengePeriodSeconds()
		require.NoError(t, err)

		// Adds two challenge periods to the chain timestamp.
		err = acc.backend.AdjustTime(challengePeriod * 2)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(genesisId)
		require.ErrorIs(t, err, ErrTooLate)
	})
	t.Run("OK", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		_, err := chain.CreateAssertion(commit1, genesisId)
		require.NoError(t, err)

		commit2 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{2}),
		}

		_, err = chain.CreateAssertion(commit2, genesisId)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(genesisId)
		require.NoError(t, err)
	})
	t.Run("challenge already exists", func(t *testing.T) {
		chain, _ := setupAssertionChainWithChallengeManager(t)
		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		_, err := chain.CreateAssertion(commit1, genesisId)
		require.NoError(t, err)

		commit2 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{2}),
		}

		_, err = chain.CreateAssertion(commit2, genesisId)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(genesisId)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(genesisId)
		require.ErrorIs(t, err, ErrAlreadyExists)
	})
}

func setupAssertionChainWithChallengeManager(t *testing.T) (*AssertionChain, *testAccount) {
	t.Helper()
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)

	genesisStateRoot := common.BytesToHash([]byte("foo"))
	challengePeriodSeconds := big.NewInt(1000)
	assertionChainAddr, _, _, err := outgen.DeployAssertionChain(
		acc.txOpts,
		acc.backend,
		genesisStateRoot,
		challengePeriodSeconds,
	)
	require.NoError(t, err)
	acc.backend.Commit()

	// Chain contract should be deployed.
	code, err := acc.backend.CodeAt(ctx, assertionChainAddr, nil)
	require.NoError(t, err)
	require.Equal(t, true, len(code) > 0)

	ospAddr, _, _, err := outgen.DeployMockOneStepProofEntry(
		acc.txOpts,
		acc.backend,
	)
	require.NoError(t, err)
	acc.backend.Commit()

	code, err = acc.backend.CodeAt(ctx, ospAddr, nil)
	require.NoError(t, err)
	require.Equal(t, true, len(code) > 0)

	miniStakeValue := big.NewInt(1)
	chalManagerAddr, _, _, err := outgen.DeployChallengeManagerImpl(
		acc.txOpts,
		acc.backend,
		assertionChainAddr,
		miniStakeValue,
		challengePeriodSeconds,
		ospAddr,
	)
	require.NoError(t, err)
	acc.backend.Commit()

	code, err = acc.backend.CodeAt(ctx, chalManagerAddr, nil)
	require.NoError(t, err)
	require.Equal(t, true, len(code) > 0)

	chain, err := NewAssertionChain(
		ctx, assertionChainAddr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
	)
	require.NoError(t, err)
	err = chain.UpdateChallengeManager(chalManagerAddr)
	require.NoError(t, err)
	acc.backend.Commit()

	return chain, acc
}

// Represents a test EOA account in the simulated backend,
type testAccount struct {
	accountAddr common.Address
	backend     *backends.SimulatedBackend
	txOpts      *bind.TransactOpts
}

func setupAccount() (*testAccount, error) {
	genesis := make(core.GenesisAlloc)
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	// Strip off the 0x and the first 2 characters 04 which is always the
	// EC prefix and is not required.
	publicKeyBytes := crypto.FromECDSAPub(pubKeyECDSA)[4:]
	var pubKey = make([]byte, 48)
	copy(pubKey, publicKeyBytes)

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(1337)
	txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		return nil, err
	}
	startingBalance, _ := new(big.Int).SetString(
		"100000000000000000000000000000000000000",
		10,
	)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	gasLimit := uint64(100000000)
	backend := backends.NewSimulatedBackend(genesis, gasLimit)
	return &testAccount{
		accountAddr: addr,
		backend:     backend,
		txOpts:      txOpts,
	}, nil
}
