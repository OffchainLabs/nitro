// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

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

// #########################################################################################################
// #########################################################################################################
//                                             CALL
// #########################################################################################################
// #########################################################################################################

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, virgin, no-code callee, no value transfer, no memory expansion.
func TestDimTxOpCallColdNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, virgin, no-code callee, no value transfer, with memory expansion.
func TestDimTxOpCallColdNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, no-code callee, no value transfer, no memory expansion.
func TestDimTxOpCallColdNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, no-code callee, no value transfer, with memory expansion.
func TestDimTxOpCallColdNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, contract callee, no value transfer, no memory expansion.
func TestDimTxOpCallColdNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, contract callee, no value transfer, with memory expansion.
func TestDimTxOpCallColdNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, virgin, no-code callee, with value transfer, no memory expansion.
func TestDimTxOpCallColdPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, virgin, no-code callee, with value transfer, with memory expansion.
func TestDimTxOpCallColdPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, no-code callee, with value transfer, no memory expansion.
func TestDimTxOpCallColdPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeFundedAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, no-code callee, with value transfer, with memory expansion.
func TestDimTxOpCallColdPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeFundedAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, contract callee, with value transfer, no memory expansion.
func TestDimTxOpCallColdPayingContractFundedMemUnchanged(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, contract callee, with value transfer, with memory expansion.
func TestDimTxOpCallColdPayingContractFundedMemExpansion(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, virgin, no-code callee, no value transfer, no memory expansion.
func TestDimTxOpCallWarmNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, virgin, no-code callee, no value transfer, with memory expansion.
func TestDimTxOpCallWarmNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, no-code callee, no value transfer, no memory expansion.
func TestDimTxOpCallWarmNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, no-code callee, no value transfer, with memory expansion.
func TestDimTxOpCallWarmNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, contract callee, no value transfer, no memory expansion.
func TestDimTxOpCallWarmNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, contract callee, no value transfer, with memory expansion.
func TestDimTxOpCallWarmNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, virgin, no-code callee, with value transfer, no memory expansion.
func TestDimTxOpCallWarmPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, virgin, no-code callee, with value transfer, with memory expansion.
func TestDimTxOpCallWarmPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, no-code callee, with value transfer, no memory expansion.
func TestDimTxOpCallWarmPayingNoCodeFundedMemUnchanged(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, no-code callee, with value transfer, with memory expansion.
func TestDimTxOpCallWarmPayingNoCodeFundedMemExpansion(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, contract callee, with value transfer, no memory expansion.
func TestDimTxOpCallWarmPayingContractFundedMemUnchanged(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, contract callee, with value transfer, with memory expansion.
func TestDimTxOpCallWarmPayingContractFundedMemExpansion(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
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

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, virgin, no-code callee, no value transfer, no memory expansion.
func TestDimTxOpCallCodeColdNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, virgin, no-code callee, no value transfer, with memory expansion.
func TestDimTxOpCallCodeColdNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, no-code callee, no value transfer, no memory expansion.
func TestDimTxOpCallCodeColdNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, no-code callee, no value transfer, with memory expansion.
func TestDimTxOpCallCodeColdNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, contract callee, no value transfer, no memory expansion.
func TestDimTxOpCallCodeColdNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, contract callee, no value transfer, with memory expansion.
func TestDimTxOpCallCodeColdNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, virgin, no-code callee, with value transfer, no memory expansion.
func TestDimTxOpCallCodeColdPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, virgin, no-code callee, with value transfer, with memory expansion.
func TestDimTxOpCallCodeColdPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, no-code callee, with value transfer, no memory expansion.
func TestDimTxOpCallCodeColdPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeFundedAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, no-code callee, with value transfer, with memory expansion.
func TestDimTxOpCallCodeColdPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeFundedAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, contract callee, with value transfer, no memory expansion.
func TestDimTxOpCallCodeColdPayingContractFundedMemUnchanged(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, contract callee, with value transfer, with memory expansion.
func TestDimTxOpCallCodeColdPayingContractFundedMemExpansion(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, virgin, no-code callee, no value transfer, no memory expansion.
func TestDimTxOpCallCodeWarmNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, virgin, no-code callee, no value transfer, with memory expansion.
func TestDimTxOpCallCodeWarmNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, no-code callee, no value transfer, no memory expansion.
func TestDimTxOpCallCodeWarmNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, no-code callee, no value transfer, with memory expansion.
func TestDimTxOpCallCodeWarmNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, contract callee, no value transfer, no memory expansion.
func TestDimTxOpCallCodeWarmNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, contract callee, no value transfer, with memory expansion.
func TestDimTxOpCallCodeWarmNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, virgin, no-code callee, with value transfer, no memory expansion.
func TestDimTxOpCallCodeWarmPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, virgin, no-code callee, with value transfer, with memory expansion.
func TestDimTxOpCallCodeWarmPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeVirginAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, no-code callee, with value transfer, no memory expansion.
func TestDimTxOpCallCodeWarmPayingNoCodeFundedMemUnchanged(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, no-code callee, with value transfer, with memory expansion.
func TestDimTxOpCallCodeWarmPayingNoCodeFundedMemExpansion(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, contract callee, with value transfer, no memory expansion.
func TestDimTxOpCallCodeWarmPayingContractFundedMemUnchanged(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, contract callee, with value transfer, with memory expansion.
func TestDimTxOpCallCodeWarmPayingContractFundedMemExpansion(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// This tests a call that does another call,
// specifically a CALL that then does a DELEGATECALL
func TestDimTxOpCallNestedCall(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployNestedCall)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployNestedTarget)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.Entrypoint, calleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
