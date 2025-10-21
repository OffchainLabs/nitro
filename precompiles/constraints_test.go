// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
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

	arbGasInfo := &ArbGasInfo{}
	arbOwner := &ArbOwner{}

	callCtx := testContext(caller, evm)

	return evm, state, callCtx, arbGasInfo, arbOwner
}

func TestFailToSetInvalidConstraints(t *testing.T) {
	t.Parallel()

	evm, _, callCtx, _, arbOwner := setupResourceConstraintHandles(t)

	// Empty constraints
	err := arbOwner.SetGasPricingConstraints(callCtx, evm, [][3]uint64{})
	require.Error(t, err)

	// Zero target
	err = arbOwner.SetGasPricingConstraints(callCtx, evm, [][3]uint64{{0, 17, 1000}})
	require.Error(t, err)

	// Zero period
	err = arbOwner.SetGasPricingConstraints(callCtx, evm, [][3]uint64{{10_000_000, 0, 0}})
	require.Error(t, err)
}

func TestConstraintsStorage(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, arbGasInfo, arbOwner := setupResourceConstraintHandles(t)

	// Set constraints
	constraints := [][3]uint64{
		{30_000_000, 1, 800_000},       // short-term
		{15_000_000, 86400, 1_600_000}, // long-term
	}
	err := arbOwner.SetGasPricingConstraints(callCtx, evm, constraints)
	require.NoError(t, err)

	// Verify constraints are stored correctly
	length, err := state.L2PricingState().ConstraintsLength()
	require.NoError(t, err)
	require.Equal(t, uint64(2), length)

	first := state.L2PricingState().OpenConstraintAt(0)
	second := state.L2PricingState().OpenConstraintAt(1)

	firstTarget, err := first.Target()
	require.NoError(t, err)
	require.Equal(t, uint64(30_000_000), firstTarget)

	firstPeriod, err := first.Period()
	require.NoError(t, err)
	require.Equal(t, uint64(1), firstPeriod)

	firstBacklog, err := first.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(800_000), firstBacklog)

	secondTarget, err := second.Target()
	require.NoError(t, err)
	require.Equal(t, uint64(15_000_000), secondTarget)

	secondPeriod, err := second.Period()
	require.NoError(t, err)
	require.Equal(t, uint64(86400), secondPeriod)

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
	require.Equal(t, uint64(86400), result[1][1])

	// Set new constraints
	constraints = [][3]uint64{
		{7_000_000, 12, 50_000_000},
	}
	err = arbOwner.SetGasPricingConstraints(callCtx, evm, constraints)
	require.NoError(t, err)

	// Verify old constraints are cleared and new constraint is stored correctly
	length, err = state.L2PricingState().ConstraintsLength()
	require.NoError(t, err)
	require.Equal(t, uint64(1), length)

	first = state.L2PricingState().OpenConstraintAt(0)
	target, err := first.Target()
	require.NoError(t, err)
	period, err := first.Period()
	require.NoError(t, err)
	require.Equal(t, uint64(7_000_000), target)
	require.Equal(t, uint64(12), period)

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

	err = state.L2PricingState().OpenConstraintAt(0).SetBacklog(5_000_000)
	require.NoError(t, err)
	err = state.L2PricingState().OpenConstraintAt(1).SetBacklog(10_000_000)
	require.NoError(t, err)

	// Verify backlogs are updated correctly
	result, err := arbGasInfo.GetGasPricingConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Equal(t, 2, len(result))
	require.Equal(t, uint64(5_000_000), result[0][2])
	require.Equal(t, uint64(10_000_000), result[1][2])
}
