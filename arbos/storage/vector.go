// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package storage

import (
	"encoding/binary"
	"errors"
)

const subStorageVectorLengthOffset uint64 = 0

// SubStorageVector is a storage space that contains a vector of sub-storages.
// It keeps track of the number of sub-storages and It is possible to push and pop them.
type SubStorageVector struct {
	storage *Storage
	length  StorageBackedUint64
}

// OpenSubStorageVector creates a SubStorageVector in given the root storage.
func OpenSubStorageVector(sto *Storage) *SubStorageVector {
	return &SubStorageVector{
		sto.WithoutCache(),
		sto.OpenStorageBackedUint64(subStorageVectorLengthOffset),
	}
}

// Length returns the number of sub-storages.
func (v *SubStorageVector) Length() (uint64, error) {
	length, err := v.length.Get()
	if err != nil {
		return 0, err
	}
	return length, err
}

// Push adds a new sub-storage at the end of the vector and return it.
func (v *SubStorageVector) Push() (*Storage, error) {
	length, err := v.length.Get()
	if err != nil {
		return nil, err
	}
	id := binary.BigEndian.AppendUint64(nil, length)
	subStorage := v.storage.OpenSubStorage(id)
	if err := v.length.Set(length + 1); err != nil {
		return nil, err
	}
	return subStorage, nil
}

// Pop removes the last sub-storage from the end of the vector and return it.
func (v *SubStorageVector) Pop() (*Storage, error) {
	length, err := v.length.Get()
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, errors.New("sub-storage vector: can't pop empty")
	}
	id := binary.BigEndian.AppendUint64(nil, length-1)
	subStorage := v.storage.OpenSubStorage(id)
	if err := v.length.Set(length - 1); err != nil {
		return nil, err
	}
	return subStorage, nil
}

// At returns the substorage at the given index.
// NOTE: This function does not verify out-of-bounds.
func (v *SubStorageVector) At(i uint64) *Storage {
	id := binary.BigEndian.AppendUint64(nil, i)
	subStorage := v.storage.OpenSubStorage(id)
	return subStorage
}
