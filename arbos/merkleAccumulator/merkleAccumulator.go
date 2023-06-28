// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package merkleAccumulator

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
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
	return &MerkleAccumulator{sto, &size, nil}
}

func NewNonpersistentMerkleAccumulator() *MerkleAccumulator {
	return &MerkleAccumulator{nil, &storage.MemoryBackedUint64{}, make([]*common.Hash, 0)}
}

func CalcNumPartials(size uint64) uint64 {
	return arbmath.Log2ceil(size)
}

func NewNonpersistentMerkleAccumulatorFromPartials(partials []*common.Hash) (*MerkleAccumulator, error) {
	size := uint64(0)
	levelSize := uint64(1)
	for i := range partials {
		if *partials[i] != (common.Hash{}) {
			size += levelSize
		}
		levelSize *= 2
	}
	mbu := &storage.MemoryBackedUint64{}
	return &MerkleAccumulator{nil, mbu, partials}, mbu.Set(size)
}

func (acc *MerkleAccumulator) NonPersistentClone() (*MerkleAccumulator, error) {
	size, err := acc.size.Get()
	if err != nil {
		return nil, err
	}
	numPartials := CalcNumPartials(size)
	partials := make([]*common.Hash, numPartials)
	for i := uint64(0); i < numPartials; i++ {
		partial, err := acc.getPartial(i)
		if err != nil {
			return nil, err
		}
		partials[i] = partial
	}
	mbu := &storage.MemoryBackedUint64{}
	return &MerkleAccumulator{nil, mbu, partials}, mbu.Set(size)
}

func (acc *MerkleAccumulator) Keccak(data ...[]byte) ([]byte, error) {
	if acc.backingStorage != nil {
		return acc.backingStorage.Keccak(data...)
	}
	return crypto.Keccak256(data...), nil
}

func (acc *MerkleAccumulator) KeccakHash(data ...[]byte) (common.Hash, error) {
	if acc.backingStorage != nil {
		return acc.backingStorage.KeccakHash(data...)
	}
	return crypto.Keccak256Hash(data...), nil
}

func (acc *MerkleAccumulator) getPartial(level uint64) (*common.Hash, error) {
	if acc.backingStorage == nil {
		if acc.partials[level] == nil {
			h := common.Hash{}
			acc.partials[level] = &h
		}
		return acc.partials[level], nil
	}
	ret, err := acc.backingStorage.GetByUint64(2 + level)
	return &ret, err
}

func (acc *MerkleAccumulator) GetPartials() ([]*common.Hash, error) {
	size, err := acc.size.Get()
	if err != nil {
		return nil, err
	}
	partials := make([]*common.Hash, CalcNumPartials(size))
	for i := range partials {
		p, err := acc.getPartial(uint64(i))
		if err != nil {
			return nil, err
		}
		partials[i] = p
	}
	return partials, nil
}

func (acc *MerkleAccumulator) setPartial(level uint64, val *common.Hash) error {
	if acc.backingStorage != nil {
		err := acc.backingStorage.SetByUint64(2+level, *val)
		if err != nil {
			return err
		}
	} else if level == uint64(len(acc.partials)) {
		acc.partials = append(acc.partials, val)
	} else {
		acc.partials[level] = val
	}
	return nil
}

// Note: itemHash is hashed before being included in the tree, to prevent confusing leafs with branches.
func (acc *MerkleAccumulator) Append(itemHash common.Hash) ([]MerkleTreeNodeEvent, uint64, error) {
	size, err := acc.size.Increment()
	if err != nil {
		return nil, 0, err
	}
	events := []MerkleTreeNodeEvent{}

	level := uint64(0)
	soFar := crypto.Keccak256(itemHash.Bytes())
	for {
		if level == CalcNumPartials(size-1) { // -1 to counteract the acc.size++ at top of this function
			h := common.BytesToHash(soFar)
			err := acc.setPartial(level, &h)
			return events, size, err
		}
		thisLevel, err := acc.getPartial(level)
		if err != nil {
			return nil, size, err
		}
		if *thisLevel == (common.Hash{}) {
			h := common.BytesToHash(soFar)
			err := acc.setPartial(level, &h)
			return events, size, err
		}
		soFar, err = acc.Keccak(thisLevel.Bytes(), soFar)
		if err != nil {
			return nil, size, err
		}
		h := common.Hash{}
		err = acc.setPartial(level, &h)
		if err != nil {
			return nil, size, err
		}
		level += 1
		events = append(events, MerkleTreeNodeEvent{level, size - 1, common.BytesToHash(soFar)})
	}
}

func (acc *MerkleAccumulator) Size() (uint64, error) {
	return acc.size.Get()
}

func (acc *MerkleAccumulator) Root() (common.Hash, error) {
	size, err := acc.size.Get()
	if size == 0 || err != nil {
		return common.Hash{}, err
	}

	var hashSoFar *common.Hash
	var capacityInHash uint64
	capacity := uint64(1)
	for level := uint64(0); level < CalcNumPartials(size); level++ {
		partial, err := acc.getPartial(level)
		if err != nil {
			return common.Hash{}, err
		}
		if *partial != (common.Hash{}) {
			if hashSoFar == nil {
				hashSoFar = partial
				capacityInHash = capacity
			} else {
				for capacityInHash < capacity {
					h, err := acc.KeccakHash(hashSoFar.Bytes(), make([]byte, 32))
					if err != nil {
						return common.Hash{}, err
					}
					hashSoFar = &h
					capacityInHash *= 2
				}
				h, err := acc.KeccakHash(partial.Bytes(), hashSoFar.Bytes())
				if err != nil {
					return common.Hash{}, err
				}
				hashSoFar = &h
				capacityInHash = 2 * capacity
			}
		}
		capacity *= 2
	}
	return *hashSoFar, nil
}

func (acc *MerkleAccumulator) StateForExport() (uint64, common.Hash, []common.Hash, error) {
	root, err := acc.Root()
	if err != nil {
		return 0, common.Hash{}, nil, err
	}
	size, err := acc.size.Get()
	if err != nil {
		return 0, common.Hash{}, nil, err
	}
	numPartials := CalcNumPartials(size)
	partials := make([]common.Hash, numPartials)
	for i := uint64(0); i < numPartials; i++ {
		partial, err := acc.getPartial(i)
		if err != nil {
			return 0, common.Hash{}, nil, err
		}
		partials[i] = *partial
	}
	return size, root, partials, nil
}

type MerkleTreeNodeEvent struct {
	Level     uint64
	NumLeaves uint64
	Hash      common.Hash
}
