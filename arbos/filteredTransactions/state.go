// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package filteredTransactions

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
)

var presentHash = common.BytesToHash([]byte{1})

type FilteredTransactionsState struct {
	store *storage.Storage
}

func Open(statedb vm.StateDB, burner burn.Burner) *FilteredTransactionsState {
	return &FilteredTransactionsState{
		store: storage.FilteredTransactionsStorage(statedb, burner),
	}
}

func (s *FilteredTransactionsState) Add(txHash common.Hash) error {
	if s == nil {
		return nil
	}
	return s.store.Set(txHash, presentHash)
}

func (s *FilteredTransactionsState) Delete(txHash common.Hash) error {
	if s == nil {
		return nil
	}
	return s.store.Clear(txHash)
}

func (s *FilteredTransactionsState) IsFiltered(txHash common.Hash) (bool, error) {
	if s == nil {
		return false, nil
	}
	value, err := s.store.Get(txHash)
	if err != nil {
		return false, err
	}
	return value == presentHash, nil
}

func (s *FilteredTransactionsState) IsFilteredFree(txHash common.Hash) bool {
	if s == nil {
		return false
	}
	return s.store.GetFree(txHash) == presentHash
}

// DeleteFree removes a tx hash from the filter without charging gas.
// This is called after a filtered tx is executed as a no-op to clean up.
// The entry is truly deleted from the storage trie (not just set to zero).
func (s *FilteredTransactionsState) DeleteFree(txHash common.Hash) {
	if s == nil {
		return
	}
	s.store.ClearFree(txHash)
}
