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

func Initialize(sto *storage.Storage) error {
	return sto.SetUint64ByUint64(0, 0)
}

func OpenAddressSet(sto *storage.Storage) *AddressSet {
	return &AddressSet{
		sto,
		sto.OpenStorageBackedUint64(0),
		sto.OpenSubStorage([]byte{0}),
	}
}

func (aset *AddressSet) Size() (uint64, error) {
	return aset.size.Get()
}

func (aset *AddressSet) IsMember(addr common.Address) (bool, error) {
	value, err := aset.byAddress.Get(util.AddressToHash(addr))
	return value != (common.Hash{}), err
}

func (aset *AddressSet) GetAnyMember() (*common.Address, error) {
	size, err := aset.size.Get()
	if err != nil || size == 0 {
		return nil, err
	}
	addrAsHash, err := aset.backingStorage.GetByUint64(1)
	if err != nil {
		return nil, err
	}
	addr := common.BytesToAddress(addrAsHash.Bytes())
	return &addr, nil
}

func (aset *AddressSet) Clear() error {
	size, err := aset.size.Get()
	if err != nil || size == 0 {
		return err
	}
	for i := uint64(1); i <= size; i++ {
		contents, _ := aset.backingStorage.GetByUint64(i)
		_ = aset.backingStorage.ClearByUint64(i)
		err = aset.byAddress.Clear(contents)
		if err != nil {
			return err
		}
	}
	return aset.size.Clear()
}

func (aset *AddressSet) AllMembers() ([]common.Address, error) {
	size, err := aset.size.Get()
	if err != nil {
		return nil, err
	}
	ret := make([]common.Address, size)
	for i := range ret {
		bytes, err := aset.backingStorage.GetByUint64(uint64(i + 1))
		if err != nil {
			return nil, err
		}
		ret[i] = common.BytesToAddress(bytes.Bytes())
	}
	return ret, nil
}

func (aset *AddressSet) Add(addr common.Address) error {
	present, err := aset.IsMember(addr)
	if present || err != nil {
		return err
	}
	size, err := aset.size.Get()
	if err != nil {
		return err
	}
	slot := util.UintToHash(1 + size)
	addrAsHash := common.BytesToHash(addr.Bytes())
	err = aset.byAddress.Set(addrAsHash, slot)
	if err != nil {
		return err
	}
	err = aset.backingStorage.Set(slot, addrAsHash)
	if err != nil {
		return err
	}
	_, err = aset.size.Increment()
	return err
}

func (aset *AddressSet) Remove(addr common.Address) error {
	addrAsHash := common.BytesToHash(addr.Bytes())
	slot, err := aset.byAddress.GetUint64(addrAsHash)
	if slot == 0 || err != nil {
		return err
	}
	err = aset.byAddress.Clear(addrAsHash)
	if err != nil {
		return err
	}
	size, err := aset.size.Get()
	if err != nil {
		return err
	}
	if slot < size {
		atSize, err := aset.backingStorage.GetByUint64(size)
		if err != nil {
			return err
		}
		err = aset.backingStorage.SetByUint64(slot, atSize)
		if err != nil {
			return err
		}
	}
	err = aset.backingStorage.ClearByUint64(size)
	if err != nil {
		return err
	}
	_, err = aset.size.Decrement()
	return err
}
