//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkletree

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/storage"
)

func initializedMerkleAccumulatorForTesting() *merkleAccumulator.MerkleAccumulator {
	sto := storage.NewMemoryBacked()
	merkleAccumulator.InitializeMerkleAccumulator(sto)
	return merkleAccumulator.OpenMerkleAccumulator(sto)
}

func TestProofForNext(t *testing.T) {
	leaves := make([]common.Hash, 13)
	for i := range leaves {
		leaves[i] = pseudorandomForTesting(uint64(i))
	}

	acc := initializedMerkleAccumulatorForTesting()
	for i, leaf := range leaves {
		proof := ProofFromAccumulator(acc, leaf)
		if proof == nil {
			t.Fatal(i)
		}
		if proof.LeafHash != leaf {
			t.Fatal(i)
		}
		if !proof.IsCorrect() {
			t.Fatal(proof)
		}
		acc.Append(leaf)
		if proof.RootHash != acc.Root() {
			t.Fatal(i)
		}
	}
}
