//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleTree

import (
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestMerkleProofs(t *testing.T) {
	items := make([]common.Hash, 13)
	for i, _ := range items {
		items[i] = pseudorandomForTesting(uint64(i))
	}

	tree := NewEmptyMerkleTree()
	for i, item := range items {
		tree = tree.Append(item)
		for j := 0; j <= i; j++ {
			proof := tree.Prove(uint64(j))
			if proof.LeafHash != items[j] {
				t.Fatal()
			}
			if proof.RootHash != tree.Hash() {
				t.Fatal()
			}
			if proof == nil {
				t.Fatal(j, tree.Capacity())
			}
			if !proof.IsCorrect() {
				t.Fatal(j, tree.Capacity(), len(proof.Proof))
			}
		}
	}
}
