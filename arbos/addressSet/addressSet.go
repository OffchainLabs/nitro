//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package addressSet

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
)

// Represents a set of addresses
//   size is stored at position 0
//   members of the set are stored sequentially from 1 onward
type AddressSet struct {
	backingStorage *storage.Storage
	size           storage.StorageBackedUint64
	byAddress      *storage.Storage
}

func Initialize(sto *storage.Storage) {
	sto.SetUint64ByUint64(0, 0)
}

func OpenAddressSet(sto *storage.Storage) *AddressSet {
	return &AddressSet{
		sto,
		sto.NewStorageBackedUint64(0),
		sto.OpenSubStorage([]byte{0}),
	}
}

func (aset *AddressSet) Size() uint64 {
	return aset.size.Get()
}

func (aset *AddressSet) IsMember(addr common.Address) bool {
	return aset.byAddress.Get(common.BytesToHash(addr.Bytes())) != (common.Hash{})
}

func (aset *AddressSet) AllMembers() []common.Address {
	ret := make([]common.Address, aset.size.Get())
	for i := range ret {
		ret[i] = common.BytesToAddress(aset.backingStorage.GetByUint64(uint64(i + 1)).Bytes())
	}
	return ret
}

func (aset *AddressSet) Add(addr common.Address) {
	if aset.IsMember(addr) {
		return
	}
	slot := util.UintToHash(1 + aset.size.Get())
	addrAsHash := common.BytesToHash(addr.Bytes())
	aset.byAddress.Set(addrAsHash, slot)
	aset.backingStorage.Set(slot, addrAsHash)
	aset.size.Set(aset.size.Get() + 1)
}

func (aset *AddressSet) Remove(addr common.Address) {
	addrAsHash := common.BytesToHash(addr.Bytes())
	slot := aset.byAddress.GetUint64(addrAsHash)
	if slot == 0 {
		return
	}
	aset.byAddress.Set(addrAsHash, common.Hash{})
	sz := aset.size.Get()
	if slot < sz {
		aset.backingStorage.SetByUint64(slot, aset.backingStorage.GetByUint64(sz))
	}
	aset.backingStorage.SetByUint64(sz, common.Hash{})
	aset.size.Set(sz - 1)
}
