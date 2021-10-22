//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleTree

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"io"
)

type MerkleAccumulator struct {
	backingStorage *storage.Storage
	size           uint64
	numPartials    uint64
}

func InitializeMerkleAccumulator(sto *storage.Storage) {
	// no initialization needed
}

func OpenMerkleAccumulator(sto *storage.Storage) *MerkleAccumulator {
	size := sto.GetByInt64(0).Big().Uint64()
	numPartials := sto.GetByInt64(1).Big().Uint64()
	return &MerkleAccumulator{sto, size, numPartials}
}

func (acc *MerkleAccumulator) Append(itemHash common.Hash) {
	acc.size++
	acc.backingStorage.SetByInt64(0, util.IntToHash(int64(acc.size)))
	level := uint64(0)
	soFar := itemHash.Bytes()
	for {
		if level == acc.numPartials {
			acc.numPartials++
			acc.backingStorage.SetByInt64(1, util.IntToHash(int64(acc.numPartials)))
			acc.backingStorage.SetByInt64(int64(2+level), common.BytesToHash(soFar))
			return
		}
		thisLevel := acc.backingStorage.GetByInt64(int64(2 + level))
		if thisLevel == (common.Hash{}) {
			acc.backingStorage.SetByInt64(int64(2+level), common.BytesToHash(soFar))
			return
		}
		soFar = crypto.Keccak256(thisLevel.Bytes(), soFar)
		acc.backingStorage.SetByInt64(int64(2+level), common.Hash{})
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
		return acc.backingStorage.GetByInt64(2)
	}
	ret := make([]byte, 32)
	emptySoFar := true
	for i := uint64(0); i < acc.numPartials; i++ {
		thisLevel := acc.backingStorage.GetByInt64(int64(2 + i))
		if thisLevel == (common.Hash{}) {
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

func (acc *MerkleAccumulator) Serialize(wr io.Writer) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], acc.size)
	if _, err := wr.Write(buf[:]); err != nil {
		return err
	}
	for i:=uint64(0); i<acc.size; i++ {
		if _, err := wr.Write(acc.backingStorage.GetByInt64(int64(i + 2)).Bytes()); err != nil {
			return err
		}
	}
	return nil
}
