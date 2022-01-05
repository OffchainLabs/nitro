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
	size           storage.WrappedUint64
	partials       []*common.Hash // nil if we are using backingStorage (in that case we access partials in backingStorage
}

func InitializeMerkleAccumulator(sto *storage.Storage) {
	// no initialization needed
}

func OpenMerkleAccumulator(sto *storage.Storage) *MerkleAccumulator {
	size := sto.OpenStorageBackedUint64(0)
	return &MerkleAccumulator{sto, size, nil}
}

func NewNonpersistentMerkleAccumulator() *MerkleAccumulator {
	return &MerkleAccumulator{nil, &storage.MemoryBackedUint64{}, make([]*common.Hash, 0)}
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
	mbu := &storage.MemoryBackedUint64{}
	mbu.Set(size)
	return &MerkleAccumulator{nil, mbu, partials}
}

func (acc *MerkleAccumulator) NonPersistentClone() *MerkleAccumulator {
	numPartials := CalcNumPartials(acc.size.Get())
	partials := make([]*common.Hash, numPartials)
	for i := uint64(0); i < numPartials; i++ {
		partials[i] = acc.getPartial(i)
	}
	mbu := &storage.MemoryBackedUint64{}
	mbu.Set(acc.size.Get())
	return &MerkleAccumulator{nil, mbu, partials}
}

func (acc *MerkleAccumulator) getPartial(level uint64) *common.Hash {
	if acc.backingStorage == nil {
		if acc.partials[level] == nil {
			h := common.Hash{}
			acc.partials[level] = &h
		}
		return acc.partials[level]
	} else {
		ret := acc.backingStorage.GetByUint64(2 + level)
		return &ret
	}
}

func (acc *MerkleAccumulator) GetPartials() []*common.Hash {
	partials := make([]*common.Hash, CalcNumPartials(acc.size.Get()))
	for i := range partials {
		p := *acc.getPartial(uint64(i))
		partials[i] = &p
	}
	return partials
}

func (acc *MerkleAccumulator) setPartial(level uint64, val *common.Hash) {
	if acc.backingStorage != nil {
		acc.backingStorage.SetByUint64(2+level, *val)
	} else if level == uint64(len(acc.partials)) {
		acc.partials = append(acc.partials, val)
	} else {
		acc.partials[level] = val
	}
}

func (acc *MerkleAccumulator) Append(itemHash common.Hash) []MerkleTreeNodeEvent {
	acc.size.Set(acc.size.Get() + 1)
	events := []MerkleTreeNodeEvent{}

	level := uint64(0)
	soFar := itemHash.Bytes()
	for {
		if level == CalcNumPartials(acc.size.Get()-1) { // -1 to counteract the acc.size++ at top of this function
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
		events = append(events, MerkleTreeNodeEvent{level, acc.size.Get() - 1, common.BytesToHash(soFar)})
	}
}

func (acc *MerkleAccumulator) Size() uint64 {
	return acc.size.Get()
}

func (acc *MerkleAccumulator) Root() common.Hash {
	if acc.size.Get() == 0 {
		return common.Hash{}
	}

	var hashSoFar *common.Hash
	var capacityInHash uint64
	capacity := uint64(1)
	for level := uint64(0); level < CalcNumPartials(acc.size.Get()); level++ {
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
	root = acc.Root()
	numPartials := CalcNumPartials(acc.size.Get())
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
