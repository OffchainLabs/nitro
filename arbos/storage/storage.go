//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package storage

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/util"
)

// Storage allows ArbOS to store data persistently in the Ethereum-compatible stateDB. This is represented in
// the stateDB as the storage of a fictional Ethereum account at address 0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF.
//
// The storage is logically a tree of storage spaces which can be nested hierarchically, with each storage space
// containing a key-value store with 256-bit keys and values. Uninitialized storage spaces and uninitialized keys
// within initialized storage spaces are deemed to be filled with zeroes (consistent with the behavior of Ethereum
// account storage). Logically, when a chain is launched, all possible storage spaces and all possible keys within
// them exist and contain zeroes.
//
// A storage space (represented by a Storage object) has a byte-slice storageKey which distinguishes it from other
// storage spaces. The root Storage has its storageKey as the empty string. A parent storage space can contain children,
// each with a distinct name. The storageKey of a child is keccak256(parent.storageKey, name). Note that two spaces
// cannot have the same storageKey because that would imply a collision in keccak256.
//
// The contents of all storage spaces are stored in a single, flat key-value store that is implemented as the storage
// of the fictional Ethereum account. The contents of key, within a storage space with storageKey, are stored
// at location keccak256(storageKey, key) in the flat KVS. Two slots, whether in the same or different storage spaces,
// cannot occupy the same location because that would imply a collision in keccak256.

type Storage struct {
	account    common.Address
	db         vm.StateDB
	storageKey []byte
}

// Use a Geth database to create an evm key-value store
func NewGeth(statedb vm.StateDB) *Storage {
	account := common.HexToAddress("0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	statedb.SetNonce(account, 1) // setting the nonce ensures Geth won't treat ArbOS as empty
	return &Storage{
		account:    account,
		db:         statedb,
		storageKey: []byte{},
	}
}

// Use Geth's memory-backed database to create an evm key-value store
func NewMemoryBacked() *Storage {
	return NewGeth(NewMemoryBackedStateDB())
}

// Use Geth's memory-backed database to create a statedb
func NewMemoryBackedStateDB() vm.StateDB {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		panic("failed to init empty statedb")
	}
	return statedb
}

func (store *Storage) Get(key common.Hash) common.Hash {
	return store.db.GetState(store.account, crypto.Keccak256Hash(store.storageKey, key.Bytes()))
}

func (store *Storage) GetByUint64(key uint64) common.Hash {
	return store.Get(util.UintToHash(key))
}

func (store *Storage) Set(key common.Hash, value common.Hash) {
	store.db.SetState(store.account, crypto.Keccak256Hash(store.storageKey, key.Bytes()), value)
}

func (store *Storage) SetByUint64(key uint64, value common.Hash) {
	store.Set(util.UintToHash(key), value)
}

func (store *Storage) Swap(key common.Hash, newValue common.Hash) common.Hash {
	oldValue := store.Get(key)
	store.Set(key, newValue)
	return oldValue
}

func (store *Storage) OpenSubStorage(id []byte) *Storage {
	return &Storage{
		store.account,
		store.db,
		crypto.Keccak256(store.storageKey, id),
	}
}

func (store *Storage) WriteBytes(b []byte) {
	store.SetByUint64(0, util.UintToHash(uint64(len(b))))
	offset := uint64(1)
	for len(b) >= 32 {
		store.SetByUint64(offset, common.BytesToHash(b[:32]))
		b = b[32:]
		offset++
	}
	store.SetByUint64(offset, common.BytesToHash(b))
}

func (store *Storage) GetBytes() []byte {
	bytesLeft := store.GetByUint64(0).Big().Uint64()
	ret := []byte{}
	offset := uint64(1)
	for bytesLeft >= 32 {
		ret = append(ret, store.GetByUint64(offset).Bytes()...)
		bytesLeft -= 32
		offset++
	}
	ret = append(ret, store.GetByUint64(offset).Bytes()[32-bytesLeft:]...)
	return ret
}

func (store *Storage) DeleteBytes() {
	bytesLeft := store.GetByUint64(0).Big().Uint64()
	offset := uint64(1)
	for bytesLeft > 0 {
		store.SetByUint64(offset, common.Hash{})
		offset++
		if bytesLeft < 32 {
			bytesLeft = 0
		} else {
			bytesLeft -= 32
		}
	}
}

// StorageBackedInt64 exists because the conversions between common.Hash and big.Int that is provided by
//     go-ethereum don't handle negative values cleanly.  This class hides that complexity.
type StorageBackedInt64 struct {
	storage *Storage
	offset  common.Hash
}

func (sto *Storage) OpenStorageBackedInt64(offset common.Hash) *StorageBackedInt64 {
	return &StorageBackedInt64{sto, offset}
}

func (sbu *StorageBackedInt64) Get() int64 {
	raw := sbu.storage.Get(sbu.offset).Big()
	if raw.Bit(255) != 0 {
		raw = new(big.Int).SetBit(raw, 255, 0)
		raw = new(big.Int).Neg(raw)
	}
	if !raw.IsInt64() {
		panic("expected int64 compatible value in storage")
	}
	return raw.Int64()
}

func (sbu *StorageBackedInt64) Set(value int64) {
	var bigValue *big.Int
	if value >= 0 {
		bigValue = big.NewInt(value)
	} else {
		bigValue = new(big.Int).SetBit(big.NewInt(-value), 255, 1)
	}
	sbu.storage.Set(sbu.offset, common.BigToHash(bigValue))
}

type StorageBackedUint64 struct {
	storage *Storage
	offset  common.Hash
}

func (sto *Storage) OpenStorageBackedUint64(offset common.Hash) *StorageBackedUint64 {
	return &StorageBackedUint64{sto, offset}
}

func (sbu *StorageBackedUint64) Get() uint64 {
	raw := sbu.storage.Get(sbu.offset).Big()
	if !raw.IsUint64() {
		panic("expected uint64 compatible value in storage")
	}
	return raw.Uint64()
}

func (sbu *StorageBackedUint64) Set(value uint64) {
	bigValue := new(big.Int).SetUint64(value)
	sbu.storage.Set(sbu.offset, common.BigToHash(bigValue))
}

type MemoryBackedUint64 struct {
	contents uint64
}

func (mbu *MemoryBackedUint64) Get() uint64 {
	return mbu.contents
}

func (mbu *MemoryBackedUint64) Set(val uint64) {
	mbu.contents = val
}

type WrappedUint64 interface {
	Get() uint64
	Set(uint64)
}

type StorageBackedBigInt struct {
	storage *Storage
	offset  common.Hash
}

func (sto *Storage) OpenStorageBackedBigInt(offset common.Hash) *StorageBackedBigInt {
	return &StorageBackedBigInt{sto, offset}
}

func (sbbi *StorageBackedBigInt) Get() *big.Int {
	return sbbi.storage.Get(sbbi.offset).Big()
}

func (sbbi *StorageBackedBigInt) Set(val *big.Int) {
	sbbi.storage.Set(sbbi.offset, common.BigToHash(val))
}

type StorageBackedAddress struct {
	storage *Storage
	offset  common.Hash
}

func (sto *Storage) OpenStorageBackedAddress(offset common.Hash) *StorageBackedAddress {
	return &StorageBackedAddress{sto, offset}
}

func (sba *StorageBackedAddress) Get() common.Address {
	return common.BytesToAddress(sba.storage.Get(sba.offset).Bytes())
}

func (sba *StorageBackedAddress) Set(val common.Address) {
	sba.storage.Set(sba.offset, common.BytesToHash(val.Bytes()))
}
