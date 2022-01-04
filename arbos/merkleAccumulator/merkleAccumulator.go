//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleAccumulator

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	util_math "github.com/offchainlabs/arbstate/util"
)

type MerkleAccumulator struct {
	backingStorage *storage.Storage
	size           uint64         // stored in backingStorage, slot 0, if backingStorage exists
	numPartials    uint64         // not stored in the DB, but calculated from size
	partials       []*common.Hash // nil means we haven't yet loaded it from backingStorage
}

func InitializeMerkleAccumulator(sto *storage.Storage) {
	// no initialization needed
}

func OpenMerkleAccumulator(sto *storage.Storage) *MerkleAccumulator {
	size := sto.GetUint64ByUint64(0)
	numPartials := CalcNumPartials(size)
	return &MerkleAccumulator{sto, size, numPartials, make([]*common.Hash, numPartials)}
}

func NewNonpersistentMerkleAccumulator() *MerkleAccumulator {
	return &MerkleAccumulator{nil, 0, 0, make([]*common.Hash, 0)}
}

func CalcNumPartials(size uint64) uint64 {
	return util_math.Log2ceil(size)
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
			h := acc.backingStorage.GetByUint64(2 + level)
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
		acc.partials = append(acc.partials, val)
	} else {
		acc.partials[level] = val
	}
	if acc.backingStorage != nil {
		acc.backingStorage.SetByUint64(2+level, *val)
	}
}

func (acc *MerkleAccumulator) Append(itemHash common.Hash) []MerkleTreeNodeEvent {
	acc.size++
	events := []MerkleTreeNodeEvent{}

	if acc.backingStorage != nil {
		acc.backingStorage.SetUint64ByUint64(0, acc.size)
	}
	level := uint64(0)
	soFar := itemHash.Bytes()
	for {
		if level == acc.numPartials {
			h := common.BytesToHash(soFar)
			acc.setPartial(level, &h)
			return events
		}
		thisLevel := acc.getPartial(level)
		if *thisLevel == (common.Hash{}) {
			h := common.BytesToHash(soFar)
			acc.setPartial(level, &h)
			return events
		}
		soFar = crypto.Keccak256(thisLevel.Bytes(), soFar)
		h := common.Hash{}
		acc.setPartial(level, &h)
		level += 1
		events = append(events, MerkleTreeNodeEvent{level, acc.size - 1, common.BytesToHash(soFar)})
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

type MerkleTreeNodeEvent struct {
	Level     uint64
	NumLeaves uint64
	Hash      common.Hash
}
