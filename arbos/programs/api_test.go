// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"math"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestApiClosuresMultiGas_GetBytes32(t *testing.T) {
	// EVM + state setup
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	evm := vm.NewEVM(vm.BlockContext{}, statedb, params.TestChainConfig, vm.Config{})

	// Scope + contract setup
	caller := common.Address{}
	acting := common.Address{1}
	var key common.Hash // dummy hash
	contractGas := uint64(1_000_000)
	contract := vm.NewContract(caller, acting, new(uint256.Int), contractGas, nil)
	scope := &vm.ScopeContext{Contract: contract}

	// Execute handler to update contract multi-gas usage
	handler := newApiClosures(evm, nil, scope, &MemoryModel{}, nil)
	_, _, expectedCost := handler(GetBytes32, key[:])

	statedb_testing, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	storageAccessGas := vm.WasmStateLoadCost(statedb_testing, acting, key)

	require.Equal(t, expectedCost, storageAccessGas.SingleGas(), "storage-access single gas mismatch")
	require.Equal(t, storageAccessGas, scope.Contract.UsedMultiGas, "contract used multigas mismatch")

	if gotTotal := scope.Contract.UsedMultiGas.SingleGas(); gotTotal != expectedCost {
		t.Fatalf("total multigas mismatch: got %d, want %d", gotTotal, expectedCost)
	}
}

func newTestHandler(t *testing.T, coinbase common.Address, runCtx *core.MessageRunContext, maxPages uint16) (RequestHandler, vm.StateDB, *MemoryModel) {
	t.Helper()
	db := state.NewDatabaseForTesting()
	db.SetArbNodeConfig(&ArbNodeConfig{MaxOpenPages: maxPages})
	statedb, _ := state.New(types.EmptyRootHash, db)
	evm := vm.NewEVM(vm.BlockContext{Coinbase: coinbase}, statedb, params.TestChainConfig, vm.Config{})
	caller := common.Address{}
	acting := common.Address{1}
	contract := vm.NewContract(caller, acting, new(uint256.Int), 1_000_000, nil)
	scope := &vm.ScopeContext{Contract: contract}
	model := NewMemoryModel(InitialFreePages, InitialPageGas)
	handler := newApiClosures(evm, nil, scope, model, runCtx)
	return handler, statedb, model
}

func callAddPages(handler RequestHandler, pages uint16) uint64 {
	_, _, cost := handler(AddPages, arbmath.Uint16ToBytes(pages))
	return cost
}

func TestAddPages_UnderLimit(t *testing.T) {
	handler, _, model := newTestHandler(t, common.Address{}, nil, 128)
	cost := callAddPages(handler, 5)
	require.Equal(t, model.GasCost(5, 0, 0), cost, "should return normal gas cost when under limit")
}

func TestAddPages_LimitDisabled(t *testing.T) {
	// limit=0 disables the check entirely, so the exceeded-limit branch is never taken
	handler, _, model := newTestHandler(t, common.Address{}, nil, 0)
	cost := callAddPages(handler, 20)
	require.Equal(t, model.GasCost(20, 0, 0), cost, "should return normal gas cost when limit is disabled")
}

func TestAddPages_ExceedsLimit_NilRunCtx(t *testing.T) {
	// A nil runCtx is treated as on-chain-exempt: charges normal gas, no FilterTx.
	// This is the consensus-safe default — OOG'ing here would diverge from the
	// sequencer if a nil ever slipped through on a replay/recording path.
	handler, statedb, model := newTestHandler(t, common.Address{}, nil, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, model.GasCost(20, 0, 0), cost, "nil runCtx should fall through to normal gas cost when over limit")
	require.False(t, statedb.IsTxFiltered(), "nil runCtx should not call FilterTx")
}

func TestAddPages_ExceedsLimit_EthCall(t *testing.T) {
	runCtx := core.NewMessageEthcallContext()
	handler, statedb, _ := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "eth_call should return MaxUint64 when over limit")
	require.False(t, statedb.IsTxFiltered(), "eth_call should not call FilterTx")
}

func TestAddPages_ExceedsLimit_GasEstimation(t *testing.T) {
	runCtx := core.NewMessageGasEstimationContext()
	handler, statedb, _ := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "gas estimation should return MaxUint64 when over limit")
	require.False(t, statedb.IsTxFiltered(), "gas estimation should not call FilterTx")
}

func TestAddPages_ExceedsLimit_Sequencing(t *testing.T) {
	runCtx := core.NewMessageSequencingContext(nil)
	handler, statedb, _ := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "regular sequencing should return MaxUint64 when over limit")
	require.True(t, statedb.IsTxFiltered(), "regular sequencing should call FilterTx when over limit")
}

func TestAddPages_ExceedsLimit_DelayedSequencing(t *testing.T) {
	// Delayed-inbox sequencing must never trigger FilterTx (censorship resistance).
	// Because IsSequencing() returns false for delayed contexts, addPages skips the
	// FilterTx branch and charges normal gas.
	runCtx := core.NewMessageDelayedSequencingContext(nil)
	handler, statedb, model := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, model.GasCost(20, 0, 0), cost, "delayed sequencing should return normal gas cost even when over limit")
	require.False(t, statedb.IsTxFiltered(), "delayed sequencing must not call FilterTx")
}

func TestAddPages_ExceedsLimit_Commit(t *testing.T) {
	runCtx := core.NewMessageCommitContext(nil)
	handler, statedb, model := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, model.GasCost(20, 0, 0), cost, "commit (replay/digest) should return normal gas cost even when over limit")
	require.False(t, statedb.IsTxFiltered(), "commit (replay/digest) should not call FilterTx")
}

func TestAddPages_ExceedsLimit_Replay(t *testing.T) {
	runCtx := core.NewMessageReplayContext()
	handler, statedb, model := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, model.GasCost(20, 0, 0), cost, "replay should return normal gas cost even when over limit")
	require.False(t, statedb.IsTxFiltered(), "replay should not call FilterTx")
}

func TestAddPages_ExceedsLimit_Recording(t *testing.T) {
	runCtx := core.NewMessageRecordingContext(nil)
	handler, statedb, model := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, model.GasCost(20, 0, 0), cost, "recording should return normal gas cost even when over limit")
	require.False(t, statedb.IsTxFiltered(), "recording should not call FilterTx")
}

func TestAddPages_CumulativePages(t *testing.T) {
	handler, _, model := newTestHandler(t, common.Address{}, core.NewMessageEthcallContext(), 10)
	// First call: 5 pages, under limit. oldOpen=0, ever=0 → open=5, ever=5
	cost := callAddPages(handler, 5)
	require.Equal(t, model.GasCost(5, 0, 0), cost, "5 pages should be under limit of 10")

	// Second call: 4 more pages, still under limit (total 9). oldOpen=5, ever=5 → open=9, ever=9
	cost = callAddPages(handler, 4)
	require.Equal(t, model.GasCost(4, 5, 5), cost, "9 cumulative pages should be under limit of 10")

	// Third call: 2 more pages, now over limit (total 11)
	cost = callAddPages(handler, 2)
	require.Equal(t, uint64(math.MaxUint64), cost, "11 cumulative pages should exceed limit of 10")
}

func TestAddPages_ExactlyAtLimit(t *testing.T) {
	handler, _, model := newTestHandler(t, common.Address{}, core.NewMessageEthcallContext(), 10)
	// Request exactly 10 pages with limit 10: newOpen = 0 + 10 = 10, NOT > 10
	cost := callAddPages(handler, 10)
	require.Equal(t, model.GasCost(10, 0, 0), cost, "exactly at limit should be allowed")
}

func TestAddPages_OneOverLimit(t *testing.T) {
	handler, _, _ := newTestHandler(t, common.Address{}, core.NewMessageEthcallContext(), 10)
	// Request 11 pages with limit 10: newOpen = 0 + 11 = 11 > 10
	cost := callAddPages(handler, 11)
	require.Equal(t, uint64(math.MaxUint64), cost, "one over limit should be rejected")
}

// TestAddPages_NestedFrameRollback simulates the nested-call frame structure
// that programs.CallProgram uses (capture open, add footprint, defer restore)
// and verifies two invariants:
//  1. After an inner frame returns, openWasmPages is restored to the outer
//     frame's post-footprint value — subsequent gas costs use the correct base.
//  2. An inner frame that exceeds the page limit does not affect the outer
//     frame's accounting after the inner unwinds.
//
// The defer is simulated via a nested closure rather than going through
// CallProgram directly.
func TestAddPages_NestedFrameRollback(t *testing.T) {
	limit := uint16(20)
	handler, statedb, model := newTestHandler(t, common.Address{}, core.NewMessageEthcallContext(), limit)

	// Outer frame: capture, add footprint 8, defer restore. Mirrors programs.go:220-221.
	outerOpen := statedb.GetStylusPagesOpen()
	statedb.AddStylusPages(8)
	defer statedb.SetStylusPagesOpen(outerOpen)
	require.Equal(t, uint16(8), statedb.GetStylusPagesOpen(), "outer footprint applied")

	// Outer hostio addPages(5): open 8 → 13, under limit.
	cost := callAddPages(handler, 5)
	require.Equal(t, model.GasCost(5, 8, 8), cost, "outer addPages under limit")
	require.Equal(t, uint16(13), statedb.GetStylusPagesOpen())

	// Inner frame, scoped in a closure so the defer fires on return — exactly
	// what programs.CallProgram does on a nested Stylus call.
	func() {
		innerOpen := statedb.GetStylusPagesOpen()
		statedb.AddStylusPages(4) // inner footprint
		defer statedb.SetStylusPagesOpen(innerOpen)
		require.Equal(t, uint16(17), statedb.GetStylusPagesOpen(), "inner footprint stacked")

		// Inner hostio addPages(10): open 17 → 27, exceeds limit 20 → OOG.
		innerCost := callAddPages(handler, 10)
		require.Equal(t, uint64(math.MaxUint64), innerCost, "inner over-limit should OOG")
	}()

	// After inner unwinds: open must be back to the outer's post-addPages value
	// (13), not leaked at 27 and not over-rolled to 0 or 8.
	require.Equal(t, uint16(13), statedb.GetStylusPagesOpen(), "inner frame rollback restored outer open")

	// Outer continues: addPages(2) → 13+2=15, still under limit → normal gas.
	// If the rollback had leaked (e.g., open stayed at 27), this would OOG.
	// Note: everWasmPages was bumped to 27 during the inner frame (AddStylusPages
	// mutates before the limit check returns MaxUint64) and is NOT rolled back by
	// the defer — only openWasmPages is. So the oldEver passed to GasCost is 27.
	cost = callAddPages(handler, 2)
	require.Equal(t, model.GasCost(2, 13, 27), cost, "outer addPages after inner rollback charges normal gas; oldEver reflects inner's peak")
	require.Equal(t, uint16(15), statedb.GetStylusPagesOpen())
}

// TestAddPages_WrongConfigTypeFailsOpen covers the defensive else branch in
// addPages: if geth's `any` slot ever holds a non-*ArbNodeConfig value
// (a Nitro-internal wiring bug), addPages must fall through and charge normal
// gas rather than panicking or returning OOG. This preserves the pre-feature
// behavior (no limit) — the comment at the call site calls this fail-open.
// A regression that changed the behavior to panic or OOG would silently break
// every Stylus program on the node until the wiring bug is fixed.
func TestAddPages_WrongConfigTypeFailsOpen(t *testing.T) {
	db := state.NewDatabaseForTesting()
	// Stuff a value of the wrong concrete type through the any slot. The type
	// assertion in addPages (raw.(*ArbNodeConfig)) will fail, triggering the
	// fail-open branch.
	db.SetArbNodeConfig("not a *ArbNodeConfig")
	statedb, _ := state.New(types.EmptyRootHash, db)
	evm := vm.NewEVM(vm.BlockContext{}, statedb, params.TestChainConfig, vm.Config{})
	caller := common.Address{}
	acting := common.Address{1}
	contract := vm.NewContract(caller, acting, new(uint256.Int), 1_000_000, nil)
	scope := &vm.ScopeContext{Contract: contract}
	model := NewMemoryModel(InitialFreePages, InitialPageGas)
	// Use an eth_call runCtx so that if the limit *were* enforced, we'd expect
	// MaxUint64. The test asserts the limit is NOT enforced (fail-open).
	handler := newApiClosures(evm, nil, scope, model, core.NewMessageEthcallContext())

	// Request 100 pages — well over any realistic limit. Because limit stays 0
	// (fail-open), this should return normal gas cost, not MaxUint64.
	cost := callAddPages(handler, 100)
	require.Equal(t, model.GasCost(100, 0, 0), cost, "wrong config type should fail open (charge normal gas), not OOG")
	require.False(t, statedb.IsTxFiltered(), "wrong config type should not call FilterTx")
}
