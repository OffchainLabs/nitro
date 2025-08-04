// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

const ColdSloadCost = params.ColdSloadCostEIP2929

// #########################################################################################################
// #########################################################################################################
//                SIMPLE STATE ACCESS OPCODES (BALANCE, EXTCODESIZE, EXTCODEHASH)
// #########################################################################################################
// #########################################################################################################
// BALANCE, EXTCODESIZE, EXTCODEHASH are all read-only operations on state access
// They only operate on one axis:
// Warm vs Cold (in the access list)

// ############################################################
//                        BALANCE
// ############################################################

// this test deployes a contract that calls BALANCE on a cold access list address
//
// on the cold BALANCE, we expect the total one-dimensional gas cost to be 2600
// the computation to be 100 (for the warm access cost of the address)
// and the state access to be 2500 (for the cold access cost of the address)
// and all other gas dimensions to be 0
func TestDimLogBalanceCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployBalance)
	_, receipt := callOnContract(t, builder, auth, contract.CallBalanceCold)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	balanceLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "BALANCE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		balanceLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, balanceLog)
}

// this test deployes a contract that calls BALANCE on a warm access list address
//
// on the warm BALANCE, we expect the total one-dimensional gas cost to be 100
// the computation to be 100 (for the warm access cost of the address)
// and all other gas dimensions to be 0
func TestDimLogBalanceWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployBalance)
	_, receipt := callOnContract(t, builder, auth, contract.CallBalanceWarm)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	balanceLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "BALANCE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		balanceLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, balanceLog)
}

// ############################################################
//                        EXTCODESIZE
// ############################################################

// this test deployes a contract that calls EXTCODESIZE on a cold access list address
//
// on the cold EXTCODESIZE, we expect the total one-dimensional gas cost to be 2600
// the computation to be 100 (for the warm access cost of the address)
// and the state access to be 2500 (for the cold access cost of the address)
// and all other gas dimensions to be 0
func TestDimLogExtCodeSizeCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeSize)
	_, receipt := callOnContract(t, builder, auth, contract.GetExtCodeSizeCold)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeSizeLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODESIZE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		extCodeSizeLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeSizeLog)
}

// this test deployes a contract that calls EXTCODESIZE on a warm access list address
//
// on the warm EXTCODESIZE, we expect the total one-dimensional gas cost to be 100
// the computation to be 100 (for the warm access cost of the address)
// and all other gas dimensions to be 0
func TestDimLogExtCodeSizeWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeSize)
	_, receipt := callOnContract(t, builder, auth, contract.GetExtCodeSizeWarm)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeSizeLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODESIZE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		extCodeSizeLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeSizeLog)
}

// ############################################################
//                        EXTCODEHASH
// ############################################################

// this test deployes a contract that calls EXTCODEHASH on a cold access list address
//
// on the cold EXTCODEHASH, we expect the total one-dimensional gas cost to be 2600
// the computation to be 100 (for the warm access cost of the address)
// and the state access to be 2500 (for the cold access cost of the address)
// and all other gas dimensions to be 0
func TestDimLogExtCodeHashCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeHash)
	_, receipt := callOnContract(t, builder, auth, contract.GetExtCodeHashCold)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeHashLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODEHASH")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		extCodeHashLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeHashLog)
}

// this test deployes a contract that calls EXTCODEHASH on a warm access list address
//
// on the warm EXTCODEHASH, we expect the total one-dimensional gas cost to be 100
// the computation to be 100 (for the warm access cost of the address)
// and all other gas dimensions to be 0
func TestDimLogExtCodeHashWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeHash)
	_, receipt := callOnContract(t, builder, auth, contract.GetExtCodeHashWarm)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeHashLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODEHASH")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		extCodeHashLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeHashLog)
}

// ############################################################
//                        SLOAD
// ############################################################
// SLOAD has only one axis of variation:
// Warm vs Cold (in the access list)

// In this test we deploy a contract with a function that all it does
// is perform an sload on a cold slot that has not been touched yet
//
// on the cold sload, we expect the total one-dimensional gas cost to be 2100
// the computation to be 100 (for the warm base access cost)
// the state access to be 2000 (for the cold sload cost)
// all others zero
func TestDimLogSloadCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySload)
	_, receipt := callOnContract(t, builder, auth, contract.ColdSload)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sloadLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SLOAD")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdSloadCost,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmSloadCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		sloadLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, sloadLog)
}

// In this test we deploy a contract with a function that all it does
// is perform an sload on an already warm slot (by SSTORE-ing to the slot first)
//
// on the warm sload, we expect the total one-dimensional gas cost to be 100
// the computation to be 100 (for the warm base access cost)
// all others zero
func TestDimLogSloadWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySload)
	_, receipt := callOnContract(t, builder, auth, contract.WarmSload)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sloadLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SLOAD")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		sloadLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, sloadLog)
}
