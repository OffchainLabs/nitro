// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func setupArbGasInfo(
	t *testing.T,
) (
	*vm.EVM,
	*arbosState.ArbosState,
	*Context,
	*ArbGasInfo,
) {
	evm := newMockEVMForTesting()
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tracer := util.NewTracingInfo(evm, testhelpers.RandomAddress(), types.ArbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracer, false))
	Require(t, err)

	arbGasInfo := &ArbGasInfo{}
	callCtx := testContext(caller, evm)

	return evm, state, callCtx, arbGasInfo
}

func TestGetGasBacklog(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, arbGasInfo := setupArbGasInfo(t)

	backlog := uint64(1000)
	err := state.L2PricingState().SetGasBacklog(backlog)
	Require(t, err)
	retrievedBacklog, err := arbGasInfo.GetGasBacklog(callCtx, evm)
	Require(t, err)
	if retrievedBacklog != backlog {
		t.Fatal("expected backlog to be", backlog, "but got", retrievedBacklog)
	}
}

func TestGetL1PricingUpdateTime(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, arbGasInfo := setupArbGasInfo(t)

	lastUpdateTime := uint64(1001)
	err := state.L1PricingState().SetLastUpdateTime(lastUpdateTime)
	Require(t, err)
	retrievedLastUpdateTime, err := arbGasInfo.GetLastL1PricingUpdateTime(callCtx, evm)
	Require(t, err)
	if retrievedLastUpdateTime != lastUpdateTime {
		t.Fatal("expected last update time to be", lastUpdateTime, "but got", retrievedLastUpdateTime)
	}
}

func TestGetL1PricingFundsDueForRewards(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, arbGasInfo := setupArbGasInfo(t)

	fundsDueForRewards := big.NewInt(1002)
	err := state.L1PricingState().SetFundsDueForRewards(fundsDueForRewards)
	Require(t, err)
	retrievedFundsDueForRewards, err := arbGasInfo.GetL1PricingFundsDueForRewards(callCtx, evm)
	Require(t, err)
	if retrievedFundsDueForRewards.Cmp(fundsDueForRewards) != 0 {
		t.Fatal("expected funds due for rewards to be", fundsDueForRewards, "but got", retrievedFundsDueForRewards)
	}
}

func TestGetL1PricingUnitsSinceUpdate(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, arbGasInfo := setupArbGasInfo(t)

	pricingUnitsSinceUpdate := uint64(1003)
	err := state.L1PricingState().SetUnitsSinceUpdate(pricingUnitsSinceUpdate)
	Require(t, err)
	retrievedPricingUnitsSinceUpdate, err := arbGasInfo.GetL1PricingUnitsSinceUpdate(callCtx, evm)
	Require(t, err)
	if retrievedPricingUnitsSinceUpdate != pricingUnitsSinceUpdate {
		t.Fatal("expected pricing units since update to be", pricingUnitsSinceUpdate, "but got", retrievedPricingUnitsSinceUpdate)
	}
}

func TestGetLastL1PricingSurplus(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, arbGasInfo := setupArbGasInfo(t)

	lastSurplus := big.NewInt(1004)
	err := state.L1PricingState().SetLastSurplus(lastSurplus, params.ArbosVersion_Stylus)
	Require(t, err)
	retrievedLastSurplus, err := arbGasInfo.GetLastL1PricingSurplus(callCtx, evm)
	Require(t, err)
	if retrievedLastSurplus.Cmp(lastSurplus) != 0 {
		t.Fatal("expected last surplus to be", lastSurplus, "but got", retrievedLastSurplus)
	}
}

func TestGetPricesInArbGas(t *testing.T) {
	t.Parallel()

	evm := newMockEVMForTesting()
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	arbGasInfo := &ArbGasInfo{}
	callCtx := testContext(caller, evm)

	evm.Context.BaseFee = big.NewInt(1005)
	expectedGasPerL2Tx := big.NewInt(111442786069)
	expectedGasForL1Calldata := big.NewInt(796019900)
	expectedStorageArbGas := big.NewInt(int64(storage.StorageWriteCost))
	gasPerL2Tx, gasForL1Calldata, storageArbGas, err := arbGasInfo.GetPricesInArbGas(callCtx, evm)
	Require(t, err)
	if gasPerL2Tx.Cmp(expectedGasPerL2Tx) != 0 {
		t.Fatal("expected gas per L2 tx to be", expectedGasPerL2Tx, "but got", gasPerL2Tx)
	}
	if gasForL1Calldata.Cmp(expectedGasForL1Calldata) != 0 {
		t.Fatal("expected gas for L1 calldata to be", expectedGasForL1Calldata, "but got", gasForL1Calldata)
	}
	if storageArbGas.Cmp(expectedStorageArbGas) != 0 {
		t.Fatal("expected storage arb gas to be", expectedStorageArbGas, "but got", storageArbGas)
	}
}
