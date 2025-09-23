// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
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
	handler := newApiClosures(evm, nil, scope, &MemoryModel{})
	_, _, expectedCost := handler(GetBytes32, key[:])

	statedb_testing, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	storageAccessGas := vm.WasmStateLoadCost(statedb_testing, acting, key)

	require.Equal(t, expectedCost, storageAccessGas.SingleGas(), "storage-access single gas mismatch")
	require.Equal(t, storageAccessGas, scope.Contract.UsedMultiGas, "contract used multigas mismatch")

	if gotTotal := scope.Contract.UsedMultiGas.SingleGas(); gotTotal != expectedCost {
		t.Fatalf("total multigas mismatch: got %d, want %d", gotTotal, expectedCost)
	}
}
