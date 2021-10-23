//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleTree

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
)

type MerkleAccumulator struct {
	backingStorage *storage.Storage
	size           uint64
	numPartials    uint64
	partials       []*common.Hash
}

func InitializeMerkleAccumulator(sto *storage.Storage) {
	// no initialization needed
}

func OpenMerkleAccumulator(sto *storage.Storage) *MerkleAccumulator {
	size := sto.GetByInt64(0).Big().Uint64()
	numPartials := sto.GetByInt64(1).Big().Uint64()
	return &MerkleAccumulator{sto, size, numPartials, make([]*common.Hash, numPartials) }
}

func NewNonpersistentMerkleAccumulator() *MerkleAccumulator {
	return &MerkleAccumulator{ nil, 0, 0, make([]*common.Hash, 0) }
}

func (acc *MerkleAccumulator) getPartial(level uint64) *common.Hash {
	if acc.partials[level] == nil {
		if acc.backingStorage != nil {
			h := acc.backingStorage.GetByInt64(int64(2 + level))
			acc.partials[level] = &h
		} else {
			h := common.Hash{}
			acc.partials[level] = &h
		}
	}
	return acc.partials[level]
}

func (acc *MerkleAccumulator) setPartial(level uint64, val *common.Hash) {
	if level == acc.numPartials {
		acc.numPartials++
		if acc.backingStorage != nil {
			acc.backingStorage.SetByInt64(1, util.IntToHash(int64(acc.numPartials)))
		}
		acc.partials = append(acc.partials, val)
	} else {
		acc.partials[level] = val
	}
	if acc.backingStorage != nil {
		acc.backingStorage.SetByInt64(int64(2+level), *val)
	}
}

func (acc *MerkleAccumulator) Append(itemHash common.Hash) *EventForTreeBuilding {
	acc.size++
	if acc.backingStorage != nil {
		acc.backingStorage.SetByInt64(0, util.IntToHash(int64(acc.size)))
	}
	level := uint64(0)
	soFar := itemHash.Bytes()
	for {
		if level == acc.numPartials {
			h := common.BytesToHash(soFar)
			acc.setPartial(level, &h)
			return &EventForTreeBuilding{level, acc.size-1, h}
		}
		thisLevel := acc.getPartial(level)
		if *thisLevel == (common.Hash{}) {
			h := common.BytesToHash(soFar)
			acc.setPartial(level, &h)
			return &EventForTreeBuilding{level, acc.size-1,h}
		}
		soFar = crypto.Keccak256(thisLevel.Bytes(), soFar)
		h := common.Hash{}
		acc.setPartial(level, &h)
		level += 1
	}
}

func (acc *MerkleAccumulator) Size() uint64 {
	return acc.size
}

func (acc *MerkleAccumulator) Root() common.Hash {
	if acc.size == 0 {
		return common.Hash{}
	}
	if acc.size == 1 {
		return *acc.getPartial(0)
	}
	ret := make([]byte, 32)
	emptySoFar := true
	for i := uint64(0); i < acc.numPartials; i++ {
		thisLevel := acc.getPartial(i)
		if *thisLevel == (common.Hash{}) {
			if !emptySoFar {
				ret = crypto.Keccak256(make([]byte, 32), ret)
			}
		} else {
			if emptySoFar {
				if i+1 == acc.numPartials {
					ret = thisLevel.Bytes()
				} else {
					emptySoFar = false
					ret = crypto.Keccak256(thisLevel.Bytes(), make([]byte, 32))
				}
			} else {
				ret = crypto.Keccak256(thisLevel.Bytes(), ret)
			}
		}
	}

	return common.BytesToHash(ret)
}

func (acc *MerkleAccumulator) ToMerkleTree() MerkleTree {
	if acc.numPartials == 0 {
		return NewEmptyMerkleTree()
	}
	var tree MerkleTree
	capacity := uint64(1)
	for level := uint64(0); level < acc.numPartials; level++ {
		partial := acc.getPartial(level)
		if *partial != (common.Hash{}) {
			if tree == nil {
				tree = &merkleCompleteSubtreeSummary{ *partial, capacity, capacity }
			} else {
				for tree.Capacity() < capacity {
					tree = newMerkleInternal(tree, newMerkleEmpty(tree.Capacity()))
				}
				tree = newMerkleInternal(&merkleCompleteSubtreeSummary{ *partial, capacity, capacity }, tree)
			}
		}
		capacity *= 2
	}

	return tree
}

func NewNonPersistentMerkleAccumulatorFromEvents(events []EventForTreeBuilding) *MerkleAccumulator {
	acc := NewNonpersistentMerkleAccumulator()
	acc.numPartials = uint64(len(events))
	acc.partials = make([]*common.Hash, len(events))
	zero := common.Hash{}
	for i, _ := range acc.partials {
		acc.partials[i] = &zero
	}

	latestSeen := uint64(0)
	for i := len(events)-1; i >= 0; i-- {
		event := events[i]
		if event.leafNum > latestSeen {
			latestSeen = event.leafNum
			acc.size = event.leafNum
			acc.setPartial(uint64(i), &event.hash)
		}
		if acc.size <= event.leafNum {
			acc.size = event.leafNum + 1
		}
	}
	return acc
}
