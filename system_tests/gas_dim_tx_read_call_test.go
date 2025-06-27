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
func TestDimExDelegateCallColdNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyCold, emptyAccountAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a warm, no-code address, no memory expansion.
func TestDimExDelegateCallWarmNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyWarm, emptyAccountAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a cold contract address, no memory expansion.
func TestDimExDelegateCallColdContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyCold, delegateCalleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a warm contract address, no memory expansion.
func TestDimExDelegateCallWarmContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyWarm, delegateCalleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)

}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a cold, no-code address, with memory expansion.
func TestDimExDelegateCallColdNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyColdMemExpansion, emptyAccountAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a warm, no-code address, with memory expansion.
func TestDimExDelegateCallWarmNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyWarmMemExpansion, emptyAccountAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a cold contract address, with memory expansion.
func TestDimExDelegateCallColdContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyColdMemExpansion, delegateCalleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: DELEGATECALL to a warm contract address, with memory expansion.
func TestDimExDelegateCallWarmContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyWarmMemExpansion, delegateCalleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a cold, no-code address, no memory expansion.
func TestDimExStaticCallColdNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyCold, emptyAccountAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a warm, no-code address, no memory expansion.
func TestDimExStaticCallWarmNoCodeMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyWarm, emptyAccountAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a cold contract address, no memory expansion.
func TestDimExStaticCallColdContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyCold, staticCalleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a warm contract address, no memory expansion.
func TestDimExStaticCallWarmContractMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyWarm, staticCalleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a cold, no-code address, with memory expansion.
func TestDimExStaticCallColdNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyColdMemExpansion, emptyAccountAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a warm, no-code address, with memory expansion.
func TestDimExStaticCallWarmNoCodeMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyWarmMemExpansion, emptyAccountAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a cold contract address, with memory expansion.
func TestDimExStaticCallColdContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyColdMemExpansion, staticCalleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: STATICCALL to a warm contract address, with memory expansion.
func TestDimExStaticCallWarmContractMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyWarmMemExpansion, staticCalleeAddress)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}
