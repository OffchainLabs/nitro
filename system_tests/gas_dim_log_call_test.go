package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/gasdimensionsgen"
)

// this file tests the opcodes that are able to do write-operation CALLs
// looking for STATICCALL and DELEGATECALL? check out gas_dim_log_read_call_test.go

// #########################################################################################################
// #########################################################################################################
//                                             CALL and CALLCODE
// #########################################################################################################
// #########################################################################################################
//
// CALL and CALLCODE have many permutations /axes of variation
//
// Warm vs Cold (in the access list)
// Paying vs NoTransfer (is ether value being sent with this call?)
// Contract vs NoCode  (is there code at the target address that we are calling to? or is it an EOA?)
// Virgin vs Funded (has the target address ever been seen before / is the account already in the accounts tree?)
// MemExpansion or MemUnchanged (memory expansion or not)

// #########################################################################################################
// #########################################################################################################
//                                             CALL
// #########################################################################################################
// #########################################################################################################

// this test is the case where we do a CALL to a target contract
// that does not have any code or value already in the account
// the contract is cold in the access list, we send no eth value with the call
// and there is no memory expansion in this case.
//
// we expect the one dimensional gas to be 2600 because of the cold account access cost
// the computation to be 100 for the warm storage read cost
// the state access to be 2500, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallColdNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract that does have code
// the contract is cold in the access list, we send no eth value with the call
// and there is no memory expansion in this case.
//
// we expect the one dimensional gas to be 2600 + the memory expansion cost
// because of the cold account access cost
// because of debugging and tracing, we know that the memory expansion cost is 6
// the computation to be 100+6 for the warm storage read cost + the memory expansion cost
// the state access to be 2500, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallColdNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that does not have any code or value already in the account
// the contract is cold in the access list, we send eth value with the call
// and there is no memory expansion in this case.
//
// we expect the one dimensional gas to be 2600 + the positive value cost (9000)
// less the call gas stipend (-2300) AND the value to empty account cost (25000)
// the computation to be 100 for the warm storage read cost
// the state access to be 2500+9000-2300, the state growth to be 25000,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallColdPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend + params.CallNewAccountGas,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           params.CallNewAccountGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that has some value in the account already, but no code, e.g. it's an EOA
// the contract is cold in the access list, we send eth value with the call
// and there is no memory expansion in this case.
//
// we expect the one dimensional gas to be 2600 + the positive value cost (9000)
// less the call gas stipend (-2300)
// the computation to be 100 for the warm storage read cost
// the state access to be 2500, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallColdPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	// prefund the caller with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that does not have any code or value already in the account
// the contract is cold in the access list, we send eth value with the call
// and we expect memory expansion in this case.
//
// we expect the one dimensional gas to be 2600 + the positive value cost (9000)
// less the call gas stipend (-2300) AND the value to empty account cost (25000)
// and the memory expansion cost (which from debuggging we know is 6)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 2500+9000-2300, the state growth to be 25000,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallColdPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")
	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend + params.CallNewAccountGas + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           params.CallNewAccountGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that has some value in the account already, but no code, e.g. it's an EOA
// the contract is cold in the access list, we send eth value with the call
// and we expect memory expansion in this case.
//
// we expect the one dimensional gas to be 2600 + the positive value cost (9000)
// less the call gas stipend (-2300)
// and the memory expansion cost (which from debuggging we know is 6)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 2500+9000-2300, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallColdPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	// prefund the caller with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that does not have any code or value already in the account
// the contract is warm in the access list, we send no eth value with the call
// and there is no memory expansion in this case.
//
// we expect the one dimensional gas to be the warm storage read cost
// the computation to be 100 for the warm storage read cost
// the state access to be 0, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that does not have any code or value already in the account
// the contract is warm in the access list, we send no eth value with the call
// and we expect memory expansion in this case.
//
// we expect the one dimensional gas to be 100 + the memory expansion cost (6)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 0, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that does not have any code or value already in the account
// the contract is warm in the access list, we send eth value with the call
// and there is no memory expansion in this case.
//
// we expect the one dimensional gas to be 100 + the positive value cost (9000)
// less the call gas stipend (-2300) AND the value to empty account cost (25000)
// the computation to be 100 for the warm storage read cost
// the state access to be 9000-2300, the state growth to be 25000,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend + params.CallNewAccountGas,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           params.CallNewAccountGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that has some value in the account already, but no code, e.g. it's an EOA
// the contract is warm in the access list, we send eth value with the call
// and there is no memory expansion in this case.
//
// we expect the one dimensional gas to be 100 + the positive value cost (9000)
// less the call gas stipend (-2300)
// the computation to be 100 for the warm storage read cost
// the state access to be 9000-2300, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// prefund the callee with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that does not have any code or value already in the account
// the contract is warm in the access list, we send eth value with the call
// and we expect memory expansion in this case.
//
// we expect the one dimensional gas to be 100 + the positive value cost (9000)
// less the call gas stipend (-2300) and the memory expansion cost (6), and
// the value to empty account cost (25000)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 9000-2300, the state growth to be 25000,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")
	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend + params.CallNewAccountGas + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           params.CallNewAccountGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)

}

// this test is the case where we do a CALL to a target contract
// that has some value in the account already, but no code, e.g. it's an EOA
// the contract is warm in the access list, we send eth value with the call
// and there is memory expansion in this case.
//
// we expect the one dimensional gas to be 100 + the positive value cost (9000)
// less the call gas stipend (-2300) and the memory expansion cost
// (which from debuggging we know is 6)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 9000-2300, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// prefund the callee with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")
	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract that does have code
// the contract is cold in the access list, we send no eth value with the call
// and there is no memory expansion in this case.
//
// we expect the one dimensional gas to be 2600 + the child execution gas.
// we happen to know from debugging and tracing that the child execution gas is 22468
// 2600 because of the cold account access cost
// the computation to be 100 for the warm storage read cost
// the state access to be 2500, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallColdNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 22468
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that has code at the target address already. We do not send any
// eth value with the call, and we force memory expansion in this case.
//
// we expect the one dimensional gas to be 2600 + the child execution gas plus
// the memory expansion cost (which from debuggging we know is 6)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 2500, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallColdNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 22468
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that has code at the target address already. We send some eth value
// with the call, and we force memory expansion in this case.
//
// we expect the one dimensional gas to be 2600 + the child execution gas plus
// the memory expansion cost (which from debuggging we know is 6)
// plus the positive value cost (9000) minus the call gas stipend (-2300)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 9000-2300, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallColdPayingContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCallee)
	// fund the caller with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// fund the callee with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + expectedMemExpansionCost + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 22468
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that has code at the target address already. We send some eth value
// with the call, and we force memory expansion in this case.
// the target address is already warm in the access list.
//
// we expect the one dimensional gas to be 100 + the child execution gas plus
// the memory expansion cost (which from debuggging we know is 6)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 0, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 22468
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that has code at the target address already. We do not send any
// eth value with the call, and we force memory expansion in this case.
// the target address is already warm in the access list.
//
// we expect the one dimensional gas to be 100 + the child execution gas plus
// the memory expansion cost (which from debuggging we know is 6)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 0, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCallee)

	// fund the callee address with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 22468
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// this test is the case where we do a CALL to a target contract
// that has code at the target address already. We send some eth value
// with the call, and we do not have memory expansion in this case.
// the target address is already warm in the access list.
//
// we expect the one dimensional gas to be 100 + the child execution gas plus
// the positive value cost (9000) minus the call gas stipend (-2300)
// the computation to be 100 for the warm storage read cost
// the state access to be 9000-2300, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmPayingContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCallee)

	// prefund the caller with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// prefund the callee with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 22468
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)

}

// this test is the case where we do a CALL to a target contract
// that has code at the target address already. We send some eth value
// with the call, and we force memory expansion in this case.
// the target address is already warm in the access list.
//
// we expect the one dimensional gas to be 100 + the child execution gas plus
// the memory expansion cost (which from debuggging we know is 6)
// plus the positive value cost (9000) minus the call gas stipend (-2300)
// the computation to be 100+6 for the warm storage read cost
// the state access to be 9000-2300, the state growth to be 0,
// the history growth to be 0, and the state growth refund to be 0
func TestDimLogCallWarmPayingContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCallee)

	// fund the caller with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// fund the callee address with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)

	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 22468
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// Missing permutation: Cold, NoTransfer, NoCode, Funded, MemUnchanged
func TestDimLogCallColdNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	//prefund callee address
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// Missing permutation: Cold, NoTransfer, NoCode, Funded, MemExpansion
func TestDimLogCallColdNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	//prefund callee address
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")
	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// Missing permutation: Warm, NoTransfer, NoCode, Funded, MemUnchanged
func TestDimLogCallWarmNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	//prefund callee address
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// Missing permutation: Warm, NoTransfer, NoCode, Funded, MemExpansion
func TestDimLogCallWarmNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	//prefund callee address
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")
	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// Missing permutation: Cold, Paying, Contract, Funded, MemUnchanged
func TestDimLogCallColdPayingContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCallee)

	//fund the caller with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// fund the callee address with some funds
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 22468
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, callLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, callLog, expectedChildGasExecutionCost)
}

// #########################################################################################################
// #########################################################################################################
//                                             CALLCODE
// #########################################################################################################
// #########################################################################################################

//// Todo: implement all callcode tests by copying call lol
//func TestDimLogCallCode(t *testing.T) {
//	t.Fail()
//}
