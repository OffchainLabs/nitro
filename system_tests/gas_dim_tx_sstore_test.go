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
//                                               SSTORE
// #########################################################################################################
// #########################################################################################################
//
// SSTORE has many permutations
// warm or cold
// 0 -> 0
// 0 -> non-zero
// non-zero -> 0
// non-zero -> non-zero (same value)
// non-zero -> non-zero (different value)

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE cold slot from 0 to 0 (no state change).
func TestDimExSstoreColdZeroToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdZeroToZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE cold slot from 0 to non-zero (state growth).
func TestDimExSstoreColdZeroToNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdZeroToNonZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE cold slot from non-zero to 0 (state cleanup with refund).
func TestDimExSstoreColdNonZeroValueToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroValueToZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE cold slot from non-zero to same non-zero value (no state change).
func TestDimExSstoreColdNonZeroToSameNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroToSameNonZeroValue)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE cold slot from non-zero to different non-zero value (state update).
func TestDimExSstoreColdNonZeroToDifferentNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroToDifferentNonZeroValue)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from 0 to 0 (no state change).
func TestDimExSstoreWarmZeroToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmZeroToZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from 0 to non-zero (state growth).
func TestDimExSstoreWarmZeroToNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmZeroToNonZeroValue)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from non-zero to 0 (state cleanup with refund).
func TestDimExSstoreWarmNonZeroValueToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroValueToZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from non-zero to same non-zero value (no state change).
func TestDimExSstoreWarmNonZeroToSameNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroToSameNonZeroValue)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from non-zero to different non-zero value (state update).
func TestDimExSstoreWarmNonZeroToDifferentNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroToDifferentNonZeroValue)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: non-zero -> non-zero -> different non-zero.
func TestDimExSstoreMultipleWarmNonZeroToNonZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToNonZeroToNonZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: non-zero -> non-zero -> same non-zero (with refund).
func TestDimExSstoreMultipleWarmNonZeroToNonZeroToSameNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToNonZeroToSameNonZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: non-zero -> zero -> non-zero (refund adjustment).
func TestDimExSstoreMultipleWarmNonZeroToZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToZeroToNonZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: non-zero -> zero -> same non-zero (refund adjustment).
func TestDimExSstoreMultipleWarmNonZeroToZeroToSameNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToZeroToSameNonZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: zero -> non-zero -> different non-zero.
func TestDimExSstoreMultipleWarmZeroToNonZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmZeroToNonZeroToNonZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: zero -> non-zero -> zero (with refund).
func TestDimExSstoreMultipleWarmZeroToNonZeroBackToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmZeroToNonZeroBackToZero)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}
