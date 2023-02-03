package assertionchain

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"context"
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
	addr, _, _, err := outgen.DeployAssertionChain(acc.txOpts, acc.backend)
	require.NoError(t, err)

	acc.backend.Commit()

	chain, err := NewAssertionChain(
		ctx, addr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
	)
	require.NoError(t, err)

	genesisStateRoot := common.BytesToHash([]byte("foo"))
	_, err = chain.writer.SetupGenesis(acc.txOpts, genesisStateRoot)
	require.NoError(t, err)

	acc.backend.Commit()

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
		created, err := chain.AssertionByID(id)
		require.NoError(t, err)
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
}

func TestAssertionByID(t *testing.T) {
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)
	addr, _, _, err := outgen.DeployAssertionChain(acc.txOpts, acc.backend)
	require.NoError(t, err)

	acc.backend.Commit()

	chain, err := NewAssertionChain(
		ctx, addr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
	)
	require.NoError(t, err)

	genesisStateRoot := common.BytesToHash([]byte("foo"))
	_, err = chain.writer.SetupGenesis(acc.txOpts, genesisStateRoot)
	require.NoError(t, err)

	acc.backend.Commit()

	genesisId := common.Hash{}
	resp, err := chain.AssertionByID(genesisId)
	require.NoError(t, err)

	require.Equal(t, genesisStateRoot[:], resp.inner.StateHash[:])

	_, err = chain.AssertionByID(common.BytesToHash([]byte("bar")))
	require.ErrorIs(t, err, ErrNotFound)
}

func TestChallengePeriodLength(t *testing.T) {
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)
	addr, _, _, err := outgen.DeployAssertionChain(acc.txOpts, acc.backend)
	require.NoError(t, err)

	acc.backend.Commit()

	chain, err := NewAssertionChain(
		ctx, addr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
	)
	require.NoError(t, err)
	chalPeriod, err := chain.ChallengePeriodLength()
	require.NoError(t, err)
	require.Equal(t, time.Second*1000, chalPeriod)
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
	gasLimit := uint64(2100000000000)
	backend := backends.NewSimulatedBackend(genesis, gasLimit)
	return &testAccount{
		accountAddr: addr,
		backend:     backend,
		txOpts:      txOpts,
	}, nil
}
