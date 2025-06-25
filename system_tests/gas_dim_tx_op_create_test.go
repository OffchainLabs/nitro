// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"math/big"
	"testing"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

// #########################################################################################################
// #########################################################################################################
//	                                             CREATE & CREATE2
// #########################################################################################################
// #########################################################################################################
// CREATE and CREATE2  have permutations:
// Paying vs NoTransfer (is ether value being sent with this call?)
// MemExpansion vs MemUnchanged (does the creation write to new additional memory?)

// #########################################################################################################
//                                              CREATE
// #########################################################################################################

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CREATE operation with no value transfer and no memory expansion.
func TestDimTxOpCreateNoTransferMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreator)

	_, receipt := callOnContract(t, builder, auth, creator.CreateNoTransferMemUnchanged)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CREATE operation with no value transfer and memory expansion.
func TestDimTxOpCreateNoTransferMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreator)

	_, receipt := callOnContract(t, builder, auth, creator.CreateNoTransferMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CREATE operation with value transfer and no memory expansion.
func TestDimTxOpCreatePayingMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	creatorAddress, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreator)
	// transfer some eth to the creator contract
	builder.L2.TransferBalanceTo(t, "Owner", creatorAddress, big.NewInt(1e17), builder.L2Info)

	_, receipt := callOnContract(t, builder, auth, creator.CreatePayableMemUnchanged)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CREATE operation with value transfer and memory expansion.
func TestDimTxOpCreatePayingMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	creatorAddress, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreator)
	// transfer some eth to the creator contract
	builder.L2.TransferBalanceTo(t, "Owner", creatorAddress, big.NewInt(1e17), builder.L2Info)

	_, receipt := callOnContract(t, builder, auth, creator.CreatePayableMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// #########################################################################################################
//                                              CREATE2
// #########################################################################################################

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CREATE2 operation with no value transfer and no memory expansion.
func TestDimTxOpCreate2NoTransferMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreatorTwo)

	receipt := callOnContractWithOneArg(t, builder, auth, creator.CreateTwoNoTransferMemUnchanged, [32]byte{0x13, 0x37})

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CREATE2 operation with no value transfer and memory expansion.
func TestDimTxOpCreate2NoTransferMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreatorTwo)

	receipt := callOnContractWithOneArg(t, builder, auth, creator.CreateTwoNoTransferMemExpansion, [32]byte{0x13, 0x37})

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CREATE2 operation with value transfer and no memory expansion.
func TestDimTxOpCreate2PayingMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreatorTwo)

	receipt := callOnContractWithOneArg(t, builder, auth, creator.CreateTwoNoTransferMemUnchanged, [32]byte{0x13, 0x37})

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: CREATE2 operation with value transfer and memory expansion.
func TestDimTxOpCreate2PayingMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	creatorAddress, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreatorTwo)
	// transfer some eth to the creator contract
	builder.L2.TransferBalanceTo(t, "Owner", creatorAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, creator.CreateTwoPayableMemExpansion, [32]byte{0x13, 0x37})

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
