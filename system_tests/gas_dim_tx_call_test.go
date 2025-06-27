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
func TestDimExCallColdNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, virgin, no-code callee, no value transfer, with memory expansion.
func TestDimExCallColdNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, no-code callee, no value transfer, no memory expansion.
func TestDimExCallColdNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, no-code callee, no value transfer, with memory expansion.
func TestDimExCallColdNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, contract callee, no value transfer, no memory expansion.
func TestDimExCallColdNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, contract callee, no value transfer, with memory expansion.
func TestDimExCallColdNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, virgin, no-code callee, with value transfer, no memory expansion.
func TestDimExCallColdPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, virgin, no-code callee, with value transfer, with memory expansion.
func TestDimExCallColdPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, no-code callee, with value transfer, no memory expansion.
func TestDimExCallColdPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeFundedAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, no-code callee, with value transfer, with memory expansion.
func TestDimExCallColdPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeFundedAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, contract callee, with value transfer, no memory expansion.
func TestDimExCallColdPayingContractFundedMemUnchanged(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a cold, funded, contract callee, with value transfer, with memory expansion.
func TestDimExCallColdPayingContractFundedMemExpansion(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, virgin, no-code callee, no value transfer, no memory expansion.
func TestDimExCallWarmNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, virgin, no-code callee, no value transfer, with memory expansion.
func TestDimExCallWarmNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, no-code callee, no value transfer, no memory expansion.
func TestDimExCallWarmNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, no-code callee, no value transfer, with memory expansion.
func TestDimExCallWarmNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, contract callee, no value transfer, no memory expansion.
func TestDimExCallWarmNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, contract callee, no value transfer, with memory expansion.
func TestDimExCallWarmNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallee)

	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, virgin, no-code callee, with value transfer, no memory expansion.
func TestDimExCallWarmPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, virgin, no-code callee, with value transfer, with memory expansion.
func TestDimExCallWarmPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCaller)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, no-code callee, with value transfer, no memory expansion.
func TestDimExCallWarmPayingNoCodeFundedMemUnchanged(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, no-code callee, with value transfer, with memory expansion.
func TestDimExCallWarmPayingNoCodeFundedMemExpansion(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, contract callee, with value transfer, no memory expansion.
func TestDimExCallWarmPayingContractFundedMemUnchanged(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALL to a warm, funded, contract callee, with value transfer, with memory expansion.
func TestDimExCallWarmPayingContractFundedMemExpansion(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
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
func TestDimExCallCodeColdNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, virgin, no-code callee, no value transfer, with memory expansion.
func TestDimExCallCodeColdNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, no-code callee, no value transfer, no memory expansion.
func TestDimExCallCodeColdNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, no-code callee, no value transfer, with memory expansion.
func TestDimExCallCodeColdNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, contract callee, no value transfer, no memory expansion.
func TestDimExCallCodeColdNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemUnchanged, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, contract callee, no value transfer, with memory expansion.
func TestDimExCallCodeColdNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdNoTransferMemExpansion, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, virgin, no-code callee, with value transfer, no memory expansion.
func TestDimExCallCodeColdPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, virgin, no-code callee, with value transfer, with memory expansion.
func TestDimExCallCodeColdPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, no-code callee, with value transfer, no memory expansion.
func TestDimExCallCodeColdPayingNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemUnchanged, noCodeFundedAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, no-code callee, with value transfer, with memory expansion.
func TestDimExCallCodeColdPayingNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	// prefund the caller with some funds
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)
	noCodeFundedAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	builder.L2.TransferBalanceTo(t, "Owner", noCodeFundedAddress, big.NewInt(1e17), builder.L2Info)
	receipt := callOnContractWithOneArg(t, builder, auth, caller.ColdPayableMemExpansion, noCodeFundedAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, contract callee, with value transfer, no memory expansion.
func TestDimExCallCodeColdPayingContractFundedMemUnchanged(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a cold, funded, contract callee, with value transfer, with memory expansion.
func TestDimExCallCodeColdPayingContractFundedMemExpansion(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, virgin, no-code callee, no value transfer, no memory expansion.
func TestDimExCallCodeWarmNoTransferNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, virgin, no-code callee, no value transfer, with memory expansion.
func TestDimExCallCodeWarmNoTransferNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, no-code callee, no value transfer, no memory expansion.
func TestDimExCallCodeWarmNoTransferNoCodeFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, no-code callee, no value transfer, with memory expansion.
func TestDimExCallCodeWarmNoTransferNoCodeFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// prefund callee address
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, contract callee, no value transfer, no memory expansion.
func TestDimExCallCodeWarmNoTransferContractFundedMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemUnchanged, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, contract callee, no value transfer, with memory expansion.
func TestDimExCallCodeWarmNoTransferContractFundedMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCodee)

	// fund the callee address with some funds
	builder.L2.TransferBalanceTo(t, "Owner", calleeAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmNoTransferMemExpansion, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, virgin, no-code callee, with value transfer, no memory expansion.
func TestDimExCallCodeWarmPayingNoCodeVirginMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemUnchanged, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, virgin, no-code callee, with value transfer, with memory expansion.
func TestDimExCallCodeWarmPayingNoCodeVirginMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	callerAddress, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCallCoder)

	noCodeVirginAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	// prefund the caller with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", callerAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.WarmPayableMemExpansion, noCodeVirginAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, no-code callee, with value transfer, no memory expansion.
func TestDimExCallCodeWarmPayingNoCodeFundedMemUnchanged(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, no-code callee, with value transfer, with memory expansion.
func TestDimExCallCodeWarmPayingNoCodeFundedMemExpansion(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, contract callee, with value transfer, no memory expansion.
func TestDimExCallCodeWarmPayingContractFundedMemUnchanged(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CALLCODE to a warm, funded, contract callee, with value transfer, with memory expansion.
func TestDimExCallCodeWarmPayingContractFundedMemExpansion(t *testing.T) {
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

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// This tests a call that does another call,
// specifically a CALL that then does a DELEGATECALL
func TestDimExCallNestedCall(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, caller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployNestedCall)
	calleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployNestedTarget)

	receipt := callOnContractWithOneArg(t, builder, auth, caller.Entrypoint, calleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}
