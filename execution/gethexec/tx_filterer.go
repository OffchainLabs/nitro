// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
)

// ErrAddressFiltered wraps state.ErrArbTxFilter with the filtered address
// records so callers can build a FilteredTxReport without re-running the check.
type ErrAddressFiltered struct {
	Records []filter.FilteredAddressRecord
}

func (e *ErrAddressFiltered) Error() string { return state.ErrArbTxFilter.Error() }
func (e *ErrAddressFiltered) Unwrap() error { return state.ErrArbTxFilter }

// txFilterer implements core.TxFilterer for address-based transaction filtering
// for node API calls such as eth_estimateGas and eth_call. It wraps ExecutionEngine to resolve the address
// checker lazily, so tests can inject checkers via ExecEngine.SetAddressChecker.
type txFilterer struct {
	execEngine  *ExecutionEngine
	eventFilter *eventfilter.EventFilter
}

func (f *txFilterer) Setup(statedb *state.StateDB) {
	statedb.SetAddressChecker(f.execEngine.addressChecker)
	statedb.SetTxContext(common.Hash{}, 0)
}

func (f *txFilterer) TouchAddresses(statedb *state.StateDB, tx *types.Transaction, sender common.Address) {
	touchAddresses(statedb, tx, sender)
}

func (f *txFilterer) CheckFiltered(statedb *state.StateDB) error {
	applyEventFilter(f.eventFilter, statedb)
	if filtered, records := statedb.IsAddressFiltered(); filtered {
		return &ErrAddressFiltered{Records: records}
	}
	return nil
}
