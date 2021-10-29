//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/util"
)

// Represents a set of common.Hash objects
//   size is stored at position 0
//   members of the set are stored sequentially from 1 onward
type HashSet struct {
	backingStorage *Storage
	size           uint64
	cachedMembers map[common.Hash]struct{}
	byHash        *Storage
}

func InitializeHashSet(sto *Storage) {
	sto.SetByInt64(0, util.IntToHash(0))
}

func OpenHashSet(sto *Storage) *HashSet {
	return &HashSet{
		sto,
		sto.GetByInt64(0).Big().Uint64(),
		make(map[common.Hash]struct{}),
		sto.OpenSubStorage([]byte{0}),
	}
}

func (hset *HashSet) Size() uint64 {
	return hset.size
}

func (hset *HashSet) IsMember(h common.Hash) bool {
	if _, cached := hset.cachedMembers[h]; cached {
		return true
	}
	if hset.byHash.Get(h) != (common.Hash{}) {
		hset.cachedMembers[h] = struct{}{}
		return true
	}
	return false
}

func (hset *HashSet) AllMembers() []common.Hash {
	ret := make([]common.Hash, hset.size)
	for i := range ret {
		ret[i] = common.BytesToHash(hset.backingStorage.GetByInt64(int64(i + 1)).Bytes())
	}
	return ret
}

func (hset *HashSet) Add(h common.Hash) {
	if hset.IsMember(h) {
		return
	}
	slot := util.IntToHash(int64(1 + hset.size))
	addrAsHash := common.BytesToHash(h.Bytes())
	hset.byHash.Set(addrAsHash, slot)
	hset.backingStorage.Set(slot, addrAsHash)
	hset.size++
	hset.backingStorage.SetByInt64(0, util.IntToHash(int64(hset.size)))
}

func (hset *HashSet) Remove(h common.Hash) {
	slot := hset.byHash.Get(h).Big().Uint64()
	if slot == 0 {
		return
	}
	delete(hset.cachedMembers, h)
	hset.byHash.Set(h, common.Hash{})
	if slot < hset.size {
		hset.backingStorage.SetByInt64(int64(slot), hset.backingStorage.GetByInt64(int64(hset.size)))
	}
	hset.backingStorage.SetByInt64(int64(hset.size), common.Hash{})
	hset.size--
	hset.backingStorage.SetByInt64(0, util.IntToHash(int64(hset.size)))
}
