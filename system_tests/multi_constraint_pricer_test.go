// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestSetAndGetGasPricingConstraints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	require.NoError(t, err)

	// Set constraints
	constraints := [][3]uint64{
		{30_000_000, 102, 800_000},   // short-term
		{15_000_000, 600, 1_600_000}, // long-term
	}
	// TODO: SetGasPricingConstraints/GetGasPricingConstraints not in generated bindings yet
	// tx, err := arbOwner.SetGasPricingConstraints(&auth, constraints)
	// require.NoError(t, err)
	// require.NotNil(t, tx)

	// Get and check values
	// constraints, err = arbGasInfo.GetGasPricingConstraints(callOpts)
	// require.NoError(t, err)
	_ = auth       // TODO: Remove when API is available
	_ = callOpts   // TODO: Remove when API is available
	_ = arbOwner   // TODO: Remove when API is available
	_ = arbGasInfo // TODO: Remove when API is available
	t.Skip("SetGasPricingConstraints/GetGasPricingConstraints not in generated bindings yet")
	require.Equal(t, 2, len(constraints))
	require.Equal(t, uint64(30_000_000), constraints[0][0])
	require.Equal(t, uint64(102), constraints[0][1])
	require.GreaterOrEqual(t, constraints[0][2], uint64(800_000))
	require.Equal(t, uint64(15_000_000), constraints[1][0])
	require.Equal(t, uint64(600), constraints[1][1])
	require.GreaterOrEqual(t, constraints[1][2], uint64(1_600_000))
}
