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

type Storage struct {
	account common.Address
	db      vm.StateDB
	key     []byte
}

// Use a Geth database to create an evm key-value store
func NewGeth(statedb vm.StateDB) *Storage {
	account := common.HexToAddress("0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	statedb.SetNonce(account, 1) // setting the nonce ensures Geth won't treat ArbOS as empty
	return &Storage{
		account: account,
		db:      statedb,
		key:     []byte{},
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

// We map addresses using "pages" of 256 storage slots. We hash over the page number but not the offset within
//     a page, to preserve contiguity within a page. This will reduce cost if/when Ethereum switches to storage
//     representations that reward contiguity.
// Because page numbers are 248 bits, this gives us 124-bit security against collision attacks, which is good enough.
func mapAddress(storageKey []byte, key common.Hash) common.Hash {
	keyBytes := key.Bytes()
	return common.BytesToHash(
		append(
			crypto.Keccak256(storageKey, keyBytes[:common.HashLength-1])[:common.HashLength-1],
			keyBytes[common.HashLength-1],
		),
	)
}

func (store *Storage) Get(key common.Hash) common.Hash {
	return store.db.GetState(store.account, mapAddress(store.key, key))
}

func (store *Storage) GetUint64(key common.Hash) uint64 {
	return store.Get(key).Big().Uint64()
}

func (store *Storage) GetByUint64(key uint64) common.Hash {
	return store.Get(util.UintToHash(key))
}

func (store *Storage) GetUint64ByUint64(key uint64) uint64 {
	return store.Get(util.UintToHash(key)).Big().Uint64()
}

func (store *Storage) Set(key common.Hash, value common.Hash) {
	store.db.SetState(store.account, mapAddress(store.key, key), value)
}

func (store *Storage) SetByUint64(key uint64, value common.Hash) {
	store.Set(util.UintToHash(key), value)
}

func (store *Storage) SetUint64ByUint64(key uint64, value uint64) {
	store.Set(util.UintToHash(key), util.UintToHash(value))
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
		crypto.Keccak256(store.key, id),
	}
}

func (store *Storage) WriteBytes(b []byte) {
	store.SetUint64ByUint64(0, uint64(len(b)))
	offset := uint64(1)
	for len(b) >= 32 {
		store.SetByUint64(offset, common.BytesToHash(b[:32]))
		b = b[32:]
		offset++
	}
	store.SetByUint64(offset, common.BytesToHash(b))
}

func (store *Storage) GetBytes() []byte {
	bytesLeft := store.GetUint64ByUint64(0)
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

func (store *Storage) GetBytesSize() uint64 {
	return store.GetUint64ByUint64(0)
}

func (store *Storage) DeleteBytes() {
	bytesLeft := store.GetUint64ByUint64(0)
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
	cache   *int64
}

func (sto *Storage) OpenStorageBackedInt64(offset common.Hash) *StorageBackedInt64 {
	return &StorageBackedInt64{sto, offset, nil}
}

func (sbu *StorageBackedInt64) Get() int64 {
	if sbu.cache == nil {
		raw := sbu.storage.Get(sbu.offset).Big()
		if raw.Bit(255) != 0 {
			raw = new(big.Int).SetBit(raw, 255, 0)
			raw = new(big.Int).Neg(raw)
		}
		if !raw.IsInt64() {
			panic("expected int64 compatible value in storage")
		}
		i := raw.Int64()
		sbu.cache = &i
	}
	return *sbu.cache
}

func (sbu *StorageBackedInt64) Set(value int64) {
	i := value
	sbu.cache = &i
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
	cache   *uint64
}

func (sto *Storage) OpenStorageBackedUint64(offset common.Hash) *StorageBackedUint64 {
	return &StorageBackedUint64{sto, offset, nil}
}

func (sbu *StorageBackedUint64) Get() uint64 {
	if sbu.cache == nil {
		raw := sbu.storage.Get(sbu.offset).Big()
		if !raw.IsUint64() {
			panic("expected uint64 compatible value in storage")
		}
		i := raw.Uint64()
		sbu.cache = &i
	}
	return *sbu.cache
}

func (sbu *StorageBackedUint64) Set(value uint64) {
	i := value
	sbu.cache = &i
	bigValue := new(big.Int).SetUint64(value)
	sbu.storage.Set(sbu.offset, common.BigToHash(bigValue))
}
