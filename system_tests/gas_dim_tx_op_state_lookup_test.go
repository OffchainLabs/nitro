package arbtest

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/gasdimensionsgen"
)

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
func TestDimTxOpBalanceCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployBalance)
	tx, receipt := callOnContract(t, builder, auth, contract.CallBalanceCold)
	intrinsicTxCost := getIntrinsicTxCost(t, tx)
	traceResult := callDebugTraceTransactionWithTxGasDimensionByOpcodeTracer(t, ctx, builder, receipt.TxHash)

	expectedGasUsed := receipt.GasUsedForL2()
	sumOneDimensionalGasCosts, sumAllGasCosts := sumUpDimensionalGasCosts(t, traceResult.Dimensions)
	sumOneDimensionalGasCosts += intrinsicTxCost
	sumAllGasCosts += intrinsicTxCost

	CheckEqual(t, expectedGasUsed, sumOneDimensionalGasCosts,
		fmt.Sprintf("expected gas used: %d, one-dim used: %d\nDimensions:\n%v", expectedGasUsed, sumOneDimensionalGasCosts, traceResult.Dimensions))
	CheckEqual(t, expectedGasUsed, sumAllGasCosts,
		fmt.Sprintf("expected gas used: %d, all used: %d\nDimensions:\n%v", expectedGasUsed, sumAllGasCosts, traceResult.Dimensions))
}

// this test deployes a contract that calls BALANCE on a warm access list address
//
// on the warm BALANCE, we expect the total one-dimensional gas cost to be 100
// the computation to be 100 (for the warm access cost of the address)
// and all other gas dimensions to be 0
func TestDimTxOpBalanceWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployBalance)
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
	checkDimensionLogGasCostsEqual(
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
func TestDimTxOpExtCodeSizeCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeSize)
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
	checkDimensionLogGasCostsEqual(
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
func TestDimTxOpExtCodeSizeWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeSize)
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
	checkDimensionLogGasCostsEqual(
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
func TestDimTxOpExtCodeHashCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeHash)
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
	checkDimensionLogGasCostsEqual(
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
func TestDimTxOpExtCodeHashWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeHash)
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
	checkDimensionLogGasCostsEqual(
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
func TestDimTxOpSloadCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySload)
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
	checkDimensionLogGasCostsEqual(
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
func TestDimTxOpSloadWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySload)
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
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		sloadLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, sloadLog)
}
