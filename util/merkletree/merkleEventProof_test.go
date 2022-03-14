//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package merkletree

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
	"github.com/offchainlabs/nitro/arbos/storage"
)

func initializedMerkleAccumulatorForTesting() *merkleAccumulator.MerkleAccumulator {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(false))
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

func ProofFromAccumulator(acc *merkleAccumulator.MerkleAccumulator, nextHash common.Hash) (*MerkleProof, error) {
	origPartials, err := acc.GetPartials()
	if err != nil {
		return nil, err
	}
	partials := make([]common.Hash, len(origPartials))
	for i, orig := range origPartials {
		partials[i] = *orig
	}
	clone, err := acc.NonPersistentClone()
	if err != nil {
		return nil, err
	}
	_, err = clone.Append(nextHash)
	if err != nil {
		return nil, err
	}
	root, _ := clone.Root()
	size, err := acc.Size()
	if err != nil {
		return nil, err
	}

	return &MerkleProof{
		RootHash:  root,
		LeafHash:  nextHash,
		LeafIndex: size,
		Proof:     partials,
	}, nil
}
