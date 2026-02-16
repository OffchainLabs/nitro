// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package prefix

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"

	"github.com/offchainlabs/nitro/bold/commitment/history"
	"github.com/offchainlabs/nitro/bold/commitment/proof/prefix"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/protocol"
	"github.com/offchainlabs/nitro/bold/state"
	"github.com/offchainlabs/nitro/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
)

func TestPrefixProofGeneration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	merkleTreeContract, _ := setupMerkleTreeContract(t)
	verify := func(t *testing.T, computed *prefixProofComputation) {
		prefixExpRaw := make([][32]byte, len(computed.prefixExpansion))
		for i := 0; i < len(computed.prefixExpansion); i++ {
			var r [32]byte
			copy(r[:], computed.prefixExpansion[i][:])
			prefixExpRaw[i] = r
		}
		proofRaw := make([][32]byte, len(computed.proof))
		for i := 0; i < len(computed.proof); i++ {
			var r [32]byte
			copy(r[:], computed.proof[i][:])
			proofRaw[i] = r
		}
		err := prefix.VerifyPrefixProof(&prefix.VerifyPrefixProofConfig{
			PreRoot:      computed.prefixRoot,
			PreSize:      computed.prefixTotalLeaves,
			PostRoot:     computed.fullRoot,
			PostSize:     computed.fullTreeTotalLeaves,
			PreExpansion: computed.prefixExpansion,
			PrefixProof:  computed.proof,
		})
		require.NoError(t, err)
		err = merkleTreeContract.VerifyPrefixProof(
			&bind.CallOpts{},
			computed.prefixRoot,
			new(big.Int).SetUint64(computed.prefixTotalLeaves),
			computed.fullRoot,
			new(big.Int).SetUint64(computed.fullTreeTotalLeaves),
			prefixExpRaw,
			proofRaw,
		)
		require.NoError(t, err)
	}
	tests := []struct {
		realLength    uint64
		virtualLength uint64
	}{
		{1, 4},
		{2, 4},
		{3, 4},
		{4, 4},
		{1, 8},
		{2, 8},
		{3, 8},
		{4, 8},
		{5, 8},
		{6, 8},
		{7, 8},
		{8, 8},
		{1, 16},
	}

	for _, tt := range tests {
		for virtual := tt.realLength; virtual < tt.virtualLength; virtual++ {
			for prefixIndex := uint64(0); prefixIndex < virtual-1; prefixIndex++ {
				t.Run(fmt.Sprintf("real length %d, virtual %d, prefix index %d", tt.realLength, virtual, prefixIndex), func(t *testing.T) {
					legacy := computeLegacyPrefixProof(t, ctx, virtual, prefixIndex)
					optimized := computeOptimizedPrefixProof(t, tt.realLength, virtual, prefixIndex)
					verify(t, legacy)
					verify(t, optimized)
				})
			}
		}
	}
}

type prefixProofComputation struct {
	prefixRoot          common.Hash
	fullRoot            common.Hash
	prefixTotalLeaves   uint64
	fullTreeTotalLeaves uint64
	prefixExpansion     []common.Hash
	proof               []common.Hash
}

func computeOptimizedPrefixProof(t *testing.T, numRealHashes uint64, virtual uint64, prefixIndex uint64) *prefixProofComputation {
	// Computes the prefix proof and expansion.
	simpleHash := crypto.Keccak256Hash([]byte("foo"))
	hashes := make([]common.Hash, prefixIndex+1)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = simpleHash
	}

	// Computes the prefix root.
	prefixRoot, err := history.ComputeRoot(hashes, prefixIndex+1)
	require.NoError(t, err)

	// Computes the full tree root.
	hashes = make([]common.Hash, numRealHashes)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = simpleHash
	}
	fullTreeRoot, err := history.ComputeRoot(hashes, virtual)
	require.NoError(t, err)

	// Computes the prefix proof.
	hashes = make([]common.Hash, numRealHashes)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = simpleHash
	}
	prefixExp, proof, err := history.GeneratePrefixProof(uint64(prefixIndex), hashes, virtual)
	require.NoError(t, err)
	return &prefixProofComputation{
		prefixRoot:          prefixRoot,
		fullRoot:            fullTreeRoot,
		prefixTotalLeaves:   uint64(prefixIndex) + 1,
		fullTreeTotalLeaves: uint64(virtual),
		prefixExpansion:     prefixExp,
		proof:               proof,
	}
}

func computeLegacyPrefixProof(t *testing.T, ctx context.Context, numHashes uint64, prefixIndex uint64) *prefixProofComputation {
	simpleHash := crypto.Keccak256Hash([]byte("foo"))
	hashes := make([]common.Hash, numHashes)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = simpleHash
	}
	manager, err := stateprovider.NewWithMockedStateRoots(hashes)
	require.NoError(t, err)

	wasmModuleRoot := common.Hash{}
	startMessageNumber := state.Height(0)
	fromMessageNumber := state.Height(prefixIndex)
	req := &state.HistoryCommitmentRequest{
		AssertionMetadata: &state.AssociatedAssertionMetadata{
			WasmModuleRoot: wasmModuleRoot,
			FromState: protocol.GoGlobalState{
				Batch:      0,
				PosInBatch: uint64(startMessageNumber),
			},
			BatchLimit: 10,
		},
		UpperChallengeOriginHeights: []state.Height{},
		UpToHeight:                  option.Some(state.Height(fromMessageNumber)),
	}
	loCommit, err := manager.HistoryCommitment(ctx, req)
	require.NoError(t, err)

	req.UpToHeight = option.Some(state.Height(numHashes - 1))
	hiCommit, err := manager.HistoryCommitment(ctx, req)
	require.NoError(t, err)

	packedProof, err := manager.PrefixProof(ctx, req, fromMessageNumber)
	require.NoError(t, err)

	data, err := stateprovider.ProofArgs.Unpack(packedProof)
	require.NoError(t, err)
	preExpansion, ok := data[0].([][32]byte)
	require.Equal(t, true, ok)
	proof, ok := data[1].([][32]byte)
	require.Equal(t, true, ok)

	preExpansionHashes := make([]common.Hash, len(preExpansion))
	for i := 0; i < len(preExpansion); i++ {
		preExpansionHashes[i] = preExpansion[i]
	}
	prefixProof := make([]common.Hash, len(proof))
	for i := 0; i < len(proof); i++ {
		prefixProof[i] = proof[i]
	}
	return &prefixProofComputation{
		prefixRoot:          loCommit.Merkle,
		fullRoot:            hiCommit.Merkle,
		prefixTotalLeaves:   uint64(prefixIndex) + 1,
		fullTreeTotalLeaves: uint64(numHashes),
		prefixExpansion:     preExpansionHashes,
		proof:               prefixProof,
	}
}

func setupMerkleTreeContract(t testing.TB) (*mocksgen.MerkleTreeAccess, *simulated.Backend) {
	numChains := uint64(1)
	accs, backend := setupAccounts(t, numChains)
	_, _, merkleTreeContract, err := mocksgen.DeployMerkleTreeAccess(accs[0].txOpts, backend.Client())
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

func setupAccounts(t testing.TB, numAccounts uint64) ([]*testAccount, *simulated.Backend) {
	genesis := make(types.GenesisAlloc)
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
		genesis[addr] = types.Account{Balance: startingBalance}
		accs[i] = &testAccount{
			accountAddr: addr,
			txOpts:      txOpts,
		}
	}
	backend := simulated.NewBackend(genesis, simulated.WithBlockGasLimit(gasLimit))
	return accs, backend
}
