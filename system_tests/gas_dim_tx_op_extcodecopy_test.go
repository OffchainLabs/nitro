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
//                                               EXTCODECOPY
// #########################################################################################################
// #########################################################################################################

// EXTCODECOPY reads from state and copies code to memory
// for gas dimensions, we don't care about expanding memory, but
// we do care about the cost being correct
// EXTCODECOPY has two axes of variation:
// Warm vs Cold (in the access list)
// MemExpansion or MemUnchanged (memory expansion or not)

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: EXTCODECOPY with cold code access, no memory expansion.
// Expected: 2600 gas total (100 computation + 2500 state access + minimum word cost).
func TestDimTxOpExtCodeCopyColdMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeCopy)
	_, receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyColdNoMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: EXTCODECOPY with cold code access and memory expansion.
// Expected: 2600 gas + memory expansion cost (100 computation + 2500 state access + minimum word cost + memory expansion).
func TestDimTxOpExtCodeCopyColdMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeCopy)
	_, receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyColdMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: EXTCODECOPY with warm code access, no memory expansion.
// Expected: 100 gas total (100 computation + minimum word cost).
func TestDimTxOpExtCodeCopyWarmMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeCopy)
	_, receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyWarmNoMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: EXTCODECOPY with warm code access and memory expansion.
// Expected: 100 gas + memory expansion cost (100 computation + minimum word cost + memory expansion).
func TestDimTxOpExtCodeCopyWarmMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeCopy)
	_, receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyWarmMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
