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

	"github.com/offchainlabs/nitro/arbos/l1pricing"
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

func newTestHandler(t *testing.T, coinbase common.Address, runCtx *core.MessageRunContext, maxPages uint16) (RequestHandler, vm.StateDB) {
	t.Helper()
	db := state.NewDatabaseForTesting()
	db.SetMaxStylusOpenPages(maxPages)
	statedb, _ := state.New(types.EmptyRootHash, db)
	evm := vm.NewEVM(vm.BlockContext{Coinbase: coinbase}, statedb, params.TestChainConfig, vm.Config{})
	caller := common.Address{}
	acting := common.Address{1}
	contract := vm.NewContract(caller, acting, new(uint256.Int), 1_000_000, nil)
	scope := &vm.ScopeContext{Contract: contract}
	model := NewMemoryModel(InitialFreePages, InitialPageGas)
	handler := newApiClosures(evm, nil, scope, model, runCtx)
	return handler, statedb
}

func callAddPages(handler RequestHandler, pages uint16) uint64 {
	_, _, cost := handler(AddPages, arbmath.Uint16ToBytes(pages))
	return cost
}

func TestAddPages_UnderLimit(t *testing.T) {
	handler, _ := newTestHandler(t, common.Address{}, nil, 128)
	cost := callAddPages(handler, 5)
	require.NotEqual(t, uint64(math.MaxUint64), cost, "should return normal gas cost when under limit")
}

func TestAddPages_LimitDisabled(t *testing.T) {
	// nil runCtx would normally trigger MaxUint64 if limit were enforced,
	// but limit=0 disables the check entirely
	handler, _ := newTestHandler(t, common.Address{}, nil, 0)
	cost := callAddPages(handler, 20)
	require.NotEqual(t, uint64(math.MaxUint64), cost, "should return normal gas cost when limit is disabled")
}

func TestAddPages_ExceedsLimit_NilRunCtx(t *testing.T) {
	handler, statedb := newTestHandler(t, common.Address{}, nil, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "nil runCtx should return MaxUint64 when over limit")
	require.False(t, statedb.IsTxFiltered(), "nil runCtx should not call FilterTx")
}

func TestAddPages_ExceedsLimit_EthCall(t *testing.T) {
	runCtx := core.NewMessageEthcallContext()
	handler, statedb := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "eth_call should return MaxUint64 when over limit")
	require.False(t, statedb.IsTxFiltered(), "eth_call should not call FilterTx")
}

func TestAddPages_ExceedsLimit_GasEstimation(t *testing.T) {
	runCtx := core.NewMessageGasEstimationContext()
	handler, statedb := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "gas estimation should return MaxUint64 when over limit")
	require.False(t, statedb.IsTxFiltered(), "gas estimation should not call FilterTx")
}

func TestAddPages_ExceedsLimit_SequencerCommit(t *testing.T) {
	runCtx := core.NewMessageCommitContext(nil)
	// BatchPosterAddress as coinbase means regular sequencing (not delayed inbox)
	handler, statedb := newTestHandler(t, l1pricing.BatchPosterAddress, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.Equal(t, uint64(math.MaxUint64), cost, "sequencer commit should return MaxUint64 when over limit")
	require.True(t, statedb.IsTxFiltered(), "sequencer commit should call FilterTx when over limit")
}

func TestAddPages_ExceedsLimit_DelayedInbox(t *testing.T) {
	runCtx := core.NewMessageCommitContext(nil)
	// Non-BatchPosterAddress coinbase means delayed inbox
	delayedCoinbase := common.HexToAddress("0xdeadbeef")
	handler, statedb := newTestHandler(t, delayedCoinbase, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.NotEqual(t, uint64(math.MaxUint64), cost, "delayed inbox should return normal gas cost even when over limit")
	require.False(t, statedb.IsTxFiltered(), "delayed inbox should not call FilterTx")
}

func TestAddPages_ExceedsLimit_Replay(t *testing.T) {
	runCtx := core.NewMessageReplayContext()
	handler, statedb := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.NotEqual(t, uint64(math.MaxUint64), cost, "replay should return normal gas cost even when over limit")
	require.False(t, statedb.IsTxFiltered(), "replay should not call FilterTx")
}

func TestAddPages_ExceedsLimit_Recording(t *testing.T) {
	runCtx := core.NewMessageRecordingContext(nil)
	handler, statedb := newTestHandler(t, common.Address{}, runCtx, 10)
	cost := callAddPages(handler, 20)
	require.NotEqual(t, uint64(math.MaxUint64), cost, "recording should return normal gas cost even when over limit")
	require.False(t, statedb.IsTxFiltered(), "recording should not call FilterTx")
}

func TestAddPages_CumulativePages(t *testing.T) {
	handler, _ := newTestHandler(t, common.Address{}, nil, 10)
	// First call: 5 pages, under limit
	cost := callAddPages(handler, 5)
	require.NotEqual(t, uint64(math.MaxUint64), cost, "5 pages should be under limit of 10")

	// Second call: 4 more pages, still under limit (total 9)
	cost = callAddPages(handler, 4)
	require.NotEqual(t, uint64(math.MaxUint64), cost, "9 cumulative pages should be under limit of 10")

	// Third call: 2 more pages, now over limit (total 11)
	cost = callAddPages(handler, 2)
	require.Equal(t, uint64(math.MaxUint64), cost, "11 cumulative pages should exceed limit of 10")
}

func TestAddPages_ExactlyAtLimit(t *testing.T) {
	handler, _ := newTestHandler(t, common.Address{}, nil, 10)
	// Request exactly 10 pages with limit 10: newOpen = 0 + 10 = 10, NOT > 10
	cost := callAddPages(handler, 10)
	require.NotEqual(t, uint64(math.MaxUint64), cost, "exactly at limit should be allowed")
}

func TestAddPages_OneOverLimit(t *testing.T) {
	handler, _ := newTestHandler(t, common.Address{}, nil, 10)
	// Request 11 pages with limit 10: newOpen = 0 + 11 = 11 > 10
	cost := callAddPages(handler, 11)
	require.Equal(t, uint64(math.MaxUint64), cost, "one over limit should be rejected")
}
