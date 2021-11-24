//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleAccumulator

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
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
	size := sto.GetByUint64(0).Big().Uint64()
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

func (acc *MerkleAccumulator) setSize(size uint64) {
	acc.size = size
	if acc.backingStorage != nil {
		acc.backingStorage.SetByUint64(0, util.IntToHash(int64(size)))
	}
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

func (acc *MerkleAccumulator) setPartial(level uint64, val *common.Hash, adjustSize bool) {
	if level == acc.numPartials {
		acc.numPartials++
		acc.partials = append(acc.partials, val)
		if adjustSize {
			acc.setSize(acc.size + (1 << level))
		}
	} else {
		if adjustSize {
			if *acc.partials[level] == (common.Hash{}) {
				if *val != (common.Hash{}) {
					acc.setSize(acc.size + (1 << level))
				}
			} else {
				if *val == (common.Hash{}) {
					acc.setSize(acc.size - (1 << level))
				}
			}
		}
		acc.partials[level] = val
	}
	if acc.backingStorage != nil {
		acc.backingStorage.SetByUint64(2+level, *val)
	}
}

func (acc *MerkleAccumulator) Append(itemHash common.Hash) []MerkleTreeNodeEvent {
	events := []MerkleTreeNodeEvent{}

	acc.setSize(acc.size + 1)

	level := uint64(0)
	soFar := itemHash.Bytes()
	for {
		if level == acc.numPartials {
			h := common.BytesToHash(soFar)
			acc.setPartial(level, &h, false)
			return events
		}
		thisLevel := acc.getPartial(level)
		if *thisLevel == (common.Hash{}) {
			h := common.BytesToHash(soFar)
			acc.setPartial(level, &h, false)
			return events
		}
		soFar = crypto.Keccak256(thisLevel.Bytes(), soFar)
		h := common.Hash{}
		acc.setPartial(level, &h, false)
		level += 1
		events = append(events, MerkleTreeNodeEvent{level, acc.size - 1, common.BytesToHash(soFar)})
	}
}

var ErrInvalidLevel = errors.New("invalid partial level")

func (acc *MerkleAccumulator) AppendPartial(level uint64, val *common.Hash) error {
	for i := uint64(0); i < level && i < acc.numPartials; i++ {
		res := acc.getPartial(i)
		if res != nil && *res != (common.Hash{}) {
			return ErrInvalidLevel
		}
	}
	for acc.numPartials < level {
		acc.setPartial(acc.numPartials, &common.Hash{}, false)
	}
	soFar := val.Bytes()
	for {
		if level == acc.numPartials {
			h := common.BytesToHash(soFar)
			acc.setPartial(level, &h, true)
			return nil
		}
		thisLevel := acc.getPartial(level)
		if *thisLevel == (common.Hash{}) {
			h := common.BytesToHash(soFar)
			acc.setPartial(level, &h, true)
			return nil
		}
		soFar = crypto.Keccak256(thisLevel.Bytes(), soFar)
		h := common.Hash{}
		acc.setPartial(level, &h, true)
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

type LevelAndHash struct {
	Level uint64
	Hash  common.Hash
}

func (before *MerkleAccumulator) VerifyConsistencyProof(afterHash common.Hash, proof []LevelAndHash) bool {
	working := before.NonPersistentClone()
	for _, partial := range proof {
		if working.AppendPartial(partial.Level, &partial.Hash) != nil {
			return false
		}
	}
	return working.Root() == afterHash
}

type ConciseConsistencyProof struct {
	BeforeSize uint64
	AfterSize  uint64
	BeforeHash common.Hash
	AfterHash  common.Hash
	Proof      []common.Hash
}

func (ccp *ConciseConsistencyProof) Verify() bool {
	if ccp.BeforeSize > ccp.AfterSize {
		return false
	}
	if ccp.BeforeSize == ccp.AfterSize {
		return ccp.BeforeHash == ccp.AfterHash && len(ccp.Proof) == 0
	}

	// build the before MerkleAccumulator and verify its hash
	beforeSize := ccp.BeforeSize
	acc := NewNonpersistentMerkleAccumulator()
	proof := ccp.Proof
	for beforeSize > 0 {
		if len(proof) == 0 {
			return false
		}
		level := util_math.Log2floor(beforeSize)
		if err := acc.AppendPartial(level, &proof[0]); err != nil {
			// OK to panic here because the error should be impossible, can only happen if there is a bug in this function
			panic("error building acc in ConciseConsistencyProof::Verify")
		}
		beforeSize -= 1 << level
		proof = proof[1:]
	}
	if acc.Root() != ccp.BeforeHash {
		return false
	}

	// apply a series of partials, in two passes, to transition to the after state, then verify its hash
	// upward pass
	switchoverLevel := util_math.Log2floor(ccp.BeforeSize ^ ccp.AfterSize)
	for level := uint64(0); level < switchoverLevel; level++ {
		if acc.size&(1<<level) != 0 {
			if len(proof) == 0 {
				return false
			}
			if err := acc.AppendPartial(level, &proof[0]); err != nil {
				// OK to panic here because the error should be impossible, can only happen if there is a bug in this function
				panic("error in upward pass in ConciseConsistencyProof::Verify")
			}
			proof = proof[1:]
		}
	}

	// downward pass
	for acc.size < ccp.AfterSize {
		level := util_math.Log2floor(ccp.AfterSize - acc.size)
		if len(proof) == 0 {
			return false
		}
		if err := acc.AppendPartial(uint64(level), &proof[0]); err != nil {
			// OK to panic here because the error should be impossible, can only happen if there is a bug in this function
			panic("error in downward pass in ConciseConsistencyProof::Verify")
		}
		proof = proof[1:]
	}

	return acc.Root() == ccp.AfterHash && len(proof) == 0
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
