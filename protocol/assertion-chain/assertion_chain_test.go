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
		err = chain.createAssertion(commit, genesisId)
		require.NoError(t, err)

		acc.backend.Commit()

		id := getAssertionId(commit, genesisId)
		created, err2 := chain.AssertionByID(id)
		require.NoError(t, err2)
		require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
	})
	t.Run("already exists", func(t *testing.T) {
		err = chain.createAssertion(commit, genesisId)
		require.ErrorIs(t, err, ErrAlreadyExists)
	})
	t.Run("previous assertion does not exist", func(t *testing.T) {
		commit := util.StateCommitment{
			Height:    2,
			StateRoot: common.BytesToHash([]byte{2}),
		}
		err = chain.createAssertion(commit, common.BytesToHash([]byte("nyan")))
		require.ErrorIs(t, err, ErrPrevDoesNotExist)
	})
	t.Run("invalid height", func(t *testing.T) {
		commit := util.StateCommitment{
			Height:    0,
			StateRoot: common.BytesToHash([]byte{3}),
		}
		err = chain.createAssertion(commit, genesisId)
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
		err = chain.createAssertion(commit, genesisId)
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
		err := chain.CreateSuccessionChallenge([32]byte{9})
		require.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("assertion already rejected", func(t *testing.T) {
		t.Skip(
			"Needs a challenge manager to provide a winning claim first",
		)
	})
	t.Run("at least two children required", func(t *testing.T) {
		chain, acc := setupAssertionChainWithChallengeManager(t)
		err := chain.CreateSuccessionChallenge(genesisId)
		require.ErrorIs(t, err, ErrInvalidChildren)

		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		err = chain.createAssertion(commit1, genesisId)
		require.NoError(t, err)
		acc.backend.Commit()

		err = chain.CreateSuccessionChallenge(genesisId)
		require.ErrorIs(t, err, ErrInvalidChildren)
	})

	t.Run("too late to challenge", func(t *testing.T) {
		chain, acc := setupAssertionChainWithChallengeManager(t)
		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		err := chain.createAssertion(commit1, genesisId)
		require.NoError(t, err)
		acc.backend.Commit()

		commit2 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{2}),
		}

		err = chain.createAssertion(commit2, genesisId)
		require.NoError(t, err)
		acc.backend.Commit()

		challengePeriod, err := chain.ChallengePeriodSeconds()
		require.NoError(t, err)

		// Adds two challenge periods to the chain timestamp.
		err = acc.backend.AdjustTime(challengePeriod * 2)
		require.NoError(t, err)

		err = chain.CreateSuccessionChallenge(genesisId)
		require.ErrorIs(t, err, ErrTooLate)
	})
	t.Run("OK", func(t *testing.T) {
		t.Skip("Deploy chal manager")
		chain, acc := setupAssertionChainWithChallengeManager(t)
		commit1 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{1}),
		}

		err := chain.createAssertion(commit1, genesisId)
		require.NoError(t, err)
		acc.backend.Commit()

		commit2 := util.StateCommitment{
			Height:    1,
			StateRoot: common.BytesToHash([]byte{2}),
		}

		err = chain.createAssertion(commit2, genesisId)
		require.NoError(t, err)
		acc.backend.Commit()

		err = chain.CreateSuccessionChallenge(genesisId)
		require.NoError(t, err)
		acc.backend.Commit()
	})
	t.Run("challenge already exists", func(t *testing.T) {
		t.Skip("Create a fork and successful challenge first")
	})
}

func TestChalManager(t *testing.T) {
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)
	acc.txOpts.GasLimit = acc.backend.Blockchain().GasLimit()

	// VERTEX MANAGER.
	vertexManagerAddr, tx, _, err := outgen.DeployVertexManager(acc.txOpts, acc.backend)
	require.NoError(t, err)
	acc.backend.Commit()

	receipt, err := acc.backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, true, receipt.Status == 1, "Receipt says tx failed")

	code, err := acc.backend.CodeAt(ctx, vertexManagerAddr, nil)
	require.NoError(t, err)
	t.Logf("Vertex manager code size = %d", len(code))
	require.Equal(t, true, len(code) > 0)

	// CHALLENGE LEAF ADDERS.
	blockLeafAdderAddr, tx, _, err := outgen.DeployBlockLeafAdder(acc.txOpts, acc.backend)
	require.NoError(t, err)
	acc.backend.Commit()

	receipt, err = acc.backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, true, receipt.Status == 1, "Receipt says tx failed")

	code, err = acc.backend.CodeAt(ctx, blockLeafAdderAddr, nil)
	require.NoError(t, err)
	t.Logf("BlockChallengeLeafAdder code size = %d", len(code))
	require.Equal(t, true, len(code) > 0)

	bigStepLeafAdderAddr, tx, _, err := outgen.DeployBigStepLeafAdder(acc.txOpts, acc.backend)
	require.NoError(t, err)
	acc.backend.Commit()

	receipt, err = acc.backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, true, receipt.Status == 1, "Receipt says tx failed")

	code, err = acc.backend.CodeAt(ctx, bigStepLeafAdderAddr, nil)
	require.NoError(t, err)
	t.Logf("BigStepChallengeLeafAdder code size = %d", len(code))
	require.Equal(t, true, len(code) > 0)

	smallStepLeafAdderAddr, tx, _, err := outgen.DeploySmallStepLeafAdder(acc.txOpts, acc.backend)
	require.NoError(t, err)
	acc.backend.Commit()

	receipt, err = acc.backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, true, receipt.Status == 1, "Receipt says tx failed")

	code, err = acc.backend.CodeAt(ctx, smallStepLeafAdderAddr, nil)
	require.NoError(t, err)
	t.Logf("SmallStepChallengeLeafAdder code size = %d", len(code))
	require.Equal(t, true, len(code) > 0)

	// ASSERTION CHAIN.
	genesisStateRoot := common.BytesToHash([]byte("foo"))
	challengePeriod := uint64(30)
	assertionChainAddr, tx, _, err := outgen.DeployAssertionChain(
		acc.txOpts,
		acc.backend,
		genesisStateRoot,
		big.NewInt(int64(challengePeriod)),
	)
	require.NoError(t, err)
	acc.backend.Commit()

	receipt, err = acc.backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, true, receipt.Status == 1, "Receipt says tx failed")

	code, err = acc.backend.CodeAt(ctx, assertionChainAddr, nil)
	require.NoError(t, err)
	t.Logf("Assertion chain code size = %d", len(code))
	require.Equal(t, true, len(code) > 0)

	// CHALLENGE MANAGER.
	miniStakeValue := big.NewInt(1)
	chalManagerAddr, tx, _, err := outgen.DeployChallengeManager(
		acc.txOpts,
		acc.backend,
		assertionChainAddr,
		vertexManagerAddr,
		miniStakeValue,
		big.NewInt(int64(challengePeriod)),
	)
	require.NoError(t, err)
	acc.backend.Commit()

	receipt, err = acc.backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, true, receipt.Status == 1, "Receipt says tx failed")

	// Chain contract should be deployed.
	code, err = acc.backend.CodeAt(ctx, chalManagerAddr, nil)
	require.NoError(t, err)
	t.Logf("Challenge manager code size = %d", len(code))
	require.Equal(t, true, len(code) > 0)
}

func setupAssertionChainWithChallengeManager(t *testing.T) (*AssertionChain, *testAccount) {
	t.Helper()
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)

	genesisStateRoot := common.BytesToHash([]byte("foo"))
	challengePeriod := uint64(30)
	assertionChainAddr, _, _, err := outgen.DeployAssertionChain(
		acc.txOpts,
		acc.backend,
		genesisStateRoot,
		big.NewInt(int64(challengePeriod)),
	)
	require.NoError(t, err)
	acc.backend.Commit()

	// Chain contract should be deployed.
	code, err := acc.backend.CodeAt(ctx, assertionChainAddr, nil)
	require.NoError(t, err)
	require.Equal(t, true, len(code) > 0)

	chain, err := NewAssertionChain(
		ctx, assertionChainAddr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
	)
	require.NoError(t, err)

	// miniStakeValue := big.NewInt(1)
	// chalManagerAddr, _, _, err := outgen.DeployChallengeManager(
	// 	acc.txOpts,
	// 	acc.backend,
	// 	assertionChainAddr,
	// 	miniStakeValue,
	// 	challengePeriodSeconds,
	// 	common.Address{}, // OSP entry contract.
	// )
	// require.NoError(t, err)
	// acc.backend.Commit()
	// _ = chalManagerAddr

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
