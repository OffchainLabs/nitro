// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/core/vm"
)

func TestContextBurn(t *testing.T) {
	// Start with 1000 gas available
	ctx := Context{
		gasSupplied: 1_000,
		gasUsed:     multigas.ZeroGas(),
	}
	if got, want := ctx.GasLeft(), uint64(1000); got != want {
		t.Errorf("wrong gas left: got %v, want %v", got, want)
	}
	if got, want := ctx.Burned(), uint64(0); got != want {
		t.Errorf("wrong gas burned: got %v, want %v", got, want)
	}

	// Burn 700 storage access
	if err := ctx.Burn(multigas.ResourceKindStorageAccess, 700); err != nil {
		t.Errorf("unexpected error from burn: %v", err)
	}
	if got, want := ctx.GasLeft(), uint64(300); got != want {
		t.Errorf("wrong gas left: got %v, want %v", got, want)
	}
	if got, want := ctx.Burned(), uint64(700); got != want {
		t.Errorf("wrong gas burned: got %v, want %v", got, want)
	}

	// Burn 200 storage growth
	if err := ctx.Burn(multigas.ResourceKindStorageGrowth, 200); err != nil {
		t.Errorf("unexpected error from burn: %v", err)
	}
	if got, want := ctx.GasLeft(), uint64(100); got != want {
		t.Errorf("wrong gas left: got %v, want %v", got, want)
	}
	if got, want := ctx.Burned(), uint64(900); got != want {
		t.Errorf("wrong gas burned: got %v, want %v", got, want)
	}

	// Burn 200 more storage growth, which should error with out of gas
	if err := ctx.Burn(multigas.ResourceKindStorageGrowth, 200); !errors.Is(err, vm.ErrOutOfGas) {
		t.Errorf("wrong erro from burn: got %v, want %v", err, vm.ErrOutOfGas)
	}
	if got, want := ctx.GasLeft(), uint64(0); got != want {
		t.Errorf("wrong gas left: got %v, want %v", got, want)
	}
	if got, want := ctx.Burned(), uint64(1000); got != want {
		t.Errorf("wrong gas burned: got %v, want %v", got, want)
	}

	// Check the multigas dimensions
	if got, want := ctx.gasUsed.Get(multigas.ResourceKindStorageAccess), uint64(700); got != want {
		t.Errorf("wrong storage access: got %v, want %v", got, want)
	}
	if got, want := ctx.gasUsed.Get(multigas.ResourceKindStorageGrowth), uint64(200); got != want {
		t.Errorf("wrong storage growth: got %v, want %v", got, want)
	}
	if got, want := ctx.gasUsed.Get(multigas.ResourceKindComputation), uint64(100); got != want {
		t.Errorf("wrong computation: got %v, want %v", got, want)
	}
}

func TestContextBurnMultiGas(t *testing.T) {
	ctx := Context{
		gasSupplied: 1_000,
		gasUsed:     multigas.ZeroGas(),
	}
	if got, want := ctx.GasLeft(), uint64(1000); got != want {
		t.Errorf("wrong gas left: got %v, want %v", got, want)
	}
	if got, want := ctx.Burned(), uint64(0); got != want {
		t.Errorf("wrong gas burned: got %v, want %v", got, want)
	}

	gasToBurn := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 400},
		multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 200},
	)
	if err := ctx.BurnMultiGas(gasToBurn); err != nil {
		t.Errorf("unexpected error from burn: %v", err)
	}
	if got, want := ctx.GasLeft(), uint64(400); got != want {
		t.Errorf("wrong gas left: got %v, want %v", got, want)
	}
	if got, want := ctx.Burned(), uint64(600); got != want {
		t.Errorf("wrong gas burned: got %v, want %v", got, want)
	}

	if err := ctx.BurnMultiGas(gasToBurn); !errors.Is(err, vm.ErrOutOfGas) {
		t.Errorf("wrong erro from burn: got %v, want %v", err, vm.ErrOutOfGas)
	}
	if got, want := ctx.GasLeft(), uint64(0); got != want {
		t.Errorf("wrong gas left: got %v, want %v", got, want)
	}
	if got, want := ctx.Burned(), uint64(1000); got != want {
		t.Errorf("wrong gas burned: got %v, want %v", got, want)
	}

	if got, want := ctx.gasUsed.Get(multigas.ResourceKindStorageAccess), uint64(400); got != want {
		t.Errorf("wrong storage access: got %v, want %v", got, want)
	}
	if got, want := ctx.gasUsed.Get(multigas.ResourceKindStorageGrowth), uint64(200); got != want {
		t.Errorf("wrong storage growth: got %v, want %v", got, want)
	}
	if got, want := ctx.gasUsed.Get(multigas.ResourceKindComputation), uint64(400); got != want {
		t.Errorf("wrong computation: got %v, want %v", got, want)
	}
}
