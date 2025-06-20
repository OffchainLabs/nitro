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
//	              					LOG0, LOG1, LOG2, LOG3, LOG4
// #########################################################################################################
// #########################################################################################################
//
// Logs are pretty straightforward, they have a static cost
// we assign to computation, a linear size cost we assign directly
// to history growth, and a topic data cost. The topic data cost
// we subdivide, because some of the topic cost needs to pay for the
// cryptography we do for the bloom filter.
// 32 bytes per topic are stored in the history for the topic count.
// since the other data is charged 8 gas per byte, then each of the 375
// should in theory be charged 256 gas for the 32 bytes of topic data and 119 gas for computation
// so we subdivide the 375 gas per topic count into 256 for the topic data and 119 for the computation.
//
// LOG has the following axes of variation:
// Number of topics (0-4)
// TopicsOnly vs ExtraData Do we have any un-indexed extra data (data that is not part of the topics)?
// MemExpansion or MemUnchanged (memory expansion or not)
// The memory expansion depends on doing a read of data from memory,
// so TopicsOnly cases cannot cause memory expansion.

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG0 with no topics, no data, no memory expansion.
func TestDimTxOpLog0TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicEmptyData)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG0 with no topics, 7 bytes of data, no memory expansion.
func TestDimTxOpLog0ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicNonEmptyData)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG1 with one topic, no data, no memory expansion.
func TestDimTxOpLog1TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitOneTopicEmptyData)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG1 with one topic, 9 bytes of data, no memory expansion.
func TestDimTxOpLog1ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitOneTopicNonEmptyData)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG2 with two topics, no data, no memory expansion.
func TestDimTxOpLog2TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitTwoTopics)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG2 with two topics, 32 bytes of data (address), no memory expansion.
func TestDimTxOpLog2ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitTwoTopicsExtraData)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG3 with three topics, no data, no memory expansion.
func TestDimTxOpLog3TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitThreeTopics)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG3 with three topics, 32 bytes of data (bytes32), no memory expansion.
func TestDimTxOpLog3ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitThreeTopicsExtraData)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG4 with four topics, no data, no memory expansion.
func TestDimTxOpLog4TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitFourTopics)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG4 with four topics, 32 bytes of data (bytes32), no memory expansion.
func TestDimTxOpLog4ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitFourTopicsExtraData)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG0 with no topics, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimTxOpLog0ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicNonEmptyDataAndMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG1 with one topic, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimTxOpLog1ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitOneTopicNonEmptyDataAndMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG2 with two topics, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimTxOpLog2ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitTwoTopicsExtraDataAndMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG3 with three topics, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimTxOpLog3ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitThreeTopicsExtraDataAndMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG4 with four topics, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimTxOpLog4ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitFourTopicsExtraDataAndMemExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
