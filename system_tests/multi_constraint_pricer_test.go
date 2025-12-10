// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestSetAndGetGasPricingConstraints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	require.NoError(t, err)

	// Set constraints
	constraints := [][3]uint64{
		{30_000_000, 102, 800_000},   // short-term
		{15_000_000, 600, 1_600_000}, // long-term
	}
	tx, err := arbOwner.SetGasPricingConstraints(&auth, constraints)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Get and check values
	constraints, err = arbGasInfo.GetGasPricingConstraints(callOpts)
	require.NoError(t, err)
	require.Equal(t, 2, len(constraints))
	require.Equal(t, uint64(30_000_000), constraints[0][0])
	require.Equal(t, uint64(102), constraints[0][1])
	require.GreaterOrEqual(t, constraints[0][2], uint64(800_000))
	require.Equal(t, uint64(15_000_000), constraints[1][0])
	require.Equal(t, uint64(600), constraints[1][1])
	require.GreaterOrEqual(t, constraints[1][2], uint64(1_600_000))
}

func TestSetAndGetMultiGasPricingConstraints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithArbOSVersion(l2pricing.ArbosMultiGasConstraintsVersion)

	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	require.NoError(t, err)

	constraint0 := precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		Resources: []precompilesgen.ArbMultiGasConstraintsTypesWeightedResource{
			{Resource: uint8(multigas.ResourceKindComputation), Weight: 3},
			{Resource: uint8(multigas.ResourceKindHistoryGrowth), Weight: 2},
			{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 1},
			{Resource: uint8(multigas.ResourceKindStorageGrowth), Weight: 4},
			{Resource: uint8(multigas.ResourceKindL1Calldata), Weight: 5},
		},
		AdjustmentWindowSecs: 102,
		TargetPerSec:         30_000_000,
		Backlog:              800_000,
	}

	constraint1 := precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		Resources: []precompilesgen.ArbMultiGasConstraintsTypesWeightedResource{
			{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 7},
			{Resource: uint8(multigas.ResourceKindL1Calldata), Weight: 9},
			{Resource: uint8(multigas.ResourceKindHistoryGrowth), Weight: 11},
		},
		AdjustmentWindowSecs: 600,
		TargetPerSec:         15_000_000,
		Backlog:              1_600_000,
	}

	constraints := []precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		constraint0,
		constraint1,
	}

	tx, err := arbOwner.SetMultiGasPricingConstraints(&auth, constraints)
	require.NoError(t, err)
	require.NotNil(t, tx)

	readBack, err := arbGasInfo.GetMultiGasPricingConstraints(callOpts)
	require.NoError(t, err)
	require.Equal(t, 2, len(readBack))

	toMap := func(list []precompilesgen.ArbMultiGasConstraintsTypesWeightedResource) map[uint8]uint64 {
		m := make(map[uint8]uint64, len(list))
		for _, r := range list {
			m[r.Resource] = r.Weight
		}
		return m
	}

	want0 := toMap(constraint0.Resources)
	want1 := toMap(constraint1.Resources)

	require.Equal(t, uint64(30_000_000), readBack[0].TargetPerSec)
	require.Equal(t, uint32(102), readBack[0].AdjustmentWindowSecs)
	require.GreaterOrEqual(t, readBack[0].Backlog, uint64(800_000))

	got0 := toMap(readBack[0].Resources)
	require.Equal(t, want0, got0)

	require.Equal(t, uint64(15_000_000), readBack[1].TargetPerSec)
	require.Equal(t, uint32(600), readBack[1].AdjustmentWindowSecs)
	require.GreaterOrEqual(t, readBack[1].Backlog, uint64(1_600_000))

	got1 := toMap(readBack[1].Resources)
	require.Equal(t, want1, got1)
}
