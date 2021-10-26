//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleAccumulator

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
	partials       []*common.Hash // nil means we haven't yet loaded it from backingStorage
}

func InitializeMerkleAccumulator(sto *storage.Storage) {
	// no initialization needed
}

func OpenMerkleAccumulator(sto *storage.Storage) *MerkleAccumulator {
	size := sto.GetByInt64(0).Big().Uint64()
	numPartials := sto.GetByInt64(1).Big().Uint64()
	return &MerkleAccumulator{sto, size, numPartials, make([]*common.Hash, numPartials)}
}

func NewNonpersistentMerkleAccumulator() *MerkleAccumulator {
	return &MerkleAccumulator{nil, 0, 0, make([]*common.Hash, 0)}
}

func NewNonpersistentMerkleAccumulatorFromPartials(partials []*common.Hash) *MerkleAccumulator {
	size := uint64(0)
	levelSize := uint64(1)
	for i := range partials {
		if *partials[i] != (common.Hash{}) {
			size += levelSize
		}
		levelSize *= 2
	}
	return &MerkleAccumulator{nil, size, uint64(len(partials)), partials}
}

func (acc *MerkleAccumulator) NonPersistentClone() *MerkleAccumulator {
	partials := make([]*common.Hash, acc.numPartials)
	for i := uint64(0); i < acc.numPartials; i++ {
		partials[i] = acc.getPartial(i)
	}
	return &MerkleAccumulator{nil, acc.size, acc.numPartials, partials}
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

func (acc *MerkleAccumulator) GetPartials() []*common.Hash {
	partials := make([]*common.Hash, acc.numPartials)
	for i := range partials {
		p := *acc.getPartial(uint64(i))
		partials[i] = &p
	}
	return partials
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

func (acc *MerkleAccumulator) Append(itemHash common.Hash) *MerkleAccumulatorUpdateEvent {
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
			return &MerkleAccumulatorUpdateEvent{level, acc.size - 1, h}
		}
		thisLevel := acc.getPartial(level)
		if *thisLevel == (common.Hash{}) {
			h := common.BytesToHash(soFar)
			acc.setPartial(level, &h)
			return &MerkleAccumulatorUpdateEvent{level, acc.size - 1, h}
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

	var hashSoFar *common.Hash
	var capacityInHash uint64
	capacity := uint64(1)
	for level := uint64(0); level < acc.numPartials; level++ {
		partial := acc.getPartial(level)
		if *partial != (common.Hash{}) {
			if hashSoFar == nil {
				hashSoFar = partial
				capacityInHash = capacity
			} else {
				for capacityInHash < capacity {
					h := crypto.Keccak256Hash(hashSoFar.Bytes(), make([]byte, 32))
					hashSoFar = &h
					capacityInHash *= 2
				}
				h := crypto.Keccak256Hash(partial.Bytes(), hashSoFar.Bytes())
				hashSoFar = &h
				capacityInHash = 2 * capacity
			}
		}
		capacity *= 2
	}
	return *hashSoFar
}

func (acc *MerkleAccumulator) StateForExport() (size uint64, root common.Hash, partials []common.Hash) {
	size = acc.size
	root = acc.Root()
	numPartials := acc.numPartials
	partials = make([]common.Hash, numPartials)
	for i := uint64(0); i < numPartials; i++ {
		partials[i] = *acc.getPartial(i)
	}
	return
}

type MerkleAccumulatorUpdateEvent struct {
	Level   uint64
	LeafNum uint64
	Hash    common.Hash
}
