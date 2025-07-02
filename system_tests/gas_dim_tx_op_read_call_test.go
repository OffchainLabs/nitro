// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
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

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a cold, no-code address, no memory expansion.
func TestDimTxOpDelegateCallColdNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyCold, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a warm, no-code address, no memory expansion.
func TestDimTxOpDelegateCallWarmNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyWarm, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a cold contract address, no memory expansion.
func TestDimTxOpDelegateCallColdContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyCold, delegateCalleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a warm contract address, no memory expansion.
func TestDimTxOpDelegateCallWarmContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyWarm, delegateCalleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)

}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a cold, no-code address, with memory expansion.
func TestDimTxOpDelegateCallColdNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyColdMemExpansion, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a warm, no-code address, with memory expansion.
func TestDimTxOpDelegateCallWarmNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyWarmMemExpansion, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a cold contract address, with memory expansion.
func TestDimTxOpDelegateCallColdContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyColdMemExpansion, delegateCalleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a warm contract address, with memory expansion.
func TestDimTxOpDelegateCallWarmContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyWarmMemExpansion, delegateCalleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a cold, no-code address, no memory expansion.
func TestDimTxOpStaticCallColdNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyCold, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a warm, no-code address, no memory expansion.
func TestDimTxOpStaticCallWarmNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyWarm, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a cold contract address, no memory expansion.
func TestDimTxOpStaticCallColdContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyCold, staticCalleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a warm contract address, no memory expansion.
func TestDimTxOpStaticCallWarmContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyWarm, staticCalleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a cold, no-code address, with memory expansion.
func TestDimTxOpStaticCallColdNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyColdMemExpansion, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a warm, no-code address, with memory expansion.
func TestDimTxOpStaticCallWarmNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyWarmMemExpansion, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a cold contract address, with memory expansion.
func TestDimTxOpStaticCallColdContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyColdMemExpansion, staticCalleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a warm contract address, with memory expansion.
func TestDimTxOpStaticCallWarmContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyWarmMemExpansion, staticCalleeAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
