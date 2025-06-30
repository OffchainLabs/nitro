// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
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

const (
	callChildExecutionCost     uint64 = 22468
	callCodeChildExecutionCost uint64 = 490
)

// #########################################################################################################
// #########################################################################################################
//                                             CALL
// #########################################################################################################
// #########################################################################################################

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallColdNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

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
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallColdNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallColdNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

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
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallColdNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallColdNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

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
		ChildExecutionCost:    callChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallColdNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallColdPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
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
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallColdPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend + params.CallNewAccountGas + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           params.CallNewAccountGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallColdPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
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
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallColdPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallColdPayingContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	// fund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

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
		ChildExecutionCost:    callChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallColdPayingContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	// fund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memExpansionMemoryExpansionCost + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallWarmNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

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
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallWarmNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallWarmNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

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
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallWarmNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallWarmNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

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
		ChildExecutionCost:    callChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallWarmNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallWarmPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

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
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallWarmPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend + params.CallNewAccountGas + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           params.CallNewAccountGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// ether value is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallWarmPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// prefund the callee with some funds
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)

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
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// ether value is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallWarmPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// prefund the callee with some funds
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// ether value is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallWarmPayingContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// prefund the callee with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

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
		ChildExecutionCost:    callChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALL from caller to callee where:
// callee is warm in the access list at the time of the call
// ether value is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallWarmPayingContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	// fund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// #########################################################################################################
// #########################################################################################################
//                                             CALLCODE
// #########################################################################################################
// #########################################################################################################
// CALLCODE is deprecated and also weird.
// It operates identically to DELEGATECALL but allows you to send funds with the call
// if you send those funds, it will send them to the library address rather than yourself.
// So it has a positive_value_cost just like CALL,
// but it does NOT have the positive_value_to_new_account cost (params.CallNewAccountGas)
// so all of the test cases for CALLCODE are identical to CALL, EXCEPT FOR
// the NoCodeVirgin test cases, where there is no cost to positive_value_to_new_account
// (which tbh sounds like an EVM mispricing bug)

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeColdNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeColdNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeColdNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeColdNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")
	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeColdNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callCodeChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// no ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeColdNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callCodeChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeColdPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeColdPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")
	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeColdPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeColdPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeColdPayingContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	// fund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callCodeChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is cold in the access list at the time of the call
// ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeColdPayingContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	// fund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memExpansionMemoryExpansionCost + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callCodeChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeWarmNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeWarmNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeWarmNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeWarmNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")
	var expectedMemExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + expectedMemExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeWarmNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callCodeChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// no ether is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeWarmNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callCodeChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeWarmPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// ether is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee is virgin (has never been seen before on the network)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeWarmPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeVirginAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// ether value is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeWarmPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// prefund the callee with some funds
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// ether value is being sent with the call
// the callee has no code (either it is an EOA, or has never been seen on the network)
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeWarmPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// prefund the callee with some funds
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeFundedAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend + memExpansionMemoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// ether value is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is no memory expansion as part of the CALL opcode itself (it writes to memory but stays inside the bounds of the original memory)
func TestDimLogCallCodeWarmPayingContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// prefund the callee with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callCodeChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// Perform a CALLCODE from caller to callee where:
// callee is warm in the access list at the time of the call
// ether value is being sent with the call
// the callee has contract code
// the callee has been funded before (someone sent it money in the past)
// there is memory expansion, the CALL writes to memory outside the bounds of the original memory
func TestDimLogCallCodeWarmPayingContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	// fund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALLCODE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost + params.CallValueTransferGas - params.CallStipend,
		Computation:           params.WarmStorageReadCostEIP2929 + memExpansionMemoryExpansionCost,
		StateAccess:           params.CallValueTransferGas - params.CallStipend,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callCodeChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}

// This tests a call that does another call,
// specifically a CALL that then does a DELEGATECALL
// and we check that the gas dimensions are correct
// for the parent.
func TestDimLogNestedCall(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployNestedCall)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployNestedTarget)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.Entrypoint, calleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	callLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CALL")

	var callChildExecutionCost uint64 = 25921 // from the struct logger output

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    callChildExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}
