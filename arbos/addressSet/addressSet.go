// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package addressSet

// TODO lowercase this package name

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
		backingStorage: sto.WithoutCache(),
		size:           sto.OpenStorageBackedUint64(0),
		byAddress:      sto.OpenSubStorage([]byte{0}),
	}
}

func (as *AddressSet) Size() (uint64, error) {
	return as.size.Get()
}

func (as *AddressSet) IsMember(addr common.Address) (bool, error) {
	value, err := as.byAddress.Get(util.AddressToHash(addr))
	return value != (common.Hash{}), err
}

func (as *AddressSet) GetAnyMember() (*common.Address, error) {
	size, err := as.size.Get()
	if err != nil || size == 0 {
		return nil, err
	}
	sba := as.backingStorage.OpenStorageBackedAddressOrNil(1)
	addr, err := sba.Get()
	return addr, err
}

func (as *AddressSet) Clear() error {
	size, err := as.size.Get()
	if err != nil || size == 0 {
		return err
	}
	for i := uint64(1); i <= size; i++ {
		contents, _ := as.backingStorage.GetByUint64(i)
		_ = as.backingStorage.ClearByUint64(i)
		err = as.byAddress.Clear(contents)
		if err != nil {
			return err
		}
	}
	return as.size.Clear()
}

func (as *AddressSet) AllMembers(maxNumToReturn uint64) ([]common.Address, error) {
	size, err := as.size.Get()
	if err != nil {
		return nil, err
	}
	if size > maxNumToReturn {
		size = maxNumToReturn
	}
	ret := make([]common.Address, size)
	for i := range ret {
		// #nosec G115
		sba := as.backingStorage.OpenStorageBackedAddress(uint64(i + 1))
		ret[i], err = sba.Get()
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (as *AddressSet) ClearList() error {
	size, err := as.size.Get()
	if err != nil || size == 0 {
		return err
	}
	for i := uint64(1); i <= size; i++ {
		err = as.backingStorage.ClearByUint64(i)
		if err != nil {
			return err
		}
	}
	return as.size.Clear()
}

func (as *AddressSet) RectifyMapping(addr common.Address) error {
	isOwner, err := as.IsMember(addr)
	if !isOwner || err != nil {
		return errors.New("RectifyMapping: Address is not an owner")
	}

	// If the mapping is correct, RectifyMapping shouldn't do anything
	// Additional safety check to avoid corruption of mapping after the initial fix
	addrAsHash := common.BytesToHash(addr.Bytes())
	slot, err := as.byAddress.GetUint64(addrAsHash)
	if err != nil {
		return err
	}
	atSlot, err := as.backingStorage.GetByUint64(slot)
	if err != nil {
		return err
	}
	size, err := as.size.Get()
	if err != nil {
		return err
	}
	if atSlot == addrAsHash && slot <= size {
		return errors.New("RectifyMapping: Owner address is correctly mapped")
	}

	// Remove the owner from map and add them as a new owner
	err = as.byAddress.Clear(addrAsHash)
	if err != nil {
		return err
	}

	return as.Add(addr)
}

func (as *AddressSet) Add(addr common.Address) error {
	present, err := as.IsMember(addr)
	if present || err != nil {
		return err
	}
	size, err := as.size.Get()
	if err != nil {
		return err
	}
	slot := util.UintToHash(1 + size)
	addrAsHash := common.BytesToHash(addr.Bytes())
	err = as.byAddress.Set(addrAsHash, slot)
	if err != nil {
		return err
	}
	sba := as.backingStorage.OpenStorageBackedAddress(1 + size)
	err = sba.Set(addr)
	if err != nil {
		return err
	}
	_, err = as.size.Increment()
	return err
}

func (as *AddressSet) Remove(addr common.Address, arbosVersion uint64) error {
	addrAsHash := common.BytesToHash(addr.Bytes())
	slot, err := as.byAddress.GetUint64(addrAsHash)
	if slot == 0 || err != nil {
		return err
	}
	err = as.byAddress.Clear(addrAsHash)
	if err != nil {
		return err
	}
	size, err := as.size.Get()
	if err != nil {
		return err
	}
	if slot < size {
		atSize, err := as.backingStorage.GetByUint64(size)
		if err != nil {
			return err
		}
		err = as.backingStorage.SetByUint64(slot, atSize)
		if err != nil {
			return err
		}
		if arbosVersion >= 11 {
			err = as.byAddress.Set(atSize, util.UintToHash(slot))
			if err != nil {
				return err
			}
		}
	}
	err = as.backingStorage.ClearByUint64(size)
	if err != nil {
		return err
	}
	_, err = as.size.Decrement()
	return err
}
