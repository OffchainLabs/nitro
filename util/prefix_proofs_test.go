package util_test

import (
	"context"
	"testing"

	"fmt"

	"crypto/ecdsa"
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/mocksgen"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/stretchr/testify/require"
)

func TestLeastSignificantBit_GoSolidityEquivalence(t *testing.T) {
	merkleTreeContract, _ := setupMerkleTreeContract(t)
	runBitEquivalenceTest(t, merkleTreeContract.LeastSignificantBit, util.LeastSignificantBit)
}

func TestMostSignificantBit_GoSolidityEquivalence(t *testing.T) {
	merkleTreeContract, _ := setupMerkleTreeContract(t)
	runBitEquivalenceTest(t, merkleTreeContract.MostSignificantBit, util.MostSignificantBit)
}

func FuzzBitUtils_GoSolidityEquivalence(f *testing.F) {
	testcases := []uint64{
		0,
		2,
		3,
		4,
		7,
		8,
		100,
		1 << 32,
		1<<32 - 1,
		1<<32 + 1,
		1 << 40,
	}
	for _, tc := range testcases {
		f.Add(tc)
	}
	merkleTreeContract, _ := setupMerkleTreeContract(f)
	opts := &bind.CallOpts{}
	f.Fuzz(func(t *testing.T, x uint64) {
		lsbSol, _ := merkleTreeContract.LeastSignificantBit(opts, big.NewInt(int64(x)))
		lsbGo, _ := util.LeastSignificantBit(x)
		if lsbSol != nil {
			if !lsbSol.IsUint64() {
				t.Fatal("lsb sol not a uint64")
			}
			if lsbSol.Uint64() != lsbGo {
				t.Errorf("Mismatch lsb sol=%d, go=%d", lsbSol, lsbGo)
			}
		}
		msbSol, _ := merkleTreeContract.MostSignificantBit(opts, big.NewInt(int64(x)))
		msbGo, _ := util.MostSignificantBit(x)
		if msbSol != nil {
			if !msbSol.IsUint64() {
				t.Fatal("msb sol not a uint64")
			}
			if msbSol.Uint64() != msbGo {
				t.Errorf("Mismatch msb sol=%d, go=%d", msbSol, msbGo)
			}
		}
	})
}

func runBitEquivalenceTest(
	t testing.TB,
	solFunc func(opts *bind.CallOpts, x *big.Int) (*big.Int, error),
	goFunc func(x uint64) (uint64, error),
) {
	opts := &bind.CallOpts{}
	for _, tt := range []struct {
		num        uint64
		wantSolErr bool
		solErr     string
		wantGoErr  bool
		goErr      error
	}{
		{
			num:        0,
			wantSolErr: true,
			solErr:     "has no significant bits",
			wantGoErr:  true,
			goErr:      util.ErrCannotBeZero,
		},
		{num: 2},
		{num: 3},
		{num: 4},
		{num: 7},
		{num: 8},
		{num: 10},
		{num: 100},
		{num: 256},
		{num: 1 << 32},
		{num: 1<<32 + 1},
		{num: 1<<32 - 1},
		{num: 10231920391293},
	} {
		lsbSol, err := solFunc(opts, big.NewInt(int64(tt.num)))
		if tt.wantSolErr {
			require.NotNil(t, err)
			require.ErrorContains(t, err, tt.solErr)
		} else {
			require.NoError(t, err)
		}

		lsbGo, err := goFunc(tt.num)
		if tt.wantGoErr {
			require.NotNil(t, err)
			require.ErrorIs(t, err, tt.goErr)
		} else {
			require.NoError(t, err)
			require.Equal(t, lsbSol.Uint64(), lsbGo)
		}
	}

}

func TestVerifyPrefixProof_GoSolidityEquivalence(t *testing.T) {
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

	merkleTreeContract, _ := setupMerkleTreeContract(t)
	err = merkleTreeContract.VerifyPrefixProof(
		&bind.CallOpts{},
		loCommit.Merkle,
		big.NewInt(4),
		hiCommit.Merkle,
		big.NewInt(8),
		preExpansion,
		proof,
	)
	require.NoError(t, err)

	// err = util.VerifyPrefixProofGo(&util.VerifyPrefixProofConfig{
	// 	PreRoot:      loCommit.Merkle,
	// 	PreSize:      4,
	// 	PostRoot:     hiCommit.Merkle,
	// 	PostSize:     8,
	// 	PreExpansion: preExpansionHashes,
	// 	PrefixProof:  prefixProof,
	// })
	// require.NoError(t, err)
}

func setupMerkleTreeContract(t testing.TB) (*mocksgen.MerkleTreeAccess, *backends.SimulatedBackend) {
	t.Helper()
	ctx := context.Background()
	numChains := uint64(1)
	accs, backend := setupAccounts(t, numChains)
	headerReader := headerreader.New(util.SimulatedBackendWrapper{SimulatedBackend: backend}, func() *headerreader.Config { return &headerreader.TestConfig })
	headerReader.Start(ctx)
	_, _, merkleTreeContract, err := mocksgen.DeployMerkleTreeAccess(accs[0].txOpts, backend)
	require.NoError(t, err)
	backend.Commit()
	return merkleTreeContract, backend
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
