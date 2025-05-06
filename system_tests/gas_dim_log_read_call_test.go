package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/gasdimensionsgen"
)

// this file does the tests for the read-only call opcodes, i.e. DELEGATECALL and STATICCALL

// #########################################################################################################
// #########################################################################################################
//	                                    DELEGATECALL & STATICCALL
// #########################################################################################################
// #########################################################################################################
//
// DELEGATECALL and STATICCALL have many permutations
// Warm vs Cold (in the access list)
// Contract vs NoCode  (is there code at the target address that we are calling to? or is it an EOA?)
// MemExpansion or MemUnchanged (memory expansion or not)
//
// static_gas = 0
// dynamic_gas = memory_expansion_cost + code_execution_cost + address_access_cost
// we do not consider the code_execution_cost as part of the cost of the call itself
// since those costs are counted and incurred by the children of the call.

// this test does the case where the target address being delegatecalled to
// is empty - i.e. no address code at that location. The address being called
// to is also cold, therefore incurring the access list cold read cost.
// the solidity compiler forces no memory expansion for us.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// one-dimensional gas cost to be 2600, for the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogDelegateCallColdNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyCold, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the target address being delegatecalled to
// is empty - i.e. no address code at that location.
// the solidity compiler forces no memory expansion for us.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// one-dimensional gas cost to be 100, for the warm access list read,
// computation to be 100 (for the warm access list read),
// state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogDelegateCallWarmNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyWarm, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// in this test, the target address being delegatecalled to
// is non-empty, there is code at that location that will be executed,
// and the address being called is cold.
// the solidity compiler forces no memory expansion for us.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// call to have a one-dimensional gas cost of 2600 + the child execution gas,
// due to the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogDelegateCallColdContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyCold, delegateCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	// this is a magic value from the trace of DelegateCallee contract
	var expectedChildGasExecutionCost uint64 = 22712
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// in this test, the target address being delegatecalled to
// is non-empty, there is code at that location that will be executed,
// and the address being called is warm.
// the solidity compiler forces no memory expansion for us.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// call to have a one-dimensional gas cost of 100 + the child execution gas,
// due to the warm access list read cost,
// computation to be 100 (for the warm access list read),
// state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogDelegateCallWarmContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyWarm, delegateCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	// this is a magic value from the trace of DelegateCallee contract
	var expectedChildGasExecutionCost uint64 = 22712
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the target address being delegatecalled to
// is empty - i.e. no address code at that location. The address being called
// to is also cold, therefore incurring the access list cold read cost.
// in this case we force memory expansion for the call in the solidity
// assembly. By staring at the traces and debugging, we find that the
// memory expansion cost is 6.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// one-dimensional gas cost to be 2600, for the access list cold read cost,
// computation to be 100 + 6 (warm access list read + memory expansion),
// state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogDelegateCallColdNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyColdMemExpansion, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the target address being delegatecalled to
// is empty - i.e. no address code at that location.
// we force memory expansion for the call in the solidity assembly.
// from staring at the traces and debugging,the memory expansion cost is 6.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// one-dimensional gas cost to be 100 + 6, for the warm access list read + memory expansion,
// computation to be 100 + 6 (for the warm access list read + memory expansion),
// state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogDelegateCallWarmNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyWarmMemExpansion, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// in this test, the target address being delegatecalled to
// is non-empty, there is code at that location that will be executed,
// and the address being called is cold.
// we force memory expansion for the call in the solidity assembly.
// from staring at the traces and debugging,the memory expansion cost is 6.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// call to have a one-dimensional gas cost of 2600 + 6 + the child execution gas,
// due to the access list cold read cost and memory expansion cost,
// computation to be 100 + 6 (for the warm access list read + memory expansion),
// state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogDelegateCallColdContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyColdMemExpansion, delegateCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	// this is a magic value from the trace of DelegateCallee contract
	var expectedChildGasExecutionCost uint64 = 22712
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// in this test, the target address being delegatecalled to
// is non-empty, there is code at that location that will be executed,
// and the address being called is warm.
// we force memory expansion for the call in the solidity assembly.
// from staring at the traces and debugging,the memory expansion cost is 6.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// call to have a one-dimensional gas cost of 100 + 6 + the child execution gas,
// due to the warm access list read cost and memory expansion cost,
// computation to be 100 + 6 (for the warm access list read + memory expansion),
// state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogDelegateCallWarmContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyWarmMemExpansion, delegateCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	// this is a magic value from the trace of DelegateCallee contract
	var expectedChildGasExecutionCost uint64 = 22712
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the a contract calls another contract via staticcall
// the target address is empty (does not actually have code at that address),
// and the address being called is cold.
// there is no memory expansion in this case.
//
// we expect the one-dimensional gas cost to be 2500, for the cold access list read cost,
// computation to be 100 for the warm access list read cost, state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogStaticCallColdNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyCold, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the a contract calls another contract via staticcall
// the target address is empty (does not actually have code at that address),
// and the address being called is warm.
// there is no memory expansion in this case.
//
// we expect the one-dimensional gas cost to be 100, for the warm access list read cost,
// computation to be 100 for the warm access list read cost, state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogStaticCallWarmNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyWarm, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the a contract calls another contract via staticcall
// the target address is non-empty, so there is code at that location that will be executed,
// and the address being called is cold.
// there is no memory expansion in this case.
//
// we expect the one-dimensional gas cost to be 2500, for the cold access list read cost,
// computation to be 100 for the warm access list read cost, state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogStaticCallColdContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyCold, staticCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 2409
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the a contract calls another contract via staticcall
// the target address is non-empty, so there is code at that location that will be executed,
// and the address being called is warm.
// there is no memory expansion in this case.
//
// we expect the one-dimensional gas cost to be 100, for the warm access list read cost,
// computation to be 100 for the warm access list read cost, state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogStaticCallWarmContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyWarm, staticCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 2409
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where a contract calls another contract via staticcall
// the target address is empty, so there is no code at that location that will be executed,
// and the address being called is cold.
// memory expansion occurs in this case.
//
// we expect the one-dimensional gas cost to be 2600, for the cold access list read cost,
// plus the memory expansion cost, which via debugging and tracing, we know is 6 gas.
// computation to be 100+6 for the warm access list read cost, state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogStaticCallColdNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyColdMemExpansion, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where a contract calls another contract via staticcall
// the target address is empty, so there is no code at that location that will be executed,
// and the address being called is warm.
// memory expansion occurs in this case.
//
// we expect the one-dimensional gas cost to be 100, for the warm access list read cost,
// plus the memory expansion cost, which via debugging and tracing, we know is 6 gas.
// computation to be 100+6 for the warm access list read cost, state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogStaticCallWarmNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyWarmMemExpansion, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where a contract calls another contract via staticcall
// the target address is non-empty, so there is code at that location that will be executed,
// and the address being called is cold.
// memory expansion occurs in this case.
//
// we expect the one-dimensional gas cost to be 2600, for the cold access list read cost,
// plus the memory expansion cost, which via debugging and tracing, we know is 6 gas.
// we also know the child execution gas is 2409 from debugging and tracing.
// computation to be 100+6 for the warm access list read cost, state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogStaticCallColdContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyColdMemExpansion, staticCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	var memoryExpansionCost uint64 = 6
	var expectedChildGasExecutionCost uint64 = 2409

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where a contract calls another contract via staticcall
// the target address is non-empty, so there is code at that location that will be executed,
// and the address being called is warm.
// memory expansion occurs in this case.
//
// we expect the one-dimensional gas cost to be 100, for the warm access list read cost,
// plus the memory expansion cost, which via debugging and tracing, we know is 6 gas.
// we also know the child execution gas is 2409 from debugging and tracing.
// computation to be 100+6 for the warm access list read cost, state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogStaticCallWarmContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyWarmMemExpansion, staticCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	var memoryExpansionCost uint64 = 6
	var expectedChildGasExecutionCost uint64 = 2409

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}
