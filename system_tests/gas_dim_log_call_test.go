package arbtest

import (
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

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.CallCalleeColdEmptyCodeZeroValue, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
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
	t.Fail()
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

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.CallCalleeColdEmptyCodePositiveValue, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend + params.CallNewAccountGas,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
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
	t.Fail()
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
func TestDimLogCallColdPayingNoCodeVirginMemExpansion(t *testing.T) { t.Fail() }

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
func TestDimLogCallColdPayingNoCodeFundedMemExpansion(t *testing.T) { t.Fail() }

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

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.CallCalleeWarmEmptyCodeZeroValue, emptyAccountAddress)

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

	t.Fail()
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

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.CallCalleeWarmEmptyCodePositiveValue, emptyAccountAddress)

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
func TestDimLogCallWarmPayingNoCodeFundedMemUnchanged(t *testing.T) { t.Fail() }

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
func TestDimLogCallWarmPayingNoCodeVirginMemExpansion(t *testing.T) { t.Fail() }

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
func TestDimLogCallWarmPayingNoCodeFundedMemExpansion(t *testing.T) { t.Fail() }

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

	receipt := callOnContractWithOneArg(t, builder, auth, caller.CallCalleeColdCodeZeroValue, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
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
func TestDimLogCallColdNoTransferContractFundedMemExpansion(t *testing.T) { t.Fail() }

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
func TestDimLogCallColdPayingContractFundedMemExpansion(t *testing.T) { t.Fail() }

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

	receipt := callOnContractWithOneArg(t, builder, auth, caller.CallCalleeWarmCodeZeroValue, calleeAddress)

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
func TestDimLogCallWarmNoTransferContractFundedMemExpansion(t *testing.T) { t.Fail() }

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

	_, caller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.CallCalleeWarmCodePositiveValue, calleeAddress)

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
func TestDimLogCallWarmPayingContractFundedMemExpansion(t *testing.T) { t.Fail() }

// Missing permutation: Cold, NoTransfer, NoCode, Funded, MemUnchanged
func TestDimLogCallColdNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	// This test covers: Cold access, NoTransfer, NoCode (EOA), Funded account, MemUnchanged
	t.Fail()
}

// Missing permutation: Cold, NoTransfer, NoCode, Funded, MemExpansion
func TestDimLogCallColdNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	// This test covers: Cold access, NoTransfer, NoCode (EOA), Funded account, MemExpansion
	t.Fail()
}

// Missing permutation: Warm, NoTransfer, NoCode, Funded, MemUnchanged
func TestDimLogCallWarmNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	// This test covers: Warm access, NoTransfer, NoCode (EOA), Funded account, MemUnchanged
	t.Fail()
}

// Missing permutation: Warm, NoTransfer, NoCode, Funded, MemExpansion
func TestDimLogCallWarmNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	// This test covers: Warm access, NoTransfer, NoCode (EOA), Funded account, MemExpansion
	t.Fail()
}

// Missing permutation: Cold, Paying, Contract, Funded, MemUnchanged
func TestDimLogCallColdPayingContractFundedMemUnchanged(t *testing.T) {
	// This test covers: Cold access, Paying, Contract, Funded account, MemUnchanged
	t.Fail()
}

// #########################################################################################################
// #########################################################################################################
//                                             CALLCODE
// #########################################################################################################
// #########################################################################################################

// Todo: implement all callcode tests by copying call lol
func TestDimLogCallCode(t *testing.T) {
	t.Fail()
}
