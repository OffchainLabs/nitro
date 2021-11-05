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

func (store *Storage) Get(key common.Hash) common.Hash {
	return store.db.GetState(store.account, crypto.Keccak256Hash(store.key, key.Bytes()))
}

func (store *Storage) GetByInt64(key int64) common.Hash {
	return store.Get(util.IntToHash(key))
}

func (store *Storage) Set(key common.Hash, value common.Hash) {
	store.db.SetState(store.account, crypto.Keccak256Hash(store.key, key.Bytes()), value)
}

func (store *Storage) SetByInt64(key int64, value common.Hash) {
	store.Set(util.IntToHash(key), value)
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
	store.SetByInt64(0, util.IntToHash(int64(len(b))))
	offset := int64(1)
	for len(b) >= 32 {
		store.SetByInt64(offset, common.BytesToHash(b[:32]))
		b = b[32:]
		offset++
	}
	store.SetByInt64(offset, common.BytesToHash(b))
}

func (store *Storage) GetBytes() []byte {
	bytesLeft := store.GetByInt64(0).Big().Int64()
	ret := []byte{}
	offset := int64(1)
	for bytesLeft >= 32 {
		ret = append(ret, store.GetByInt64(offset).Bytes()...)
		bytesLeft -= 32
		offset++
	}
	ret = append(ret, store.GetByInt64(offset).Bytes()[32-bytesLeft:]...)
	return ret
}

func (store *Storage) GetBytesSize() uint64 {
	return store.GetByInt64(0).Big().Uint64()
}

func (store *Storage) DeleteBytes() {
	bytesLeft := store.GetByInt64(0).Big().Int64()
	offset := int64(1)
	for bytesLeft > 0 {
		store.SetByInt64(offset, common.Hash{})
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

func (sbi *StorageBackedInt64) Get() int64 {
	if sbi.cache == nil {
		raw := sbi.storage.Get(sbi.offset).Big()
		if raw.Bit(255) != 0 {
			raw = new(big.Int).SetBit(raw, 255, 0)
			raw = new(big.Int).Neg(raw)
		}
		if !raw.IsInt64() {
			panic("expected int64 compatible value in storage")
		}
		i := raw.Int64()
		sbi.cache = &i
	}
	return *sbi.cache
}

func (sbi *StorageBackedInt64) Set(value int64) {
	i := value
	sbi.cache = &i
	var bigValue *big.Int
	if value >= 0 {
		bigValue = big.NewInt(value)
	} else {
		bigValue = new(big.Int).SetBit(big.NewInt(-value), 255, 1)
	}
	sbi.storage.Set(sbi.offset, common.BigToHash(bigValue))
}
