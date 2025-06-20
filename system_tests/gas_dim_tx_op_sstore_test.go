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
func TestDimTxOpSstoreColdZeroToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdZeroToZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE cold slot from 0 to non-zero (state growth).
func TestDimTxOpSstoreColdZeroToNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdZeroToNonZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE cold slot from non-zero to 0 (state cleanup with refund).
func TestDimTxOpSstoreColdNonZeroValueToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroValueToZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE cold slot from non-zero to same non-zero value (no state change).
func TestDimTxOpSstoreColdNonZeroToSameNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroToSameNonZeroValue)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE cold slot from non-zero to different non-zero value (state update).
func TestDimTxOpSstoreColdNonZeroToDifferentNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroToDifferentNonZeroValue)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from 0 to 0 (no state change).
func TestDimTxOpSstoreWarmZeroToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmZeroToZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from 0 to non-zero (state growth).
func TestDimTxOpSstoreWarmZeroToNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmZeroToNonZeroValue)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from non-zero to 0 (state cleanup with refund).
func TestDimTxOpSstoreWarmNonZeroValueToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroValueToZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from non-zero to same non-zero value (no state change).
func TestDimTxOpSstoreWarmNonZeroToSameNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroToSameNonZeroValue)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SSTORE warm slot from non-zero to different non-zero value (state update).
func TestDimTxOpSstoreWarmNonZeroToDifferentNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroToDifferentNonZeroValue)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: non-zero -> non-zero -> different non-zero.
func TestDimTxOpSstoreMultipleWarmNonZeroToNonZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToNonZeroToNonZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: non-zero -> non-zero -> same non-zero (with refund).
func TestDimTxOpSstoreMultipleWarmNonZeroToNonZeroToSameNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToNonZeroToSameNonZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: non-zero -> zero -> non-zero (refund adjustment).
func TestDimTxOpSstoreMultipleWarmNonZeroToZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToZeroToNonZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: non-zero -> zero -> same non-zero (refund adjustment).
func TestDimTxOpSstoreMultipleWarmNonZeroToZeroToSameNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToZeroToSameNonZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: zero -> non-zero -> different non-zero.
func TestDimTxOpSstoreMultipleWarmZeroToNonZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmZeroToNonZeroToNonZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: Multiple SSTORE operations on warm slot: zero -> non-zero -> zero (with refund).
func TestDimTxOpSstoreMultipleWarmZeroToNonZeroBackToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySstore)
	_, receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmZeroToNonZeroBackToZero)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
