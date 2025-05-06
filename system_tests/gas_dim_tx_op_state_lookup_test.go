package arbtest

import (
	"fmt"
	"testing"

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
	tx, receipt := callOnContract(t, builder, auth, contract.CallBalanceWarm)
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
	tx, receipt := callOnContract(t, builder, auth, contract.GetExtCodeSizeCold)
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
	tx, receipt := callOnContract(t, builder, auth, contract.GetExtCodeSizeWarm)
	intrinsicTxCost := getIntrinsicTxCost(t, tx)
	traceResult := callDebugTraceTransactionWithTxGasDimensionByOpcodeTracer(t, ctx, builder, receipt.TxHash)

	expectedGasUsed := receipt.GasUsedForL2()
	sumOneDimensionalGasCosts, sumAllGasCosts := sumUpDimensionalGasCosts(t, traceResult.Dimensions)
	sumOneDimensionalGasCosts += intrinsicTxCost
	sumAllGasCosts += intrinsicTxCost

	CheckEqual(t, receipt.GasUsed, traceResult.GasUsed)
	CheckEqual(t, expectedGasUsed, sumAllGasCosts,
		fmt.Sprintf("expected gas used: %d, all used: %d\nDimensions:\n%v", expectedGasUsed, sumAllGasCosts, traceResult.Dimensions))
	CheckEqual(t, expectedGasUsed, sumOneDimensionalGasCosts,
		fmt.Sprintf("expected gas used: %d, one-dim used: %d\nDimensions:\n%v", expectedGasUsed, sumOneDimensionalGasCosts, traceResult.Dimensions))
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
	tx, receipt := callOnContract(t, builder, auth, contract.GetExtCodeHashCold)
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
	tx, receipt := callOnContract(t, builder, auth, contract.GetExtCodeHashWarm)
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
	tx, receipt := callOnContract(t, builder, auth, contract.ColdSload)
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
	tx, receipt := callOnContract(t, builder, auth, contract.WarmSload)
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
