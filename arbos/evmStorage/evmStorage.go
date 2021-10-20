//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package evmStorage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
)

type T interface {
	Get(key common.Hash) common.Hash
	Set(key common.Hash, value common.Hash)
	Swap(key common.Hash, value common.Hash) common.Hash
}

type gethEvmStorage struct {
	account common.Address
	db      vm.StateDB
}

// Use a Geth database to create an evm key-value store
func NewGeth(statedb vm.StateDB) T {
	return &gethEvmStorage{
		account: common.HexToAddress("0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		db:      statedb,
	}
}

// Use Geth's memory-backed database to create an evm key-value store
func NewMemoryBacked() T {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		panic("failed to init empty statedb")
	}
	return NewGeth(statedb)
}

func (store *gethEvmStorage) Get(key common.Hash) common.Hash {
	return store.db.GetState(store.account, key)
}

func (store *gethEvmStorage) Set(key common.Hash, value common.Hash) {
	store.db.SetState(store.account, key, value)
}

func (store *gethEvmStorage) Swap(key common.Hash, newValue common.Hash) common.Hash {
	oldValue := store.Get(key)
	store.Set(key, newValue)
	return oldValue
}

