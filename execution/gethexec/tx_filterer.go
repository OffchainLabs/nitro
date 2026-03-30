// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
)

// txFilterer implements core.TxFilterer for address-based transaction filtering
// in gas estimation dry-runs. It bridges nitro's filtering logic (address checking,
// event filtering, L1 alias handling) into go-ethereum's gasestimator.
type txFilterer struct {
	checker     state.AddressChecker
	eventFilter *eventfilter.EventFilter
}

func (f *txFilterer) Setup(statedb *state.StateDB) {
	statedb.SetAddressChecker(f.checker)
}

func (f *txFilterer) TouchAddresses(statedb *state.StateDB, tx *types.Transaction, sender common.Address) {
	touchAddresses(statedb, nil, tx, sender)
}

func (f *txFilterer) CheckFiltered(statedb *state.StateDB) error {
	applyEventFilter(f.eventFilter, statedb)
	if statedb.IsAddressFiltered() {
		return state.ErrArbTxFilter
	}
	return nil
}
