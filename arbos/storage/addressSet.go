//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/util"
)

// Represents a set of addresses
//   size is stored at position 0
//   members of the set are stored sequentially from 1 onward
type AddressSet struct {
	backingStorage *Storage
	size           uint64
	cachedMembers  map[common.Address]struct{}
	byAddress      *Storage
}

func InitializeAddressSet(sto *Storage) {
	sto.SetByInt64(0, util.IntToHash(0))
}

func OpenAddressSet(sto *Storage) *AddressSet {
	return &AddressSet{
		sto,
		sto.GetByInt64(0).Big().Uint64(),
		make(map[common.Address]struct{}),
		sto.OpenSubStorage([]byte{0}),
	}
}

func (aset *AddressSet) Size() uint64 {
	return aset.size
}

func (aset *AddressSet) IsMember(addr common.Address) bool {
	if _, cached := aset.cachedMembers[addr]; cached {
		return true
	}
	if aset.byAddress.Get(common.BytesToHash(addr.Bytes())) != (common.Hash{}) {
		aset.cachedMembers[addr] = struct{}{}
		return true
	}
	return false
}

func (aset *AddressSet) AllMembers() []common.Address {
	ret := make([]common.Address, aset.size)
	for i := range ret {
		ret[i] = common.BytesToAddress(aset.backingStorage.GetByInt64(int64(i + 1)).Bytes())
	}
	return ret
}

func (aset *AddressSet) Add(addr common.Address) {
	if aset.IsMember(addr) {
		return
	}
	slot := util.IntToHash(int64(1 + aset.size))
	addrAsHash := common.BytesToHash(addr.Bytes())
	aset.byAddress.Set(addrAsHash, slot)
	aset.backingStorage.Set(slot, addrAsHash)
	aset.size++
	aset.backingStorage.SetByInt64(0, util.IntToHash(int64(aset.size)))
}

func (aset *AddressSet) Remove(addr common.Address) {
	addrAsHash := common.BytesToHash(addr.Bytes())
	slot := aset.byAddress.Get(addrAsHash).Big().Uint64()
	if slot == 0 {
		return
	}
	delete(aset.cachedMembers, addr)
	aset.byAddress.Set(addrAsHash, common.Hash{})
	if slot < aset.size {
		aset.backingStorage.SetByInt64(int64(slot), aset.backingStorage.GetByInt64(int64(aset.size)))
	}
	aset.backingStorage.SetByInt64(int64(aset.size), common.Hash{})
	aset.size--
	aset.backingStorage.SetByInt64(0, util.IntToHash(int64(aset.size)))
}
