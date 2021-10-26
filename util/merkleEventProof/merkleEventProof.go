//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleEventProof

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/util/merkletree"
)

func NewMerkleTreeFromAccumulator(acc *merkleAccumulator.MerkleAccumulator) merkletree.MerkleTree {
	partials := acc.GetPartials()
	if len(partials) == 0 {
		return merkletree.NewEmptyMerkleTree()
	}
	var tree merkletree.MerkleTree
	capacity := uint64(1)
	for level, partial := range partials {
		if *partial != (common.Hash{}) {
			var thisLevel merkletree.MerkleTree
			if level == 0 {
				thisLevel = merkletree.NewMerkleLeaf(*partial)
			} else {
				thisLevel = merkletree.NewSummaryMerkleTree(*partial, capacity)
			}
			if tree == nil {
				tree = thisLevel
			} else {
				for tree.Capacity() < capacity {
					tree = merkletree.NewMerkleInternal(tree, merkletree.NewMerkleEmpty(tree.Capacity()))
				}
				tree = merkletree.NewMerkleInternal(thisLevel, tree)
			}
		}
		capacity *= 2
	}

	return tree
}

func NewMerkleTreeFromEvents(
	events []merkleAccumulator.MerkleAccumulatorUpdateEvent, // latest event at each Level
) merkletree.MerkleTree {
	return NewMerkleTreeFromAccumulator(NewNonPersistentMerkleAccumulatorFromEvents(events))
}

func NewNonPersistentMerkleAccumulatorFromEvents(events []merkleAccumulator.MerkleAccumulatorUpdateEvent) *merkleAccumulator.MerkleAccumulator {
	partials := make([]*common.Hash, len(events))
	zero := common.Hash{}
	for i := range partials {
		partials[i] = &zero
	}

	latestSeen := uint64(0)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.LeafNum > latestSeen {
			latestSeen = event.LeafNum
			partials[i] = &event.Hash
		}
	}
	return merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(partials)
}

func ProofFromAccumulator(acc *merkleAccumulator.MerkleAccumulator, nextHash common.Hash) *merkletree.MerkleProof {
	origPartials := acc.GetPartials()
	partials := make([]common.Hash, len(origPartials))
	for i, orig := range origPartials {
		partials[i] = *orig
	}
	clone := acc.NonPersistentClone()
	_ = clone.Append(nextHash)
	return &merkletree.MerkleProof{
		RootHash:  clone.Root(),
		LeafHash:  nextHash,
		LeafIndex: acc.Size(),
		Proof:     partials,
	}
}
