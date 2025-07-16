// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

// #########################################################################################################
// #########################################################################################################
//                                               SSTORE
// #########################################################################################################
// #########################################################################################################
//
// SSTORE has many permutations
// warm or cold
// 0 -> 0
// 0 -> non-zero
// non-zero -> 0
// non-zero -> non-zero (same value)
// non-zero -> non-zero (different value)
//
// Gas dimensionally, SSTORE has one rule basically: if the total gas is
// greater than 20,000 gas, then we know that a write of a 0->non-zero
// occurred. That is the only case that actually grows state database size
// all of the other cases are either writing to existing values, or
// writing to a local cache that will eventually be written to the existing
// database values.
// so the majority of cases for SSTORE fall under read/write, except for
// the one case where the database size grows.

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to 0
//
// we expect the gas cost of this operation to be 2200, 100 for the base sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 0, state access to be 2200, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreColdZeroToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdZeroToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdMinusWarmSloadCost + params.NetSstoreDirtyGas,
		Computation:           0,
		StateAccess:           ColdMinusWarmSloadCost + params.NetSstoreDirtyGas,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to a non-zero value
//
// we expect the gas cost of this operation to be 22100, 20000 for the sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 0, state read/write to be 2100, state growth to be 20000,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreColdZeroToNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdZeroToNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SstoreSetGasEIP2200 + params.ColdSloadCostEIP2929,
		Computation:           0,
		StateAccess:           params.ColdSloadCostEIP2929,
		StateGrowth:           params.SstoreSetGasEIP2200,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to 0
//
// we expect the gas cost of this operation to be 5000 and a gas refund of 4800.
// this is from an sstore cost of 2900, and a cold access set cost of 2100.
// we expect computation to be 100, state read/write to be 0, state growth to be 4900,
// history growth to be 0, and state growth refund to be 4800
func TestDimLogSstoreColdNonZeroValueToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroValueToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SstoreResetGasEIP2200,
		Computation:           0,
		StateAccess:           params.SstoreResetGasEIP2200,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     int64(params.NetSstoreResetRefund),
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// the same non-zero value
//
// we expect the gas cost of this operation to be 2200, 100 for the base sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 0, state read/write to be 2200, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreColdNonZeroToSameNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroToSameNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdSloadCostEIP2929 + params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.ColdSloadCostEIP2929 + params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// a different non-zero value
//
// we expect the gas cost of this operation to be 5000, 2900 for the sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 0, state read/write to be 5000, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreColdNonZeroToDifferentNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroToDifferentNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SstoreClearGas,
		Computation:           0,
		StateAccess:           params.SstoreClearGas,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to 0
//
// we expect the gas cost of this operation to be 100, 100 for the base sstore cost,
// we expect computation to be 0, state access to be 100, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreWarmZeroToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmZeroToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to a non-zero value
//
// we expect the gas cost of this operation to be 20000, 20000 for the sstore cost,
// we expect computation to be 0, state access to be 0, state growth to be 20000,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreWarmZeroToNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmZeroToNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SstoreSetGasEIP2200,
		Computation:           0,
		StateAccess:           0,
		StateGrowth:           params.SstoreSetGasEIP2200,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to 0
//
// we expect the gas cost of this operation to be 2900, with a gas refund of 4800
// This is 2900 just for the sstore cost
// we expect computation to be 0, state read/write to be 2900, state growth to be 0,
// history growth to be 0, and state growth refund to be 4800
func TestDimLogSstoreWarmNonZeroValueToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroValueToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: 2900, // there's no params reference for this, it just is 2900
		Computation:           0,
		StateAccess:           2900,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     int64(params.NetSstoreResetRefund),
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// the same non-zero value
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// we expect computation to be 0, state read/write to be 100, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreWarmNonZeroToSameNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroToSameNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// a different non-zero value
//
// we expect the gas cost of this operation to be 2900 for the base sstore cost,
// we expect computation to be 0, state read/write to be 2900, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreWarmNonZeroToDifferentNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroToDifferentNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: 2900, // there's no params reference for this, it just is 2900
		Computation:           0,
		StateAccess:           2900,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from non zero to non zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to a different non zero value
// for example, changing some value from 2->3->4
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreMultipleWarmNonZeroToNonZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToNonZeroToNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from non zero to non zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to the same non zero value
// for example, changing some value from 2->3->2
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// and to get a gas refund of 2800
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be 2800
func TestDimLogSstoreMultipleWarmNonZeroToNonZeroToSameNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToNonZeroToSameNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     2800, // i didn't see anything in params directly for this case
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from non zero to zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to a non zero value
// for example, changing some value from 2->0->3
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// and to have a NEGATIVE gas refund of -4800, i.e. taking away from the previous gas refund
// that was granted for changing the sstore from non zero to zero
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be -4800
func TestDimLogSstoreMultipleWarmNonZeroToZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToZeroToNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     -4800,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from non zero to zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to the same non zero value
// for example, changing some value from 2->0->2
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// and to have a NEGATIVE gas refund of -2000, i.e. taking away from the previous gas refund
// that was granted for changing the sstore from non zero to zero
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be -2000
func TestDimLogSstoreMultipleWarmNonZeroToZeroToSameNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToZeroToSameNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     -2000,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from zero to non zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to a different non zero value
// for example, changing some value from 0->2->3
//
// We expect the gas cost of this operation to be 100 for the base sstore cost,
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSstoreMultipleWarmZeroToNonZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmZeroToNonZeroToNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from zero to non zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, back to zero
// for example, changing some value from 0->3->0
//
// We expect the gas cost of this operation to be 100 for the base sstore cost,
// we expect to get a gas refund of 19900
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be 19900
func TestDimLogSstoreMultipleWarmZeroToNonZeroBackToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmZeroToNonZeroBackToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     19900,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}
