// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package prefixproofs_test

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	prefixproofs "github.com/OffchainLabs/bold/state-commitments/prefix-proofs"
	statemanager "github.com/OffchainLabs/bold/testing/mocks/state-provider"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestRoot(t *testing.T) {
	t.Run("tree with exactly size MAX_LEVEL should pass validation", func(t *testing.T) {
		tree := make([]common.Hash, prefixproofs.MAX_LEVEL)
		_, err := prefixproofs.Root(tree)
		require.NotEqual(t, prefixproofs.ErrLevelTooHigh, err)
	})
	t.Run("tree too large", func(t *testing.T) {
		tree := make([]common.Hash, prefixproofs.MAX_LEVEL+1)
		_, err := prefixproofs.Root(tree)
		require.Equal(t, prefixproofs.ErrExpansionTooLarge, err)
	})
	t.Run("empty tree", func(t *testing.T) {
		tree := make([]common.Hash, 0)
		_, err := prefixproofs.Root(tree)
		require.Equal(t, prefixproofs.ErrRootForEmpty, err)
	})
	t.Run("single element returns itself", func(t *testing.T) {
		tree := make([]common.Hash, 1)
		tree[0] = common.HexToHash("0x1234")
		root, err := prefixproofs.Root(tree)
		require.NoError(t, err)
		require.Equal(t, tree[0], root)
	})
}

func TestVerifyPrefixProof_GoSolidityEquivalence(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	manager, err := statemanager.NewWithMockedStateRoots(hashes)
	require.NoError(t, err)

	loCommit, err := manager.HistoryCommitmentUpToBatch(ctx, 0, 3, 10)
	require.NoError(t, err)
	hiCommit, err := manager.HistoryCommitmentUpToBatch(ctx, 0, 7, 10)
	require.NoError(t, err)
	packedProof, err := manager.PrefixProofUpToBatch(ctx, 0, 3, 7, 1)
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

	err = prefixproofs.VerifyPrefixProof(&prefixproofs.VerifyPrefixProofConfig{
		PreRoot:      loCommit.Merkle,
		PreSize:      4,
		PostRoot:     hiCommit.Merkle,
		PostSize:     8,
		PreExpansion: preExpansionHashes,
		PrefixProof:  prefixProof,
	})
	require.NoError(t, err)
}

func TestVerifyPrefixProofWithHeight7_GoSolidityEquivalence1(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	manager, err := statemanager.NewWithMockedStateRoots(hashes)
	require.NoError(t, err)

	loCommit, err := manager.HistoryCommitmentUpToBatch(ctx, 0, 3, 10)
	require.NoError(t, err)
	hiCommit, err := manager.HistoryCommitmentUpToBatch(ctx, 0, 6, 10)
	require.NoError(t, err)
	packedProof, err := manager.PrefixProofUpToBatch(ctx, 0, 3, 6, 1)
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
		big.NewInt(7),
		preExpansion,
		proof,
	)
	require.NoError(t, err)

	err = prefixproofs.VerifyPrefixProof(&prefixproofs.VerifyPrefixProofConfig{
		PreRoot:      loCommit.Merkle,
		PreSize:      4,
		PostRoot:     hiCommit.Merkle,
		PostSize:     7,
		PreExpansion: preExpansionHashes,
		PrefixProof:  prefixProof,
	})
	require.NoError(t, err)
}

func TestLeastSignificantBit_GoSolidityEquivalence(t *testing.T) {
	merkleTreeContract, _ := setupMerkleTreeContract(t)
	runBitEquivalenceTest(t, merkleTreeContract.LeastSignificantBit, prefixproofs.LeastSignificantBit)
}

func TestMostSignificantBit_GoSolidityEquivalence(t *testing.T) {
	merkleTreeContract, _ := setupMerkleTreeContract(t)
	runBitEquivalenceTest(t, merkleTreeContract.MostSignificantBit, prefixproofs.MostSignificantBit)
}

func FuzzPrefixProof_Verify(f *testing.F) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	manager, err := statemanager.NewWithMockedStateRoots(hashes)
	require.NoError(f, err)

	loCommit, err := manager.HistoryCommitmentAtMessage(ctx, 3)
	require.NoError(f, err)
	hiCommit, err := manager.HistoryCommitmentAtMessage(ctx, 7)
	require.NoError(f, err)
	packedProof, err := manager.PrefixProofUpToBatch(ctx, 0, 3, 7, 1)
	require.NoError(f, err)

	data, err := statemanager.ProofArgs.Unpack(packedProof)
	require.NoError(f, err)
	preExpansion := data[0].([][32]byte)
	proof := data[1].([][32]byte)
	preExp := make([]byte, 0)
	for _, item := range preExpansion {
		preExp = append(preExp, item[:]...)
	}
	prefixProof := make([]byte, 0)
	for _, item := range proof {
		prefixProof = append(prefixProof, item[:]...)
	}

	testcases := []prefixproofs.VerifyPrefixProofConfig{
		{
			PreRoot:  loCommit.Merkle,
			PreSize:  4,
			PostRoot: hiCommit.Merkle,
			PostSize: 8,
		},
		{
			PreRoot:  loCommit.Merkle,
			PreSize:  0,
			PostRoot: hiCommit.Merkle,
			PostSize: 0,
		},
		{
			PreRoot:  loCommit.Merkle,
			PreSize:  0,
			PostRoot: hiCommit.Merkle,
			PostSize: 100,
		},
	}
	for _, tc := range testcases {
		f.Add(tc.PreRoot.String(), tc.PreSize, tc.PostRoot.String(), tc.PostSize, hexutil.Encode(preExp), hexutil.Encode(prefixProof))
	}
	merkleTreeContract, _ := setupMerkleTreeContract(f)
	opts := &bind.CallOpts{}
	f.Fuzz(func(
		t *testing.T,
		preRootF string,
		preSizeF uint64,
		postRootF string,
		postSizeF uint64,
		preExpansionF string,
		prefixProofF string,
	) {
		preExpF := make([]common.Hash, 0)
		preArray := make([][32]byte, 0)
		expansionRaw, err := hexutil.Decode(preExpansionF)
		if err != nil {
			return
		}
		proofRaw, err := hexutil.Decode(prefixProofF)
		if err != nil {
			return
		}
		preExpansionArray := make([][32]byte, 0)
		for i := 0; i < len(expansionRaw); i += 32 {
			var r [32]byte
			if i+32 <= len(expansionRaw) {
				copy(r[:], expansionRaw[i:i+32])
			} else {
				copy(r[:], expansionRaw[i:])
			}
			preExpansionArray = append(preExpansionArray, r)
		}

		preExpansionHash := make([]common.Hash, len(preExpansionArray))
		for i := range preExpansionArray {
			preExpansionHash[i] = preExpansionArray[i]
		}

		proofArray := make([][32]byte, 0)
		for i := 0; i < len(proofRaw); i += 32 {
			var r [32]byte
			if i+32 <= len(proofRaw) {
				copy(r[:], proofRaw[i:i+32])
			} else {
				copy(r[:], proofRaw[i:])
			}
			proofArray = append(proofArray, r)
		}

		proofHash := make([]common.Hash, len(proofArray))
		for i := range proofArray {
			proofHash[i] = proofArray[i]
		}
		preRoot, err := hexutil.Decode(preRootF)
		if err != nil {
			return
		}
		postRoot, err := hexutil.Decode(postRootF)
		if err != nil {
			return
		}
		cfg := &prefixproofs.VerifyPrefixProofConfig{
			PreRoot:      common.BytesToHash(preRoot),
			PreSize:      preSizeF,
			PostRoot:     common.BytesToHash(postRoot),
			PostSize:     postSizeF,
			PreExpansion: preExpF,
			PrefixProof:  proofHash,
		}
		goErr := prefixproofs.VerifyPrefixProof(cfg)
		solErr := merkleTreeContract.VerifyPrefixProof(
			opts,
			cfg.PreRoot,
			big.NewInt(int64(cfg.PreSize)),
			cfg.PostRoot,
			big.NewInt(int64(cfg.PostSize)),
			preArray,
			proofArray,
		)

		if goErr == nil && solErr != nil {
			t.Errorf("Go verified, but solidity failed to verify: %+v", cfg)
		}
		if goErr != nil && solErr == nil {
			t.Errorf("Solidity verified, but go failed to verify: %+v", cfg)
		}
	})
}

func FuzzPrefixProof_MaximumAppendBetween_GoSolidityEquivalence(f *testing.F) {
	type prePost struct {
		pre  uint64
		post uint64
	}
	testcases := []prePost{
		{4, 8},
		{10, 0},
		{0, 0},
		{0, 1},
		{3, 3},
		{3, 4},
		{0, 15},
		{128, 512},
		{128, 200},
		{128, 1 << 20},
		{1 << 20, 1<<20 + 1},
	}
	for _, tc := range testcases {
		f.Add(tc.pre, tc.post)
	}
	merkleTreeContract, _ := setupMerkleTreeContract(f)
	opts := &bind.CallOpts{}
	f.Fuzz(func(t *testing.T, pre, post uint64) {
		gotGo, err1 := prefixproofs.MaximumAppendBetween(pre, post)
		gotSol, err2 := merkleTreeContract.MaximumAppendBetween(opts, big.NewInt(int64(pre)), big.NewInt(int64(post)))
		if err1 == nil && err2 == nil {
			if !gotSol.IsUint64() {
				t.Fatal("sol result was not a uint64")
			}
			if gotSol.Uint64() != gotGo {
				t.Errorf("sol %d != go %d", gotSol.Uint64(), gotGo)
			}
		}
	})
}

func FuzzPrefixProof_BitUtils_GoSolidityEquivalence(f *testing.F) {
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
		lsbGo, _ := prefixproofs.LeastSignificantBit(x)
		if lsbSol != nil {
			if !lsbSol.IsUint64() {
				t.Fatal("lsb sol not a uint64")
			}
			if lsbSol.Uint64() != lsbGo {
				t.Errorf("Mismatch lsb sol=%d, go=%d", lsbSol, lsbGo)
			}
		}
		msbSol, _ := merkleTreeContract.MostSignificantBit(opts, big.NewInt(int64(x)))
		msbGo, _ := prefixproofs.MostSignificantBit(x)
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
			goErr:      prefixproofs.ErrCannotBeZero,
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

func setupMerkleTreeContract(t testing.TB) (*mocksgen.MerkleTreeAccess, *backends.SimulatedBackend) {
	numChains := uint64(1)
	accs, backend := setupAccounts(t, numChains)
	_, _, merkleTreeContract, err := mocksgen.DeployMerkleTreeAccess(accs[0].txOpts, backend)
	if err != nil {
		t.Fatal(err)
	}
	backend.Commit()
	return merkleTreeContract, backend
}

// Represents a test EOA account in the simulated backend,
type testAccount struct {
	accountAddr common.Address
	txOpts      *bind.TransactOpts
}

func setupAccounts(t testing.TB, numAccounts uint64) ([]*testAccount, *backends.SimulatedBackend) {
	genesis := make(core.GenesisAlloc)
	gasLimit := uint64(100000000)

	accs := make([]*testAccount, numAccounts)
	for i := uint64(0); i < numAccounts; i++ {
		privKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
		if !ok {
			t.Fatal("not ok")
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
			t.Fatal(err)
		}
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
