//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkletree

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/storage"
)

func initializedMerkleAccumulatorForTesting() *merkleAccumulator.MerkleAccumulator {
	sto := storage.NewMemoryBacked(&burn.SystemBurner{})
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
		proof, err := ProofFromAccumulator(acc, leaf)
		Require(t, err)
		if proof == nil {
			Fail(t, i)
		}
		if proof.LeafHash != leaf {
			Fail(t, i)
		}
		if !proof.IsCorrect() {
			Fail(t, proof)
		}
		_, err = acc.Append(leaf)
		Require(t, err)
		root, err := acc.Root()
		Require(t, err)
		if proof.RootHash != root {
			Fail(t, i)
		}
	}
}
