//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkletree

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
)

func NewMerkleTreeFromAccumulator(acc *merkleAccumulator.MerkleAccumulator) MerkleTree {
	partials := acc.GetPartials()
	if len(partials) == 0 {
		return NewEmptyMerkleTree()
	}
	var tree MerkleTree
	capacity := uint64(1)
	for level, partial := range partials {
		if *partial != (common.Hash{}) {
			var thisLevel MerkleTree
			if level == 0 {
				thisLevel = NewMerkleLeaf(*partial)
			} else {
				thisLevel = NewSummaryMerkleTree(*partial, capacity)
			}
			if tree == nil {
				tree = thisLevel
			} else {
				for tree.Capacity() < capacity {
					tree = NewMerkleInternal(tree, NewMerkleEmpty(tree.Capacity()))
				}
				tree = NewMerkleInternal(thisLevel, tree)
			}
		}
		capacity *= 2
	}

	return tree
}

func NewMerkleTreeFromEvents(
	events []merkleAccumulator.MerkleTreeNodeEvent, // latest event at each Level
) MerkleTree {
	return NewMerkleTreeFromAccumulator(NewNonPersistentMerkleAccumulatorFromEvents(events))
}

func NewNonPersistentMerkleAccumulatorFromEvents(events []merkleAccumulator.MerkleTreeNodeEvent) *merkleAccumulator.MerkleAccumulator {
	partials := make([]*common.Hash, len(events))
	zero := common.Hash{}
	for i := range partials {
		partials[i] = &zero
	}

	latestSeen := uint64(0)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.NumLeaves > latestSeen {
			latestSeen = event.NumLeaves
			partials[i] = &event.Hash
		}
	}
	return merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(partials)
}

func ProofFromAccumulator(acc *merkleAccumulator.MerkleAccumulator, nextHash common.Hash) *MerkleProof {
	origPartials := acc.GetPartials()
	partials := make([]common.Hash, len(origPartials))
	for i, orig := range origPartials {
		partials[i] = *orig
	}
	clone := acc.NonPersistentClone()
	_ = clone.Append(nextHash)
	return &MerkleProof{
		RootHash:  clone.Root(),
		LeafHash:  nextHash,
		LeafIndex: acc.Size(),
		Proof:     partials,
	}
}
