// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"testing"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
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

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: BALANCE operation on a cold address (not in access list).
func TestDimExBalanceCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployBalance)
	_, receipt := callOnContract(t, builder, auth, contract.CallBalanceCold)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: BALANCE operation on a warm address (in access list).
func TestDimExBalanceWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployBalance)
	_, receipt := callOnContract(t, builder, auth, contract.CallBalanceWarm)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// ############################################################
//                        EXTCODESIZE
// ############################################################

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: EXTCODESIZE operation on a cold address (not in access list).
func TestDimExExtCodeSizeCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeSize)
	_, receipt := callOnContract(t, builder, auth, contract.GetExtCodeSizeCold)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: EXTCODESIZE operation on a warm address (in access list).
func TestDimExExtCodeSizeWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeSize)
	_, receipt := callOnContract(t, builder, auth, contract.GetExtCodeSizeWarm)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// ############################################################
//                        EXTCODEHASH
// ############################################################

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: EXTCODEHASH operation on a cold address (not in access list).
func TestDimExExtCodeHashCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeHash)
	_, receipt := callOnContract(t, builder, auth, contract.GetExtCodeHashCold)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: EXTCODEHASH operation on a warm address (in access list).
func TestDimExExtCodeHashWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeHash)
	_, receipt := callOnContract(t, builder, auth, contract.GetExtCodeHashWarm)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// ############################################################
//                        SLOAD
// ############################################################
// SLOAD has only one axis of variation:
// Warm vs Cold (in the access list)

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SLOAD operation on a cold storage slot (not previously accessed).
func TestDimExSloadCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySload)
	_, receipt := callOnContract(t, builder, auth, contract.ColdSload)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SLOAD operation on a warm storage slot (previously accessed).
func TestDimExSloadWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySload)
	_, receipt := callOnContract(t, builder, auth, contract.WarmSload)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}
