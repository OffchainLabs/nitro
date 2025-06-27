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
func TestDimExLog0TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicEmptyData)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG0 with no topics, 7 bytes of data, no memory expansion.
func TestDimExLog0ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicNonEmptyData)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG1 with one topic, no data, no memory expansion.
func TestDimExLog1TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitOneTopicEmptyData)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG1 with one topic, 9 bytes of data, no memory expansion.
func TestDimExLog1ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitOneTopicNonEmptyData)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG2 with two topics, no data, no memory expansion.
func TestDimExLog2TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitTwoTopics)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG2 with two topics, 32 bytes of data (address), no memory expansion.
func TestDimExLog2ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitTwoTopicsExtraData)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG3 with three topics, no data, no memory expansion.
func TestDimExLog3TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitThreeTopics)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG3 with three topics, 32 bytes of data (bytes32), no memory expansion.
func TestDimExLog3ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitThreeTopicsExtraData)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG4 with four topics, no data, no memory expansion.
func TestDimExLog4TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitFourTopics)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG4 with four topics, 32 bytes of data (bytes32), no memory expansion.
func TestDimExLog4ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitFourTopicsExtraData)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG0 with no topics, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimExLog0ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicNonEmptyDataAndMemExpansion)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG1 with one topic, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimExLog1ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitOneTopicNonEmptyDataAndMemExpansion)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG2 with two topics, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimExLog2ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitTwoTopicsExtraDataAndMemExpansion)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG3 with three topics, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimExLog3ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitThreeTopicsExtraDataAndMemExpansion)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: LOG4 with four topics, 64 bytes of data, memory expansion from 96 to 160 bytes.
func TestDimExLog4ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitFourTopicsExtraDataAndMemExpansion)

	TxExTraceAndCheck(t, ctx, builder, receipt)
}
