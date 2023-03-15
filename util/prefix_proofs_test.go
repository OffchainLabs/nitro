package util_test

import (
	"context"
	"testing"

	"fmt"

	"crypto/ecdsa"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/stretchr/testify/require"
	"math/big"
)

func TestGoPrefixProofs(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	manager := statemanager.New(hashes)

	loCommit, err := manager.HistoryCommitmentUpTo(ctx, 3)
	require.NoError(t, err)
	hiCommit, err := manager.HistoryCommitmentUpTo(ctx, 7)
	require.NoError(t, err)
	packedProof, err := manager.PrefixProof(ctx, 3, 7)
	require.NoError(t, err)

	data, err := statemanager.ProofArgs.Unpack(packedProof)
	require.NoError(t, err)
	preExpansion := data[0].([][32]byte)
	proof := data[1].([][32]byte)

	preExpansionHashes := make([]common.Hash, len(preExpansion))
	for i := 0; i < len(preExpansion); i++ {
		preExpansionHashes[i] = preExpansion[i]
	}
	prefixProof := make([]common.Hash, len(proof))
	for i := 0; i < len(proof); i++ {
		prefixProof[i] = proof[i]
	}

	err = util.VerifyPrefixProofGo(&util.VerifyPrefixProofConfig{
		PreRoot:      loCommit.Merkle,
		PreSize:      4,
		PostRoot:     hiCommit.Merkle,
		PostSize:     8,
		PreExpansion: preExpansionHashes,
		PrefixProof:  prefixProof,
	})
	require.NoError(t, err)
}

func setupMerkleTreeContract(t testing.TB) (*testAccount, *backends.SimulatedBackend) {
	t.Helper()
	ctx := context.Background()
	numChains := uint64(1)
	accs, backend := setupAccounts(t, numChains)
	headerReader := headerreader.New(util.SimulatedBackendWrapper{SimulatedBackend: backend}, func() *headerreader.Config { return &headerreader.TestConfig })
	headerReader.Start(ctx)
	return accs[0], backend
}

// Represents a test EOA account in the simulated backend,
type testAccount struct {
	accountAddr common.Address
	txOpts      *bind.TransactOpts
}

func setupAccounts(t testing.TB, numAccounts uint64) ([]*testAccount, *backends.SimulatedBackend) {
	t.Helper()
	genesis := make(core.GenesisAlloc)
	gasLimit := uint64(100000000)

	accs := make([]*testAccount, numAccounts)
	for i := uint64(0); i < numAccounts; i++ {
		privKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
		require.Equal(t, true, ok)

		// Strip off the 0x and the first 2 characters 04 which is always the
		// EC prefix and is not required.
		publicKeyBytes := crypto.FromECDSAPub(pubKeyECDSA)[4:]
		var pubKey = make([]byte, 48)
		copy(pubKey, publicKeyBytes)

		addr := crypto.PubkeyToAddress(privKey.PublicKey)
		chainID := big.NewInt(1337)
		txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
		require.NoError(t, err)
		startingBalance, _ := new(big.Int).SetString(
			"100000000000000000000000000000000000000",
			10,
		)
		genesis[addr] = core.GenesisAccount{Balance: startingBalance}
		accs[i] = &testAccount{
			accountAddr: addr,
			txOpts:      txOpts,
		}
	}
	backend := backends.NewSimulatedBackend(genesis, gasLimit)
	return accs, backend
}
