// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"testing"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
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
	interp := vm.NewEVMInterpreter(evm)

	// Scope + contract setup
	caller := common.Address{}
	acting := common.Address{1}
	var key common.Hash // dummy hash
	contractGas := uint64(1_000_000)
	contract := vm.NewContract(caller, acting, new(uint256.Int), contractGas, nil)
	scope := &vm.ScopeContext{Contract: contract}

	// Execute handler to update contract multi-gas usage
	handler := newApiClosures(interp, nil, scope, &MemoryModel{})
	_, _, expectedCost := handler(GetBytes32, key[:])

	storageAccessGas := scope.Contract.UsedMultiGas.Get(multigas.ResourceKindStorageAccess)
	if storageAccessGas != expectedCost {
		t.Fatalf("storage-access gas mismatch: got %d, want %d", storageAccessGas, expectedCost)
	}
	if gotTotal := scope.Contract.UsedMultiGas.SingleGas(); gotTotal != expectedCost {
		t.Fatalf("total multigas mismatch: got %d, want %d", gotTotal, expectedCost)
	}
}
