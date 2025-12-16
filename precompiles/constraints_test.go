// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func setupResourceConstraintHandles(
	t *testing.T,
) (
	*vm.EVM,
	*arbosState.ArbosState,
	*Context,
	*ArbGasInfo,
	*ArbOwner,
) {
	t.Helper()

	evm := newMockEVMForTesting()
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tracer := util.NewTracingInfo(evm, testhelpers.RandomAddress(), types.ArbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracer, false))
	require.NoError(t, err)

	state.L2PricingState().ArbosVersion = l2pricing.ArbosMultiGasConstraintsVersion

	arbGasInfo := &ArbGasInfo{}
	arbOwner := &ArbOwner{}

	callCtx := testContext(caller, evm)

	return evm, state, callCtx, arbGasInfo, arbOwner
}

func TestFailToSetInvalidConstraints(t *testing.T) {
	t.Parallel()

	evm, _, callCtx, _, arbOwner := setupResourceConstraintHandles(t)

	// Zero target
	err := arbOwner.SetGasPricingConstraints(callCtx, evm, [][3]uint64{{0, 17, 1000}})
	require.Error(t, err)

	// Zero adjustment window
	err = arbOwner.SetGasPricingConstraints(callCtx, evm, [][3]uint64{{10_000_000, 0, 0}})
	require.Error(t, err)
}

func TestSetLegacyBacklog(t *testing.T) {
	t.Parallel()

	evm, _, callCtx, arbGasInfo, arbOwner := setupResourceConstraintHandles(t)

	backlog, err := arbGasInfo.GetGasBacklog(callCtx, evm)
	require.NoError(t, err)
	require.Equal(t, uint64(0), backlog)

	newBacklog := uint64(80_000)
	err = arbOwner.SetGasBacklog(callCtx, evm, newBacklog)
	require.NoError(t, err)

	backlog, err = arbGasInfo.GetGasBacklog(callCtx, evm)
	require.NoError(t, err)
	require.Equal(t, newBacklog, backlog)
}

func TestConstraintsStorage(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, arbGasInfo, arbOwner := setupResourceConstraintHandles(t)

	// Set constraints
	constraints := [][3]uint64{
		{30_000_000, 1, 800_000},     // short-term
		{15_000_000, 102, 1_600_000}, // long-term
	}
	err := arbOwner.SetGasPricingConstraints(callCtx, evm, constraints)
	require.NoError(t, err)

	// Verify constraints are stored correctly
	length, err := state.L2PricingState().GasConstraintsLength()
	require.NoError(t, err)
	require.Equal(t, uint64(2), length)

	first := state.L2PricingState().OpenGasConstraintAt(0)
	second := state.L2PricingState().OpenGasConstraintAt(1)

	firstTarget, err := first.Target()
	require.NoError(t, err)
	require.Equal(t, uint64(30_000_000), firstTarget)

	firstWindow, err := first.AdjustmentWindow()
	require.NoError(t, err)
	require.Equal(t, uint64(1), firstWindow)

	firstBacklog, err := first.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(800_000), firstBacklog)

	secondTarget, err := second.Target()
	require.NoError(t, err)
	require.Equal(t, uint64(15_000_000), secondTarget)

	secondWindow, err := second.AdjustmentWindow()
	require.NoError(t, err)
	require.Equal(t, uint64(102), secondWindow)

	secondBacklog, err := second.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(1_600_000), secondBacklog)

	// Get constraints and verify
	result, err := arbGasInfo.GetGasPricingConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Equal(t, 2, len(result))
	require.Equal(t, uint64(30_000_000), result[0][0])
	require.Equal(t, uint64(1), result[0][1])
	require.Equal(t, uint64(15_000_000), result[1][0])
	require.Equal(t, uint64(102), result[1][1])

	// Set new constraints
	constraints = [][3]uint64{
		{7_000_000, 12, 50_000_000},
	}
	err = arbOwner.SetGasPricingConstraints(callCtx, evm, constraints)
	require.NoError(t, err)

	// Verify old constraints are cleared and new constraint is stored correctly
	length, err = state.L2PricingState().GasConstraintsLength()
	require.NoError(t, err)
	require.Equal(t, uint64(1), length)

	first = state.L2PricingState().OpenGasConstraintAt(0)
	target, err := first.Target()
	require.NoError(t, err)
	window, err := first.AdjustmentWindow()
	require.NoError(t, err)
	require.Equal(t, uint64(7_000_000), target)
	require.Equal(t, uint64(12), window)

	result, err = arbGasInfo.GetGasPricingConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Equal(t, len(result), 1)
	require.Equal(t, result[0][0], uint64(7_000_000))
	require.Equal(t, result[0][1], uint64(12))
	require.Equal(t, result[0][2], uint64(50_000_000))
}

func TestConstraintsBacklogUpdate(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, arbGasInfo, arbOwner := setupResourceConstraintHandles(t)

	// Set constraints
	constraints := [][3]uint64{
		{30_000_000, 1, 0},        // short-term
		{15_000_000, 86400, 8000}, // long-term
	}
	err := arbOwner.SetGasPricingConstraints(callCtx, evm, constraints)
	require.NoError(t, err)

	err = state.L2PricingState().OpenGasConstraintAt(0).SetBacklog(5_000_000)
	require.NoError(t, err)
	err = state.L2PricingState().OpenGasConstraintAt(1).SetBacklog(10_000_000)
	require.NoError(t, err)

	// Verify backlogs are updated correctly
	result, err := arbGasInfo.GetGasPricingConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Equal(t, 2, len(result))
	require.Equal(t, uint64(5_000_000), result[0][2])
	require.Equal(t, uint64(10_000_000), result[1][2])
}

func TestEnableAndDisableMultiConstraints(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, _, arbOwner := setupResourceConstraintHandles(t)

	// Initially single-gas constraints should be disabled
	gasModel, err := state.L2PricingState().GasModelToUse()
	require.NoError(t, err)
	require.Equal(t, l2pricing.GasModelLegacy, gasModel)

	// Set gas constraints to enable single-gas constraints
	constraints := [][3]uint64{
		{30_000_000, 1, 800_000},
		{15_000_000, 102, 1_600_000},
	}
	err = arbOwner.SetGasPricingConstraints(callCtx, evm, constraints)
	require.NoError(t, err)

	gasModel, err = state.L2PricingState().GasModelToUse()
	require.NoError(t, err)
	require.Equal(t, l2pricing.GasModelSingleGasConstraints, gasModel)

	// Set multi-gas constraints to enable multi-gas constraints
	mgConstraints := []MultiGasConstraint{
		{
			Resources: []WeightedResource{
				{Resource: uint8(multigas.ResourceKindComputation), Weight: 5},
				{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 7},
			},
			AdjustmentWindowSecs: 12,
			TargetPerSec:         7_000_000,
			Backlog:              50_000_000,
		},
	}

	err = arbOwner.SetMultiGasPricingConstraints(callCtx, evm, mgConstraints)
	require.NoError(t, err)

	gasModel, err = state.L2PricingState().GasModelToUse()
	require.NoError(t, err)
	require.Equal(t, l2pricing.GasModelMultiGasConstraints, gasModel)

	// Clear multi-gas constraints to disable multi-gas constraints
	err = arbOwner.SetMultiGasPricingConstraints(callCtx, evm, []MultiGasConstraint{})
	require.NoError(t, err)

	gasModel, err = state.L2PricingState().GasModelToUse()
	require.NoError(t, err)
	require.Equal(t, l2pricing.GasModelSingleGasConstraints, gasModel)

	// Clear gas constraints to disable single-gas constraints
	err = arbOwner.SetGasPricingConstraints(callCtx, evm, [][3]uint64{})
	require.NoError(t, err)

	gasModel, err = state.L2PricingState().GasModelToUse()
	require.NoError(t, err)
	require.Equal(t, l2pricing.GasModelLegacy, gasModel)
}

func TestMultiGasConstraintsStorage(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, arbGasInfo, arbOwner := setupResourceConstraintHandles(t)

	// Set two multi-gas constraints
	constraints := []MultiGasConstraint{
		{
			Resources: []WeightedResource{
				{Resource: uint8(multigas.ResourceKindComputation), Weight: 1},
				{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 2},
			},
			AdjustmentWindowSecs: 1,
			TargetPerSec:         30_000_000,
			Backlog:              800_000,
		},
		{
			Resources: []WeightedResource{
				{Resource: uint8(multigas.ResourceKindComputation), Weight: 2},
				{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 3},
			},
			AdjustmentWindowSecs: 102,
			TargetPerSec:         15_000_000,
			Backlog:              1_600_000,
		},
	}

	err := arbOwner.SetMultiGasPricingConstraints(callCtx, evm, constraints)
	require.NoError(t, err)

	length, err := state.L2PricingState().MultiGasConstraintsLength()
	require.NoError(t, err)
	require.Equal(t, uint64(2), length)

	first := state.L2PricingState().OpenMultiGasConstraintAt(0)
	second := state.L2PricingState().OpenMultiGasConstraintAt(1)

	// First constraint
	target, err := first.Target()
	require.NoError(t, err)
	require.Equal(t, uint64(30_000_000), target)
	window, err := first.AdjustmentWindow()
	require.NoError(t, err)
	require.Equal(t, uint32(1), window)
	backlog, err := first.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(800_000), backlog)

	resMap, err := first.ResourcesWithWeights()
	require.NoError(t, err)
	require.Equal(t, uint64(1), resMap[multigas.ResourceKindComputation])
	require.Equal(t, uint64(2), resMap[multigas.ResourceKindStorageAccess])

	// Second constraint
	target, err = second.Target()
	require.NoError(t, err)
	require.Equal(t, uint64(15_000_000), target)
	window, err = second.AdjustmentWindow()
	require.NoError(t, err)
	require.Equal(t, uint32(102), window)
	backlog, err = second.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(1_600_000), backlog)

	resMap, err = second.ResourcesWithWeights()
	require.NoError(t, err)
	require.Equal(t, uint64(2), resMap[multigas.ResourceKindComputation])
	require.Equal(t, uint64(3), resMap[multigas.ResourceKindStorageAccess])

	// Verify via getter precompile
	results, err := arbGasInfo.GetMultiGasPricingConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Equal(t, 2, len(results))

	require.Equal(t, uint32(1), results[0].AdjustmentWindowSecs)
	require.Equal(t, uint64(30_000_000), results[0].TargetPerSec)
	require.Equal(t, uint64(800_000), results[0].Backlog)
	require.Equal(t, 2, len(results[0].Resources))

	require.Equal(t, uint32(102), results[1].AdjustmentWindowSecs)
	require.Equal(t, uint64(15_000_000), results[1].TargetPerSec)
	require.Equal(t, uint64(1_600_000), results[1].Backlog)
	require.Equal(t, 2, len(results[1].Resources))

	// Replace with a single new constraint
	newConstraints := []MultiGasConstraint{
		{
			Resources: []WeightedResource{
				{Resource: uint8(multigas.ResourceKindComputation), Weight: 5},
				{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 7},
			},
			AdjustmentWindowSecs: 12,
			TargetPerSec:         7_000_000,
			Backlog:              50_000_000,
		},
	}

	err = arbOwner.SetMultiGasPricingConstraints(callCtx, evm, newConstraints)
	require.NoError(t, err)

	length, err = state.L2PricingState().MultiGasConstraintsLength()
	require.NoError(t, err)
	require.Equal(t, uint64(1), length)

	first = state.L2PricingState().OpenMultiGasConstraintAt(0)
	target, err = first.Target()
	require.NoError(t, err)
	require.Equal(t, uint64(7_000_000), target)
	window, err = first.AdjustmentWindow()
	require.NoError(t, err)
	require.Equal(t, uint32(12), window)
	backlog, err = first.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(50_000_000), backlog)

	resMap, err = first.ResourcesWithWeights()
	require.NoError(t, err)
	require.Equal(t, uint64(5), resMap[multigas.ResourceKindComputation])
	require.Equal(t, uint64(7), resMap[multigas.ResourceKindStorageAccess])

	results, err = arbGasInfo.GetMultiGasPricingConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	require.Equal(t, uint32(12), results[0].AdjustmentWindowSecs)
	require.Equal(t, uint64(7_000_000), results[0].TargetPerSec)
	require.Equal(t, uint64(50_000_000), results[0].Backlog)
	require.Equal(t, 2, len(results[0].Resources))
	require.Equal(t, uint8(multigas.ResourceKindComputation), results[0].Resources[0].Resource)
	require.Equal(t, uint64(5), results[0].Resources[0].Weight)
	require.Equal(t, uint8(multigas.ResourceKindStorageAccess), results[0].Resources[1].Resource)
	require.Equal(t, uint64(7), results[0].Resources[1].Weight)
}

func TestMultiGasConstraintsCantExceedLimit(t *testing.T) {
	t.Parallel()

	evm, _, callCtx, _, arbOwner := setupResourceConstraintHandles(t)

	// Try to set a constraint that exceeds the MaxPricingExponentBips
	constraints := []MultiGasConstraint{
		{
			Resources: []WeightedResource{
				{Resource: uint8(multigas.ResourceKindComputation), Weight: 1},
				{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 2},
			},
			AdjustmentWindowSecs: 1,
			TargetPerSec:         30_000_000,
			Backlog:              800_000_000_000,
		},
	}

	err := arbOwner.SetMultiGasPricingConstraints(callCtx, evm, constraints)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds maximum allowed")
}
