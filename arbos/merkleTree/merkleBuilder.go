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

type MerkleBuilder struct {
	backingStorage *storage.Storage
	size           uint64
	numPartials    uint64
}

func InitializeMerkleBuilder(sto *storage.Storage) {
	// no initialization needed
}

func OpenMerkleBuilder(sto *storage.Storage) *MerkleBuilder {
	size := sto.GetByInt64(0).Big().Uint64()
	numPartials := sto.GetByInt64(1).Big().Uint64()
	return &MerkleBuilder{sto, size, numPartials}
}

func (b *MerkleBuilder) Append(itemHash common.Hash) {
	b.size++
	b.backingStorage.SetByInt64(0, util.IntToHash(int64(b.size)))
	level := uint64(0)
	soFar := itemHash.Bytes()
	for {
		if level == b.numPartials {
			b.numPartials++
			b.backingStorage.SetByInt64(1, util.IntToHash(int64(b.numPartials)))
			b.backingStorage.SetByInt64(int64(2+level), common.BytesToHash(soFar))
			return
		}
		thisLevel := b.backingStorage.GetByInt64(int64(2 + level))
		if thisLevel == (common.Hash{}) {
			b.backingStorage.SetByInt64(int64(2+level), common.BytesToHash(soFar))
			return
		}
		soFar = crypto.Keccak256(thisLevel.Bytes(), soFar)
		b.backingStorage.SetByInt64(int64(2+level), common.Hash{})
		level += 1
	}
}

func (b *MerkleBuilder) Size() uint64 {
	return b.size
}

func (b *MerkleBuilder) Root() common.Hash {
	if b.size == 0 {
		return common.Hash{}
	}
	if b.size == 1 {
		return b.backingStorage.GetByInt64(2)
	}
	ret := make([]byte, 32)
	emptySoFar := true
	for i := uint64(0); i < b.numPartials; i++ {
		thisLevel := b.backingStorage.GetByInt64(int64(2 + i))
		if thisLevel == (common.Hash{}) {
			if !emptySoFar {
				ret = crypto.Keccak256(make([]byte, 32), ret)
			}
		} else {
			if emptySoFar {
				if i+1 == b.numPartials {
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
