// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package censored_transactions

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
)

type CensoredTransactionsState struct {
	store *storage.Storage
}

func Open(statedb vm.StateDB, burner burn.Burner) *CensoredTransactionsState {
	return &CensoredTransactionsState{
		store: storage.CensoredTransactionsStorage(statedb, burner),
	}
}

func (s *CensoredTransactionsState) Add(txHash common.Hash) error {
	return s.store.SetUint64(txHash, 1)
}

func (s *CensoredTransactionsState) Delete(txHash common.Hash) error {
	return s.store.Clear(txHash)
}

func (s *CensoredTransactionsState) IsCensored(txHash common.Hash) (bool, error) {
	v, err := s.store.GetUint64(txHash)
	if err != nil {
		return false, err
	}
	return v != 0, nil
}
