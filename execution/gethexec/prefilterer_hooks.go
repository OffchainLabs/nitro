// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
)

// PrefiltererSequencingHooks implements arbos.SequencingHooks for the
// prechecker's dry-run filtering. It feeds a single candidate tx into
// ProduceBlockAdvanced and collects address filtering results.
type PrefiltererSequencingHooks struct {
	tx          *types.Transaction
	delivered   bool
	filtered    bool
	eventFilter *eventfilter.EventFilter
}

func (h *PrefiltererSequencingHooks) NextTxToSequence() (*types.Transaction, *arbitrum_types.ConditionalOptions, error) {
	if h.delivered {
		return nil, nil, nil
	}
	h.delivered = true
	return h.tx, nil, nil
}

func (h *PrefiltererSequencingHooks) DiscardInvalidTxsEarly() bool {
	return true
}

func (h *PrefiltererSequencingHooks) PreTxFilter(
	_ *params.ChainConfig,
	_ *types.Header,
	statedb *state.StateDB,
	_ *arbosState.ArbosState,
	tx *types.Transaction,
	_ *arbitrum_types.ConditionalOptions,
	sender common.Address,
	_ *arbos.L1Info,
) error {
	statedb.TouchAddress(sender)
	if tx.To() != nil {
		statedb.TouchAddress(*tx.To())
	}
	if statedb.IsAddressFiltered() {
		h.filtered = true
		return state.ErrArbTxFilter
	}
	return nil
}

func (h *PrefiltererSequencingHooks) PostTxFilter(
	_ *types.Header,
	statedb *state.StateDB,
	_ *arbosState.ArbosState,
	_ *types.Transaction,
	sender common.Address,
	_ uint64,
	_ *core.ExecutionResult,
	_ bool,
) error {
	applyEventFilter(h.eventFilter, statedb, sender)
	// The real sequencer's postTxFilter also checks statedb.IsTxFiltered(),
	// which is the onchain per-tx-hash filter for delayed messages. We omit
	// it here because the prechecker only processes RPC-submitted txs, never
	// delayed messages, so IsTxFiltered() would never fire.
	if statedb.IsAddressFiltered() {
		h.filtered = true
		return state.ErrArbTxFilter
	}
	return nil
}

func (h *PrefiltererSequencingHooks) InsertLastTxError(_ error) {}

func (h *PrefiltererSequencingHooks) ReportGroupRevert(err error) {
	h.filtered = true
}

func (h *PrefiltererSequencingHooks) BlockFilter(
	_ *types.Header, _ *state.StateDB, _ types.Transactions, _ types.Receipts,
) error {
	return nil
}
