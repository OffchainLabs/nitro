//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkletree

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"testing"
)

func initializedMerkleAccumulatorForTesting() *merkleAccumulator.MerkleAccumulator {
	sto := storage.NewMemoryBacked()
	merkleAccumulator.InitializeMerkleAccumulator(sto)
	return merkleAccumulator.OpenMerkleAccumulator(sto)
}

func TestReconstructFromEvents(t *testing.T) {
	leaves := make([]common.Hash, 13)
	for i := range leaves {
		leaves[i] = pseudorandomForTesting(uint64(i))
	}

	acc := initializedMerkleAccumulatorForTesting()
	events := []merkleAccumulator.MerkleTreeNodeEvent{}

	for i, leaf := range leaves {
		thisLeafEvents := acc.Append(leaf)
		for _, ev := range thisLeafEvents {
			if ev.Level >= uint64(len(events)) {
				events = append(events, ev)
			} else {
				events[ev.Level] = ev
			}
		}
		if acc.Root() != NewMerkleTreeFromAccumulator(acc).Hash() {
			t.Fatal(i)
		}
	}

	if acc.Root() != NewMerkleTreeFromAccumulator(acc).Hash() {
		t.Fatal()
	}

	reconstructedAcc := NewNonPersistentMerkleAccumulatorFromEvents(events)
	if reconstructedAcc.Root() != acc.Root() {
		t.Fatal()
	}

	if reconstructedAcc.Size() != acc.Size() {
		t.Fatal(acc.Size())
	}
	recPartials := reconstructedAcc.GetPartials()
	accPartials := acc.GetPartials()
	if len(recPartials) != len(accPartials) {
		t.Fatal()
	}
	for i, rpart := range recPartials {
		if *rpart != *accPartials[i] {
			t.Fatal(i)
		}
	}

	reconstructedTree := NewMerkleTreeFromAccumulator(reconstructedAcc)
	if reconstructedAcc.Root() != reconstructedTree.Hash() {
		t.Fatal()
	}

	reconstructedTree = NewMerkleTreeFromEvents(events)
	if reconstructedTree.Hash() != acc.Root() {
		t.Fatal()
	}
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
