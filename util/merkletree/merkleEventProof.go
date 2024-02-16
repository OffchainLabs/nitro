// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package merkletree

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

func NewMerkleTreeFromAccumulator(acc *merkleAccumulator.MerkleAccumulator) (MerkleTree, error) {
	partials, err := acc.GetPartials()
	if err != nil {
		return nil, err
	}
	if len(partials) == 0 {
		return NewEmptyMerkleTree(), nil
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

	return tree, nil
}

func NewMerkleTreeFromEvents(
	events []merkleAccumulator.MerkleTreeNodeEvent, // latest event at each Level
) (MerkleTree, error) {
	acc, err := NewNonPersistentMerkleAccumulatorFromEvents(events)
	if err != nil {
		return nil, err
	}
	return NewMerkleTreeFromAccumulator(acc)
}

func NewNonPersistentMerkleAccumulatorFromEvents(
	events []merkleAccumulator.MerkleTreeNodeEvent,
) (*merkleAccumulator.MerkleAccumulator, error) {

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
