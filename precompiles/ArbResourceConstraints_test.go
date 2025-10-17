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
	evm := newMockEVMForTesting()
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tracer := util.NewTracingInfo(evm, testhelpers.RandomAddress(), types.ArbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracer, false))
	Require(t, err)

	arbGasInfo := &ArbGasInfo{}
	arbOwner := &ArbOwner{}

	callCtx := testContext(caller, evm)

	return evm, state, callCtx, arbGasInfo, arbOwner
}

func TestStorageResourceConstraintsManagment(t *testing.T) {
	t.Parallel()

	evm, _, callCtx, arbGasInfo, arbOwner := setupResourceConstraintHandles(t)

	// Expect error when setting empty resource constraints
	err := arbOwner.SetResourceConstraint(callCtx, evm, []resourceWeight{}, 0, 0)
	require.Error(t, err)

	// Expect error when setting resource constraint with invalid weight
	err = arbOwner.SetResourceConstraint(callCtx, evm, []resourceWeight{
		{Resource: uint8(multigas.ResourceKindComputation), Weight: 0},
	}, 0, 0)
	require.Error(t, err)

	// Expect error when setting resource constraint with duplicate resource kinds
	err = arbOwner.SetResourceConstraint(callCtx, evm, []resourceWeight{
		{Resource: uint8(multigas.ResourceKindComputation), Weight: 1},
		{Resource: uint8(multigas.ResourceKindComputation), Weight: 2},
	}, 0, 0)
	require.Error(t, err)

	// Check that no resource constraints are set initially
	rcs, err := arbGasInfo.ListResourceConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Empty(t, rcs)

	// Set a valid resource constraint
	rc1 := []resourceWeight{
		{Resource: uint8(multigas.ResourceKindComputation), Weight: 1},
		{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 2},
	}
	err = arbOwner.SetResourceConstraint(callCtx, evm, rc1, 12, 10_000_000)
	require.NoError(t, err)

	// Verify the resource constraint was set
	rcs, err = arbGasInfo.ListResourceConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Len(t, rcs, 1)
	require.Equal(t, uint32(12), rcs[0].PeriodSecs)
	require.Equal(t, uint64(10_000_000), rcs[0].TargetPerSec)
	require.Len(t, rcs[0].Resources, 2)
	require.Contains(t, rcs[0].Resources, resourceWeight{Resource: uint8(multigas.ResourceKindComputation), Weight: 1})
	require.Contains(t, rcs[0].Resources, resourceWeight{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 2})

	// Set another valid resource constraint
	rc2 := []resourceWeight{
		{Resource: uint8(multigas.ResourceKindStorageGrowth), Weight: 5},
	}
	err = arbOwner.SetResourceConstraint(callCtx, evm, rc2, 60, 50_000_000)
	require.NoError(t, err)

	// Verify both resource constraints are set
	rcs, err = arbGasInfo.ListResourceConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Len(t, rcs, 2)

	// Remove the first resource constraint
	rc1Kinds := []uint8{
		uint8(multigas.ResourceKindStorageAccess),
		uint8(multigas.ResourceKindComputation),
	}
	err = arbOwner.ClearConstraint(callCtx, evm, rc1Kinds, 12)
	require.NoError(t, err)

	// Reset the second resource constraint with new values
	err = arbOwner.SetResourceConstraint(callCtx, evm, rc2, 60, 20_000_000)
	require.NoError(t, err)

	// Verify only the updated second resource constraint remains
	rcs, err = arbGasInfo.ListResourceConstraints(callCtx, evm)
	require.NoError(t, err)
	require.Len(t, rcs, 1)
	require.Equal(t, uint32(60), rcs[0].PeriodSecs)
	require.Equal(t, uint64(20_000_000), rcs[0].TargetPerSec)
	require.Len(t, rcs[0].Resources, 1)
	require.Contains(t, rcs[0].Resources, resourceWeight{Resource: uint8(multigas.ResourceKindStorageGrowth), Weight: 5})
}
