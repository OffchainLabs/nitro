// Copyright 2025-2026, Offchain Labs, Inc.
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
	handler := newApiClosures(evm, nil, scope, &MemoryModel{}, nil, nil)
	_, _, expectedCost := handler(GetBytes32, key[:])

	statedb_testing, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	storageAccessGas := vm.WasmStateLoadCost(statedb_testing, acting, key)

	require.Equal(t, expectedCost, storageAccessGas.SingleGas(), "storage-access single gas mismatch")
	require.Equal(t, storageAccessGas, scope.Contract.UsedMultiGas, "contract used multigas mismatch")

	if gotTotal := scope.Contract.UsedMultiGas.SingleGas(); gotTotal != expectedCost {
		t.Fatalf("total multigas mismatch: got %d, want %d", gotTotal, expectedCost)
	}
}

// buildAddPagesTestHandler is the shared setup used by newTestHandler (exercises
// the node-level ArbNodeConfig.MaxOpenPages path) and newConsensusTestHandler
// (exercises the chain-level StylusParams.PageLimit consensus path). Passing 0
// for maxPages or pageLimit disables the respective check; passing 0 for
// arbosVersion disables the ArbOS >= 59 gate on the consensus check.
func buildAddPagesTestHandler(t *testing.T, coinbase common.Address, runCtx *core.MessageRunContext, maxPages, pageLimit uint16, arbosVersion uint64) (RequestHandler, vm.StateDB, *MemoryModel) {
	t.Helper()
	db := state.NewDatabaseForTesting()
	db.SetArbNodeConfig(&ArbNodeConfig{MaxOpenPages: maxPages})
	statedb, _ := state.New(types.EmptyRootHash, db)
	evm := vm.NewEVM(vm.BlockContext{Coinbase: coinbase, ArbOSVersion: arbosVersion}, statedb, params.TestChainConfig, vm.Config{})
	caller := common.Address{}
	acting := common.Address{1}
	contract := vm.NewContract(caller, acting, new(uint256.Int), 1_000_000, nil)
	scope := &vm.ScopeContext{Contract: contract}
	model := NewMemoryModel(InitialFreePages, InitialPageGas)
	stylusParams := &StylusParams{PageLimit: pageLimit}
	handler := newApiClosures(evm, nil, scope, model, runCtx, stylusParams)
	return handler, statedb, model
}

// newTestHandler isolates the node-level ArbNodeConfig.MaxOpenPages path.
// The consensus-level check is disabled (pageLimit=0, arbosVersion=0).
func newTestHandler(t *testing.T, coinbase common.Address, runCtx *core.MessageRunContext, maxPages uint16) (RequestHandler, vm.StateDB, *MemoryModel) {
	t.Helper()
	return buildAddPagesTestHandler(t, coinbase, runCtx, maxPages, 0, 0)
}

// newConsensusTestHandler isolates the chain-level StylusParams.PageLimit
// consensus path. The node-level MaxOpenPages check is disabled (maxPages=0).
func newConsensusTestHandler(t *testing.T, runCtx *core.MessageRunContext, pageLimit uint16, arbosVersion uint64) (RequestHandler, vm.StateDB, *MemoryModel) {
	t.Helper()
	return buildAddPagesTestHandler(t, common.Address{}, runCtx, 0, pageLimit, arbosVersion)
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
	require.True(t, statedb.IsTxFiltered(), "eth_call should call FilterTx when over limit")
}

func TestAddPages_ExceedsLimit_GasEstimation(t *testing.T) {
	runCtx := core.NewMessageGasEstimationContext()
	handler, statedb, _ := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "gas estimation should return MaxUint64 when over limit")
	require.True(t, statedb.IsTxFiltered(), "gas estimation should call FilterTx when over limit")
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
	handler := newApiClosures(evm, nil, scope, model, core.NewMessageEthcallContext(), nil)

	// Request 100 pages — well over any realistic limit. Because limit stays 0
	// (fail-open), this should return normal gas cost, not MaxUint64.
	cost := callAddPages(handler, 100)
	require.Equal(t, model.GasCost(100, 0, 0), cost, "wrong config type should fail open (charge normal gas), not OOG")
	require.False(t, statedb.IsTxFiltered(), "wrong config type should not call FilterTx")
}

// The following tests cover the chain-level consensus page limit check added
// for ArbOS >= 59 (StylusParams.PageLimit). Unlike the node-level
// ArbNodeConfig.MaxOpenPages check exercised above, this check is:
//  - gated on evm.Context.ArbOSVersion >= params.ArbosVersion_59
//  - independent of runCtx (every run mode OOGs identically)

func TestAddPages_StylusPageLimit_ArbOS59_Exceeds_NilRunCtx(t *testing.T) {
	// runCtx=nil: the existing node-level check short-circuits on nil, but the
	// consensus check must still fire — it is independent of runCtx.
	handler, statedb, _ := newConsensusTestHandler(t, nil, 10, params.ArbosVersion_59)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "ArbOS 59 consensus limit should OOG regardless of nil runCtx")
	require.False(t, statedb.IsTxFiltered(), "consensus check must not call FilterTx")
}

func TestAddPages_StylusPageLimit_ArbOS59_Exceeds_EthCall(t *testing.T) {
	handler, statedb, _ := newConsensusTestHandler(t, core.NewMessageEthcallContext(), 10, params.ArbosVersion_59)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "eth_call should OOG under ArbOS 59 consensus limit")
	require.False(t, statedb.IsTxFiltered(), "consensus check must not call FilterTx in eth_call")
}

func TestAddPages_StylusPageLimit_ArbOS58_NotEnforced(t *testing.T) {
	// At ArbOS 58 the consensus check is inert — the call should charge normal
	// gas cost even when newOpen exceeds pageLimit.
	handler, _, model := newConsensusTestHandler(t, core.NewMessageEthcallContext(), 10, params.ArbosVersion_59-1)
	cost := callAddPages(handler, 20)
	require.Equal(t, model.GasCost(20, 0, 0), cost, "ArbOS < 59 must not enforce the consensus page limit")
}

func TestAddPages_StylusPageLimit_ArbOS59_Exceeds_Sequencing(t *testing.T) {
	handler, statedb, _ := newConsensusTestHandler(t, core.NewMessageSequencingContext(nil), 10, params.ArbosVersion_59)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "sequencing should OOG under ArbOS 59 consensus limit")
	// Crucially, the consensus check short-circuits BEFORE the node-level
	// sequencing FilterTx path, so IsTxFiltered must remain false.
	require.False(t, statedb.IsTxFiltered(), "consensus check must not call FilterTx in sequencing")
}

func TestAddPages_StylusPageLimit_ArbOS59_Exceeds_Replay(t *testing.T) {
	handler, statedb, _ := newConsensusTestHandler(t, core.NewMessageReplayContext(), 10, params.ArbosVersion_59)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "replay should OOG under ArbOS 59 consensus limit (deterministic across nodes)")
	require.False(t, statedb.IsTxFiltered(), "consensus check must not call FilterTx in replay")
}

func TestAddPages_StylusPageLimit_ArbOS59_Exceeds_Recording(t *testing.T) {
	handler, statedb, _ := newConsensusTestHandler(t, core.NewMessageRecordingContext(nil), 10, params.ArbosVersion_59)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "recording should OOG under ArbOS 59 consensus limit")
	require.False(t, statedb.IsTxFiltered(), "consensus check must not call FilterTx in recording")
}

func TestAddPages_StylusPageLimit_ArbOS59_Under(t *testing.T) {
	handler, _, model := newConsensusTestHandler(t, core.NewMessageEthcallContext(), 10, params.ArbosVersion_59)
	cost := callAddPages(handler, 5)
	require.Equal(t, model.GasCost(5, 0, 0), cost, "under the consensus limit should charge normal gas cost")
}

func TestAddPages_StylusPageLimit_ArbOS59_ExactlyAtLimit(t *testing.T) {
	// newOpen = 0 + 10 = 10, pageLimit = 10 → NOT > 10 → allowed.
	handler, _, model := newConsensusTestHandler(t, core.NewMessageEthcallContext(), 10, params.ArbosVersion_59)
	cost := callAddPages(handler, 10)
	require.Equal(t, model.GasCost(10, 0, 0), cost, "exactly at consensus limit should charge normal gas (strict >)")
}

func TestAddPages_StylusPageLimit_ArbOS59_OneOverLimit(t *testing.T) {
	// newOpen = 0 + 11 = 11, pageLimit = 10 → 11 > 10 → OOG.
	handler, statedb, _ := newConsensusTestHandler(t, core.NewMessageEthcallContext(), 10, params.ArbosVersion_59)
	cost := callAddPages(handler, 11)
	require.Equal(t, uint64(math.MaxUint64), cost, "one over consensus limit should OOG")
	require.False(t, statedb.IsTxFiltered(), "consensus check must not call FilterTx")
}

func TestAddPages_StylusPageLimit_ArbOS59_Zero_Disabled(t *testing.T) {
	// pageLimit=0 is treated as disabled for consistency with the node-level
	// check; otherwise any page allocation would OOG on a chain with an
	// unconfigured PageLimit.
	handler, _, model := newConsensusTestHandler(t, core.NewMessageEthcallContext(), 0, params.ArbosVersion_59)
	cost := callAddPages(handler, 100)
	require.Equal(t, model.GasCost(100, 0, 0), cost, "pageLimit=0 should disable the consensus check")
}

// ---------------------------------------------------------------------------
// Direct unit tests for enforceStylusPageLimit
// ---------------------------------------------------------------------------
//
// The TestAddPages_* tests above exercise enforceStylusPageLimit indirectly
// through the addPages hostio closure. The tests below call the function
// directly, which avoids the full handler setup and makes it easier to cover
// edge cases and the nil-config path.

// buildEnforceTestArgs creates a minimal EVM + StateDB for calling
// enforceStylusPageLimit directly. maxPages=0 leaves ArbNodeConfig nil (tests
// the nil-config path); otherwise it is set on the StateDB database.
func buildEnforceTestArgs(t *testing.T, maxPages uint16, setConfig bool, arbosVersion uint64) (*vm.EVM, vm.StateDB) {
	t.Helper()
	db := state.NewDatabaseForTesting()
	if setConfig {
		db.SetArbNodeConfig(&ArbNodeConfig{MaxOpenPages: maxPages})
	}
	statedb, _ := state.New(types.EmptyRootHash, db)
	evm := vm.NewEVM(vm.BlockContext{ArbOSVersion: arbosVersion}, statedb, params.TestChainConfig, vm.Config{})
	return evm, statedb
}

func TestEnforce_NilConfig_ReturnsZero(t *testing.T) {
	// ArbNodeConfig never set → raw == nil → limit stays 0 → returns 0.
	evm, statedb := buildEnforceTestArgs(t, 0, false, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 200, common.Address{1}, nil, pageLimitCallProgram)
	require.Equal(t, uint64(0), penalty, "nil config should fail open (no enforcement)")
}

func TestEnforce_WrongConfigType_ReturnsZero(t *testing.T) {
	db := state.NewDatabaseForTesting()
	db.SetArbNodeConfig("not a *ArbNodeConfig")
	statedb, _ := state.New(types.EmptyRootHash, db)
	evm := vm.NewEVM(vm.BlockContext{}, statedb, params.TestChainConfig, vm.Config{})
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 200, common.Address{1}, nil, pageLimitCallProgram)
	require.Equal(t, uint64(0), penalty, "wrong config type should fail open")
	require.False(t, statedb.IsTxFiltered())
}

func TestEnforce_LimitDisabled_ReturnsZero(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 0, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 200, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "MaxOpenPages=0 disables the check")
}

func TestEnforce_UnderLimit_ReturnsZero(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 100, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 50, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "under limit should allow")
}

func TestEnforce_ExactlyAtLimit_ReturnsZero(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 10, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 10, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "exactly at limit should be allowed (strict >)")
}

func TestEnforce_OverLimit_NilRunCtx_ReturnsZero(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 10, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, nil, 20, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "nil runCtx should fail open")
	require.False(t, statedb.IsTxFiltered())
}

func TestEnforce_OverLimit_EthCall_FiltersAndReturnsMaxUint64(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 10, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 20, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(math.MaxUint64), penalty, "eth_call over limit should OOG")
	require.True(t, statedb.IsTxFiltered(), "eth_call should call FilterTx")
}

func TestEnforce_OverLimit_GasEstimation_FiltersAndReturnsMaxUint64(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 10, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageGasEstimationContext(), 20, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(math.MaxUint64), penalty, "gas estimation over limit should OOG")
	require.True(t, statedb.IsTxFiltered(), "gas estimation should call FilterTx")
}

func TestEnforce_OverLimit_Sequencing_FiltersAndReturnsMaxUint64(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 10, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageSequencingContext(nil), 20, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(math.MaxUint64), penalty, "sequencing over limit should OOG")
	require.True(t, statedb.IsTxFiltered(), "sequencing should call FilterTx")
}

func TestEnforce_OverLimit_Commit_Exempt(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 10, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageCommitContext(nil), 20, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "commit should be exempt (log only)")
	require.False(t, statedb.IsTxFiltered())
}

func TestEnforce_OverLimit_Replay_Exempt(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 10, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageReplayContext(), 20, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "replay should be exempt (log only)")
	require.False(t, statedb.IsTxFiltered())
}

func TestEnforce_OverLimit_Recording_Exempt(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 10, true, 0)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageRecordingContext(nil), 20, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "recording should be exempt (log only)")
	require.False(t, statedb.IsTxFiltered())
}

// Consensus tests

func TestEnforce_Consensus_ArbOS59_OverPageLimit(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 0, true, params.ArbosVersion_59)
	sp := &StylusParams{PageLimit: 10}
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 20, common.Address{1}, sp, pageLimitAddPages)
	require.Equal(t, uint64(math.MaxUint64), penalty, "ArbOS 59 consensus limit should OOG")
	require.False(t, statedb.IsTxFiltered(), "consensus check must not FilterTx")
}

func TestEnforce_Consensus_ArbOS59_UnderPageLimit(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 0, true, params.ArbosVersion_59)
	sp := &StylusParams{PageLimit: 50}
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 20, common.Address{1}, sp, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "under consensus limit should allow")
}

func TestEnforce_Consensus_PreArbOS59_NotEnforced(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 0, true, params.ArbosVersion_59-1)
	sp := &StylusParams{PageLimit: 10}
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 20, common.Address{1}, sp, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "pre-ArbOS 59 should not enforce consensus limit")
}

func TestEnforce_Consensus_NilStylusParams_NotEnforced(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 0, true, params.ArbosVersion_59)
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 200, common.Address{1}, nil, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "nil stylusParams should skip consensus check")
}

func TestEnforce_Consensus_ZeroPageLimit_Disabled(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 0, true, params.ArbosVersion_59)
	sp := &StylusParams{PageLimit: 0}
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 200, common.Address{1}, sp, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "PageLimit=0 should disable consensus check")
}

func TestEnforce_Consensus_TakesPrecedence_OverNodeLimit(t *testing.T) {
	// Both caps active, consensus fires first → no FilterTx even in sequencing.
	evm, statedb := buildEnforceTestArgs(t, 10, true, params.ArbosVersion_59)
	sp := &StylusParams{PageLimit: 10}
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageSequencingContext(nil), 20, common.Address{1}, sp, pageLimitAddPages)
	require.Equal(t, uint64(math.MaxUint64), penalty, "consensus cap should fire")
	require.False(t, statedb.IsTxFiltered(), "consensus cap fires before node-level, so no FilterTx")
}

func TestEnforce_Consensus_ExactlyAtPageLimit_Allowed(t *testing.T) {
	evm, statedb := buildEnforceTestArgs(t, 0, true, params.ArbosVersion_59)
	sp := &StylusParams{PageLimit: 10}
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), 10, common.Address{1}, sp, pageLimitAddPages)
	require.Equal(t, uint64(0), penalty, "exactly at consensus limit should be allowed (strict >)")
}

func TestEnforce_Consensus_NilRunCtx_StillEnforced(t *testing.T) {
	// Unlike the node-level check, the consensus check is runCtx-independent.
	evm, statedb := buildEnforceTestArgs(t, 0, true, params.ArbosVersion_59)
	sp := &StylusParams{PageLimit: 10}
	penalty := enforceStylusPageLimit(evm, statedb, nil, 20, common.Address{1}, sp, pageLimitAddPages)
	require.Equal(t, uint64(math.MaxUint64), penalty, "consensus check must fire even with nil runCtx")
}

// The following tests pin the cumulative-footprint semantics that
// programs.CallProgram relies on: the helper is fed `open + footprint` and
// must decide OOG/allow against that cumulative value, matching the
// addPages hostio's behavior. A regression that reverted CallProgram to
// passing raw footprint would pass these helper-level tests; the
// integration tests in system_tests/program_test.go guard that call site.
func TestEnforce_CallProgram_CumulativeFootprint_OverConsensusLimit(t *testing.T) {
	// Outer frame has already consumed 24 open pages; inner footprint 120.
	// Neither alone exceeds PageLimit=128, but cumulative 24+120=144 does.
	evm, statedb := buildEnforceTestArgs(t, 0, true, params.ArbosVersion_59)
	sp := &StylusParams{PageLimit: 128}
	newOpen := arbmath.SaturatingUAdd(uint16(24), uint16(120))
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), newOpen, common.Address{1}, sp, pageLimitCallProgram)
	require.Equal(t, uint64(math.MaxUint64), penalty, "cumulative 144 > 128 should OOG on consensus path")
	require.False(t, statedb.IsTxFiltered(), "consensus path must not FilterTx")
}

func TestEnforce_CallProgram_CumulativeFootprint_UnderConsensusLimit(t *testing.T) {
	// Same split but PageLimit raised so cumulative fits.
	evm, statedb := buildEnforceTestArgs(t, 0, true, params.ArbosVersion_59)
	sp := &StylusParams{PageLimit: 200}
	newOpen := arbmath.SaturatingUAdd(uint16(24), uint16(120))
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageEthcallContext(), newOpen, common.Address{1}, sp, pageLimitCallProgram)
	require.Equal(t, uint64(0), penalty, "cumulative 144 <= 200 should allow")
}

func TestEnforce_CallProgram_CumulativeFootprint_OverNodeLimit_Sequencing(t *testing.T) {
	// Mirrors the node-level MaxOpenPages path: in a sequencing context,
	// cumulative-exceed must OOG AND call FilterTx.
	evm, statedb := buildEnforceTestArgs(t, 128, true, 0)
	newOpen := arbmath.SaturatingUAdd(uint16(24), uint16(120))
	penalty := enforceStylusPageLimit(evm, statedb, core.NewMessageSequencingContext(nil), newOpen, common.Address{1}, nil, pageLimitCallProgram)
	require.Equal(t, uint64(math.MaxUint64), penalty, "cumulative 144 > 128 must OOG in sequencing")
	require.True(t, statedb.IsTxFiltered(), "node path must FilterTx in sequencing")
}
