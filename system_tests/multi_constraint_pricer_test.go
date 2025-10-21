// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestMultiConstraintPricerPrecompiles(t *testing.T) {
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
		{30_000_000, 1, 800_000},       // short-term
		{15_000_000, 86400, 1_600_000}, // long-term
	}
	tx, err := arbOwner.SetGasPricingConstraints(&auth, constraints)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Get and check values
	constraints, err = arbGasInfo.GetGasPricingConstraints(callOpts)
	require.NoError(t, err)
	require.Equal(t, 2, len(constraints))
	require.Equal(t, uint64(30_000_000), constraints[0][0])
	require.Equal(t, uint64(1), constraints[0][1])
	require.GreaterOrEqual(t, constraints[0][2], uint64(800_000))
	require.Equal(t, uint64(15_000_000), constraints[1][0])
	require.Equal(t, uint64(86400), constraints[1][1])
	require.GreaterOrEqual(t, constraints[1][2], uint64(1_600_000))

	// Should return speed limit derived from long-term constraint
	speedLimit, gasPoolMax, _, err := arbGasInfo.GetGasAccountingParams(callOpts)
	require.NoError(t, err)
	require.Zero(t, big.NewInt(15_000_000).Cmp(speedLimit))
	require.Zero(t, big.NewInt(0).Cmp(gasPoolMax))

	// Should return the long-term constraint's backlog (the highest-period constraint)
	gasBacklog, err := arbGasInfo.GetGasBacklog(callOpts)
	require.NoError(t, err)
	require.GreaterOrEqual(t, gasBacklog, uint64(1_600_000))

	// Should return the long-term constraint's computed inertia
	pricingInertia, err := arbGasInfo.GetPricingInertia(callOpts)
	require.NoError(t, err)
	require.GreaterOrEqual(t, pricingInertia, uint64(86400))

	// Should return zero backlog tolerance
	backlogTolerance, err := arbGasInfo.GetGasBacklogTolerance(callOpts)
	require.NoError(t, err)
	require.Equal(t, uint64(0), backlogTolerance)

	// Check SetSpeedLimit returns error
	_, err = arbOwner.SetSpeedLimit(&auth, 20_000_000)
	require.Error(t, err)

	// Check SetL2GasPricingInertia returns error
	_, err = arbOwner.SetL2GasPricingInertia(&auth, 200)
	require.Error(t, err)

	// Check SetL2GasBacklogTolerance returns error
	_, err = arbOwner.SetL2GasBacklogTolerance(&auth, 100)
	require.Error(t, err)
}

func TestMultiConstraintPricerMigration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSVersion(params.ArbosVersion_41)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	require.NoError(t, err)
	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	require.NoError(t, err)

	// Check version before upgrade
	arbosVersion, err := arbSys.ArbOSVersion(callOpts)
	require.NoError(t, err)
	require.Equal(t, arbosVersion.Uint64()-55, params.ArbosVersion_41)

	// Set legacy parameters
	_, err = arbOwner.SetSpeedLimit(&auth, 20_000_000)
	require.NoError(t, err)

	_, err = arbOwner.SetL2GasPricingInertia(&auth, 300)
	require.NoError(t, err)

	_, err = arbOwner.SetL2GasBacklogTolerance(&auth, 150)
	require.NoError(t, err)

	// Check GetGasAccountingParams returns expected speed limit
	speedLimit, gasPoolMax, _, err := arbGasInfo.GetGasAccountingParams(callOpts)
	require.NoError(t, err)
	require.Zero(t, big.NewInt(20_000_000).Cmp(speedLimit))
	require.Zero(t, big.NewInt(0).Cmp(gasPoolMax))

	// Check GetPricingInertia returns expected inertia
	pricingInertia, err := arbGasInfo.GetPricingInertia(callOpts)
	require.NoError(t, err)
	require.Equal(t, uint64(300), pricingInertia)

	// Check GetGasBacklogTolerance returns expected tolerance
	backlogTolerance, err := arbGasInfo.GetGasBacklogTolerance(callOpts)
	require.NoError(t, err)
	require.Equal(t, uint64(150), backlogTolerance)

	// Upgrade to ArbOS version 50
	tx, err := arbOwner.ScheduleArbOSUpgrade(&auth, params.ArbosVersion_50, 0)
	require.NoError(t, err)

	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Send a dummy transaction to trigger the upgrade
	var data []byte
	for i := range 10 {
		for range 100 {
			data = append(data, byte(i))
		}
	}
	tx = builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, big.NewInt(1e12), data)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	arbosVersion, err = arbSys.ArbOSVersion(callOpts)
	require.NoError(t, err)
	require.GreaterOrEqual(t, arbosVersion.Uint64()-55, params.ArbosVersion_50)

	// Check SetSpeedLimit returns error after upgrade
	_, err = arbOwner.SetSpeedLimit(&auth, 25_000_000)
	require.Error(t, err)

	// Check GetPricingInertia returns expected inertia
	pricingInertia, err = arbGasInfo.GetPricingInertia(callOpts)
	require.NoError(t, err)
	require.Equal(t, uint64(300), pricingInertia)

	// Get and check pricing constraints after upgrade
	constraints, err := arbGasInfo.GetGasPricingConstraints(callOpts)
	require.NoError(t, err)
	require.Equal(t, 1, len(constraints))
	require.Equal(t, uint64(20_000_000), constraints[0][0])
	require.Equal(t, uint64(100), constraints[0][1])
	require.GreaterOrEqual(t, constraints[0][2], uint64(0))

	// Check GetGasAccountingParams after upgrade
	speedLimit, _, _, err = arbGasInfo.GetGasAccountingParams(callOpts)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(20_000_000), speedLimit)
}
