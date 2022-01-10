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
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/burn"
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
	burner     burn.Burner
}

var StorageReadCost = params.SloadGasEIP2200
var StorageWriteCost = params.SstoreSetGasEIP2200

// Use a Geth database to create an evm key-value store
func NewGeth(statedb vm.StateDB, burner burn.Burner) *Storage {
	account := common.HexToAddress("0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	statedb.SetNonce(account, 1) // setting the nonce ensures Geth won't treat ArbOS as empty
	return &Storage{
		account:    account,
		db:         statedb,
		storageKey: []byte{},
		burner:     burner,
	}
}

// Use Geth's memory-backed database to create an evm key-value store
func NewMemoryBacked(burner burn.Burner) *Storage {
	return NewGeth(NewMemoryBackedStateDB(), burner)
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

// We map addresses using "pages" of 256 storage slots. We hash over the page number but not the offset within
//     a page, to preserve contiguity within a page. This will reduce cost if/when Ethereum switches to storage
//     representations that reward contiguity.
// Because page numbers are 248 bits, this gives us 124-bit security against collision attacks, which is good enough.
func mapAddress(storageKey []byte, key common.Hash) common.Hash {
	keyBytes := key.Bytes()
	boundary := common.HashLength - 1
	return common.BytesToHash(
		append(
			crypto.Keccak256(storageKey, keyBytes[:boundary])[:boundary],
			keyBytes[boundary],
		),
	)
}

func (store *Storage) Get(key common.Hash) (common.Hash, error) {
	err := store.burner.Burn(StorageReadCost)
	if err != nil {
		return common.Hash{}, err
	}
	return store.db.GetState(store.account, mapAddress(store.storageKey, key)), nil
}

func (store *Storage) GetStorageSlot(key common.Hash) common.Hash {
	return mapAddress(store.storageKey, key)
}

func (store *Storage) GetUint64(key common.Hash) (uint64, error) {
	value, err := store.Get(key)
	if err != nil {
		return 0, err
	}
	return value.Big().Uint64(), nil
}

func (store *Storage) GetByUint64(key uint64) (common.Hash, error) {
	return store.Get(util.UintToHash(key))
}

func (store *Storage) GetUint64ByUint64(key uint64) (uint64, error) {
	return store.GetUint64(util.UintToHash(key))
}

func (store *Storage) Set(key common.Hash, value common.Hash) error {
	err := store.burner.Burn(StorageWriteCost)
	if err != nil {
		return err
	}
	store.db.SetState(store.account, mapAddress(store.storageKey, key), value)
	return nil
}

func (store *Storage) SetByUint64(key uint64, value common.Hash) error {
	return store.Set(util.UintToHash(key), value)
}

func (store *Storage) SetUint64ByUint64(key uint64, value uint64) error {
	return store.Set(util.UintToHash(key), util.UintToHash(value))
}

func (store *Storage) Clear(key common.Hash) error {
	return store.Set(key, common.Hash{})
}

func (store *Storage) ClearByUint64(key uint64) error {
	return store.Set(util.UintToHash(key), common.Hash{})
}

func (store *Storage) Swap(key common.Hash, newValue common.Hash) (common.Hash, error) {
	oldValue, err := store.Get(key)
	if err != nil {
		return common.Hash{}, err
	}
	return oldValue, store.Set(key, newValue)
}

func (store *Storage) OpenSubStorage(id []byte) *Storage {
	return &Storage{
		store.account,
		store.db,
		crypto.Keccak256(store.storageKey, id),
		store.burner,
	}
}

func (store *Storage) SetBytes(b []byte) error {
	err := store.ClearBytes()
	if err != nil {
		return err
	}
	err = store.SetUint64ByUint64(0, uint64(len(b)))
	if err != nil {
		return err
	}
	offset := uint64(1)
	for len(b) >= 32 {
		err = store.SetByUint64(offset, common.BytesToHash(b[:32]))
		if err != nil {
			return err
		}
		b = b[32:]
		offset++
	}
	return store.SetByUint64(offset, common.BytesToHash(b))
}

func (store *Storage) GetBytes() ([]byte, error) {
	bytesLeft, err := store.GetUint64ByUint64(0)
	if err != nil {
		return nil, err
	}
	ret := []byte{}
	offset := uint64(1)
	for bytesLeft >= 32 {
		next, err := store.GetByUint64(offset)
		if err != nil {
			return nil, err
		}
		ret = append(ret, next.Bytes()...)
		bytesLeft -= 32
		offset++
	}
	next, err := store.GetByUint64(offset)
	if err != nil {
		return nil, err
	}
	ret = append(ret, next.Bytes()[32-bytesLeft:]...)
	return ret, nil
}

func (store *Storage) GetBytesSize() (uint64, error) {
	return store.GetUint64ByUint64(0)
}

func (store *Storage) ClearBytes() error {
	bytesLeft, err := store.GetUint64ByUint64(0)
	if err != nil {
		return err
	}
	offset := uint64(1)
	for bytesLeft > 0 {
		err := store.SetByUint64(offset, common.Hash{})
		if err != nil {
			return err
		}
		offset++
		if bytesLeft < 32 {
			bytesLeft = 0
		} else {
			bytesLeft -= 32
		}
	}
	return store.SetByUint64(0, common.Hash{})
}

type StorageSlot struct {
	account common.Address
	db      vm.StateDB
	slot    common.Hash
	burner  burn.Burner
}

func (sto *Storage) NewSlot(offset uint64) StorageSlot {
	return StorageSlot{sto.account, sto.db, mapAddress(sto.storageKey, util.UintToHash(offset)), sto.burner}
}

func (ss *StorageSlot) Get() (common.Hash, error) {
	err := ss.burner.Burn(StorageReadCost)
	if err != nil {
		return common.Hash{}, err
	}
	return ss.db.GetState(ss.account, ss.slot), nil
}

func (ss *StorageSlot) Set(val common.Hash) error {
	err := ss.burner.Burn(StorageWriteCost)
	if err != nil {
		return err
	}
	ss.db.SetState(ss.account, ss.slot, val)
	return nil
}

// Implementation note for StorageBackedInt64: Conversions between big.Int and common.Hash give weird results
//     for negative values, so we cast to uint64 before writing to storage and cast back to int64 after reading.
//     Golang casting between uint64 and int64 doesn't change the data, it just reinterprets the same 8 bytes,
//     so this is a hacky but reliable way to store an 8-byte int64 in a common.Hash storage slot.
type StorageBackedInt64 struct {
	StorageSlot
}

func (sto *Storage) OpenStorageBackedInt64(offset uint64) StorageBackedInt64 {
	return StorageBackedInt64{sto.NewSlot(offset)}
}

func (sbu *StorageBackedInt64) Get() (int64, error) {
	raw, err := sbu.StorageSlot.Get()
	if err != nil {
		return 0, err
	}
	if !raw.Big().IsUint64() {
		panic("invalid value found in StorageBackedInt64 storage")
	}
	return int64(raw.Big().Uint64()), nil // see implementation note above
}

func (sbu *StorageBackedInt64) Set(value int64) error {
	return sbu.StorageSlot.Set(util.UintToHash(uint64(value))) // see implementation note above
}

type StorageBackedUint64 struct {
	StorageSlot
}

func (sto *Storage) OpenStorageBackedUint64(offset uint64) StorageBackedUint64 {
	return StorageBackedUint64{sto.NewSlot(offset)}
}

func (sbu *StorageBackedUint64) Get() (uint64, error) {
	raw, err := sbu.StorageSlot.Get()
	if err != nil {
		return 0, err
	}
	if !raw.Big().IsUint64() {
		panic("expected uint64 compatible value in storage")
	}
	return raw.Big().Uint64(), nil
}

func (sbu *StorageBackedUint64) Set(value uint64) error {
	bigValue := new(big.Int).SetUint64(value)
	return sbu.StorageSlot.Set(common.BigToHash(bigValue))
}

func (sbu *StorageBackedUint64) Increment() (uint64, error) {
	old, err := sbu.Get()
	if err != nil {
		return 0, err
	}
	if old+1 < old {
		panic("Overflow in StorageBackedUint64::Increment")
	}
	err = sbu.Set(old + 1)
	return old + 1, err
}

func (sbu *StorageBackedUint64) Decrement() (uint64, error) {
	old, err := sbu.Get()
	if err != nil {
		return 0, err
	}
	if old == 0 {
		panic("Underflow in StorageBackedUint64::Decrement")
	}
	err = sbu.Set(old - 1)
	return old - 1, err
}

type MemoryBackedUint64 struct {
	contents uint64
}

func (mbu *MemoryBackedUint64) Get() (uint64, error) {
	return mbu.contents, nil
}

func (mbu *MemoryBackedUint64) Set(val uint64) error {
	mbu.contents = val
	return nil
}

func (mbu *MemoryBackedUint64) Increment() (uint64, error) {
	old := mbu.contents
	if old+1 < old {
		panic("Overflow in MemoryBackedUint64::Increment")
	}
	return old + 1, mbu.Set(old + 1)
}

func (mbu *MemoryBackedUint64) Decrement() (uint64, error) {
	old := mbu.contents
	if old == 0 {
		panic("Underflow in MemoryBackedUint64::Decrement")
	}
	return old - 1, mbu.Set(old - 1)
}

type WrappedUint64 interface {
	Get() (uint64, error)
	Set(uint64) error
	Increment() (uint64, error)
	Decrement() (uint64, error)
}

type StorageBackedBigInt struct {
	StorageSlot
}

func (sto *Storage) OpenStorageBackedBigInt(offset uint64) StorageBackedBigInt {
	return StorageBackedBigInt{sto.NewSlot(offset)}
}

func (sbbi *StorageBackedBigInt) Get() (*big.Int, error) {
	value, err := sbbi.StorageSlot.Get()
	if err != nil {
		return nil, err
	}
	return value.Big(), nil
}

func (sbbi *StorageBackedBigInt) Set(val *big.Int) error {
	return sbbi.StorageSlot.Set(common.BigToHash(val))
}

type StorageBackedAddress struct {
	StorageSlot
}

func (sto *Storage) OpenStorageBackedAddress(offset uint64) StorageBackedAddress {
	return StorageBackedAddress{sto.NewSlot(offset)}
}

func (sba *StorageBackedAddress) Get() (common.Address, error) {
	value, err := sba.StorageSlot.Get()
	if err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(value.Bytes()), nil
}

func (sba *StorageBackedAddress) Set(val common.Address) error {
	return sba.StorageSlot.Set(common.BytesToHash(val.Bytes()))
}

type StorageBackedAddressOrNil struct {
	StorageSlot
}

var NilAddressRepresentation common.Hash

func init() {
	NilAddressRepresentation = common.BigToHash(new(big.Int).Lsh(big.NewInt(1), 255))
}

func (sto *Storage) OpenStorageBackedAddressOrNil(offset uint64) StorageBackedAddressOrNil {
	return StorageBackedAddressOrNil{sto.NewSlot(offset)}
}

func (sba *StorageBackedAddressOrNil) Get() (*common.Address, error) {
	asHash, err := sba.StorageSlot.Get()
	if err != nil {
		return nil, err
	}
	if asHash == NilAddressRepresentation {
		return nil, nil
	} else {
		ret := common.BytesToAddress(asHash.Bytes())
		return &ret, nil
	}
}

func (sba *StorageBackedAddressOrNil) Set(val *common.Address) error {
	if val == nil {
		return sba.StorageSlot.Set(NilAddressRepresentation)
	} else {
		return sba.StorageSlot.Set(common.BytesToHash(val.Bytes()))
	}
}

type StorageBackedBytes struct {
	Storage
}

func (sto *Storage) OpenStorageBackedBytes(id []byte) StorageBackedBytes {
	return StorageBackedBytes{
		*sto.OpenSubStorage(id),
	}
}

func (sbb *StorageBackedBytes) Get() ([]byte, error) {
	return sbb.Storage.GetBytes()
}

func (sbb *StorageBackedBytes) Set(val []byte) error {
	return sbb.Storage.SetBytes(val)
}

func (sbb *StorageBackedBytes) Clear() error {
	return sbb.Storage.ClearBytes()
}

func (sbb *StorageBackedBytes) Size() (uint64, error) {
	return sbb.Storage.GetBytesSize()
}
