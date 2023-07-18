// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package addressSet

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
)

// AddressSet represents a set of addresses
// size is stored at position 0
// members of the set are stored sequentially from 1 onward
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
	sba := aset.backingStorage.OpenStorageBackedAddressOrNil(1)
	addr, err := sba.Get()
	return addr, err
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

func (aset *AddressSet) AllMembers(maxNumToReturn uint64) ([]common.Address, error) {
	size, err := aset.size.Get()
	if err != nil {
		return nil, err
	}
	if size > maxNumToReturn {
		size = maxNumToReturn
	}
	ret := make([]common.Address, size)
	for i := range ret {
		sba := aset.backingStorage.OpenStorageBackedAddress(uint64(i + 1))
		ret[i], err = sba.Get()
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (aset *AddressSet) getMapping(addrHash common.Hash) (common.Hash, uint64, bool, error) {
	index, err := aset.byAddress.GetUint64(addrHash)
	if err != nil || index == 0 {
		return common.Hash{}, index, false, errors.New("RectifyMapping: Address is not an owner")
	}
	atIndex, err := aset.backingStorage.GetByUint64(index)
	if (err != nil || atIndex == common.Hash{}) {
		return atIndex, index, true, errors.New("RectifyMapping: Invalid mapping")
	}
	return atIndex, index, true, nil
}

func (aset *AddressSet) syncListAndMap(addr common.Address) error {
	// Iterate through the list and replace the value at first index with incorrect mapping occurance.
	addrHash := common.BytesToHash(addr.Bytes())
	size, err := aset.size.Get()
	if err != nil || size == 0 {
		return err
	}
	for i := uint64(1); i <= size; i++ {
		tmpAddrHash, err := aset.backingStorage.GetByUint64(i)
		if err != nil {
			return err
		}
		tmpAddrHashInList, tmpIndex, _, err := aset.getMapping(tmpAddrHash)
		if err != nil || tmpAddrHash != tmpAddrHashInList || tmpIndex != i {
			err = aset.backingStorage.SetByUint64(i, addrHash)
			if err != nil {
				return err
			}
			err = aset.byAddress.Set(addrHash, util.UintToHash(i))
			return err
		}
	}
	// List is correctly aligned with map, only way to sync owner is to add a new entry to list.
	err = aset.byAddress.Clear(addrHash)
	if err != nil {
		return err
	}
	err = aset.Add(addr)
	return err
}

func (aset *AddressSet) RectifyMapping(addr common.Address) error {
	addrHash := common.BytesToHash(addr.Bytes())
	addrHashInList, index, inMap, err := aset.getMapping(addrHash)
	// Key is not found in the Map.
	if !inMap {
		return err
	}
	// Map has incorrect list index for the address.
	if err != nil {
		err = aset.syncListAndMap(addr)
		return err
	}
	// Map and list are correctly synced for this address.
	if addrHash == addrHashInList {
		return nil
	}
	// Not a correct mapping Or is a correct mapping and no collision, list value at 'index' can be edited.
	tmpAddrHashInList, tmpIndex, _, err := aset.getMapping(addrHashInList)
	if err != nil || addrHashInList != tmpAddrHashInList || index != tmpIndex {
		err = aset.backingStorage.SetByUint64(index, addrHash)
		return err
	}
	// Both keys addrHash and addrHashInList point to the same index and addrHashInList is correctly synced.
	err = aset.syncListAndMap(addr)
	return err
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
	sba := aset.backingStorage.OpenStorageBackedAddress(1 + size)
	slot := util.UintToHash(1 + size)
	addrAsHash := common.BytesToHash(addr.Bytes())
	err = aset.byAddress.Set(addrAsHash, slot)
	if err != nil {
		return err
	}
	sba = aset.backingStorage.OpenStorageBackedAddress(1 + size)
	err = sba.Set(addr)
	if err != nil {
		return err
	}
	_, err = aset.size.Increment()
	return err
}

func (aset *AddressSet) Remove(addr common.Address, arbosVersion uint64) error {
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
		if arbosVersion >= 11 {
			err = aset.byAddress.Set(atSize, util.UintToHash(slot))
			if err != nil {
				return err
			}
		}
	}
	err = aset.backingStorage.ClearByUint64(size)
	if err != nil {
		return err
	}
	_, err = aset.size.Decrement()
	return err
}
