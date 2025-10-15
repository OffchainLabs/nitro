// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestSiompleResourceConstraintsPrecompiles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	Require(t, err)

	rc1 := []precompilesgen.ArbResourceConstraintsTypesResourceWeight{
		{Resource: uint8(multigas.ResourceKindComputation), Weight: 1},
	}
	periodSecs := uint32(12)
	targetPerSec := uint64(7_000_000)

	tx, err := arbOwner.SetResourceConstraint(&auth, rc1, periodSecs, targetPerSec)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// List and verify the constraint is set
	list, err := arbGasInfo.ListResourceConstraints(callOpts)
	Require(t, err)
	if len(list) != 1 {
		t.Fatalf("expected 1 constraint, got %d", len(list))
	}
	got := list[0]
	if got.PeriodSecs != periodSecs {
		t.Fatalf("expected periodSecs=%d got %d", periodSecs, got.PeriodSecs)
	}
	if got.TargetPerSec != targetPerSec {
		t.Fatalf("expected targetPerSec=%d got %d", targetPerSec, got.TargetPerSec)
	}
	if len(got.Resources) != len(rc1) {
		t.Fatalf("expected %d resources, got %d", len(rc1), len(got.Resources))
	}
	if got.Resources[0].Resource != rc1[0].Resource ||
		got.Resources[0].Weight != rc1[0].Weight {
		t.Fatalf("resource mismatch: want %+v got %+v", rc1[0], got.Resources[0])
	}

	// Clear the constraint
	tx, err = arbOwner.ClearConstraint(&auth, []uint8{uint8(multigas.ResourceKindComputation)}, periodSecs)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Verify the constraint is cleared
	list, err = arbGasInfo.ListResourceConstraints(callOpts)
	Require(t, err)
	if len(list) != 0 {
		t.Fatalf("expected constraints cleared, found %d", len(list))
	}
}

func TestMultipleResourceConstraintsPrecompiles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	Require(t, err)

	// Set first constraint
	rc1 := []precompilesgen.ArbResourceConstraintsTypesResourceWeight{
		{Resource: uint8(multigas.ResourceKindComputation), Weight: 1},
		{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 2},
	}
	periodSecs1 := uint32(12)
	targetPerSec1 := uint64(7_000_000)

	tx, err := arbOwner.SetResourceConstraint(&auth, rc1, periodSecs1, targetPerSec1)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Set second constraint
	rc2 := []precompilesgen.ArbResourceConstraintsTypesResourceWeight{
		{Resource: uint8(multigas.ResourceKindHistoryGrowth), Weight: 3},
		{Resource: uint8(multigas.ResourceKindWasmComputation), Weight: 4},
	}
	periodSecs2 := uint32(30)
	targetPerSec2 := uint64(9_000_000)

	tx, err = arbOwner.SetResourceConstraint(&auth, rc2, periodSecs2, targetPerSec2)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	list, err := arbGasInfo.ListResourceConstraints(callOpts)
	Require(t, err)
	if len(list) != 2 {
		t.Fatalf("expected 2 constraints, got %d", len(list))
	}

	// Verify first constraint
	c1 := list[0]
	if c1.PeriodSecs != periodSecs1 {
		t.Fatalf("expected periodSecs=%d got %d", periodSecs1, c1.PeriodSecs)
	}
	if c1.TargetPerSec != targetPerSec1 {
		t.Fatalf("expected targetPerSec=%d got %d", targetPerSec1, c1.TargetPerSec)
	}
	if len(c1.Resources) != len(rc1) {
		t.Fatalf("expected %d resources, got %d", len(rc1), len(c1.Resources))
	}
	for i := range rc1 {
		if c1.Resources[i].Resource != rc1[i].Resource ||
			c1.Resources[i].Weight != rc1[i].Weight {
			t.Fatalf("constraint1 resource[%d] mismatch: want %+v got %+v", i, rc1[i], c1.Resources[i])
		}
	}

	// Verify second constraint
	c2 := list[1]
	if c2.PeriodSecs != periodSecs2 {
		t.Fatalf("expected periodSecs=%d got %d", periodSecs2, c2.PeriodSecs)
	}
	if c2.TargetPerSec != targetPerSec2 {
		t.Fatalf("expected targetPerSec=%d got %d", targetPerSec2, c2.TargetPerSec)
	}
	if len(c2.Resources) != len(rc2) {
		t.Fatalf("expected %d resources, got %d", len(rc2), len(c2.Resources))
	}
	for i := range rc2 {
		if c2.Resources[i].Resource != rc2[i].Resource ||
			c2.Resources[i].Weight != rc2[i].Weight {
			t.Fatalf("constraint2 resource[%d] mismatch: want %+v got %+v", i, rc2[i], c2.Resources[i])
		}
	}
}

func TestUpdateExistingResourceConstraintPrecompile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	Require(t, err)

	// Set the initial constraint
	rc1 := []precompilesgen.ArbResourceConstraintsTypesResourceWeight{
		{Resource: uint8(multigas.ResourceKindComputation), Weight: 1},
		{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 2},
	}
	periodSecs := uint32(20)
	targetPerSec := uint64(8_000_000)

	tx, err := arbOwner.SetResourceConstraint(&auth, rc1, periodSecs, targetPerSec)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Update the existing constraint: same resources and period, different weights and target
	rc2 := []precompilesgen.ArbResourceConstraintsTypesResourceWeight{
		{Resource: uint8(multigas.ResourceKindComputation), Weight: 3},
		{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 5},
	}
	targetPerSec2 := uint64(9_000_000)

	tx, err = arbOwner.SetResourceConstraint(&auth, rc2, periodSecs, targetPerSec2)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Verify only one constraint exists, and it matches rc2 values
	list, err := arbGasInfo.ListResourceConstraints(callOpts)
	Require(t, err)
	if len(list) != 1 {
		t.Fatalf("expected 1 constraint after overwrite, got %d", len(list))
	}

	got := list[0]
	if got.PeriodSecs != periodSecs {
		t.Fatalf("expected periodSecs=%d got %d", periodSecs, got.PeriodSecs)
	}
	if got.TargetPerSec != targetPerSec2 {
		t.Fatalf("expected targetPerSec=%d got %d", targetPerSec2, got.TargetPerSec)
	}
	if len(got.Resources) != len(rc2) {
		t.Fatalf("expected %d resources, got %d", len(rc2), len(got.Resources))
	}
	for i := range rc2 {
		if got.Resources[i].Resource != rc2[i].Resource ||
			got.Resources[i].Weight != rc2[i].Weight {
			t.Fatalf("resource[%d] mismatch after update: want %+v got %+v", i, rc2[i], got.Resources[i])
		}
	}
}
