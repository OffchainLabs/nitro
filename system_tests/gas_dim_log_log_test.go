// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

const (
	LogStaticCost            = params.LogGas
	LogDataGas               = params.LogDataGas
	LogTopicGasHistoryGrowth = 256
	LogTopicGasComputation   = params.LogTopicGas - LogTopicGasHistoryGrowth
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

// This test deploys a contract that emits an empty LOG0
//
// since it has no data, we expect the gas cost to just be
// the static cost of the LOG0 opcode, and we assign that to
// computation.
// Therefore we expect the one dimensional gas cost to be
// 375, computation to be 375, and all other gas dimensions to be 0
func TestDimLogLog0TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicEmptyData)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log0Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG0")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost,
		Computation:           LogStaticCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log0Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log0Log)
}

// This test deploys a contract that emits a LOG0 with 7 bytes ("abcdefg") of data.
//
// since it has data, we expect the gas cost to be the static cost of the LOG0 opcode
// plus the 8 * 7 bytes of data cost.
// Therefore we expect the one dimensional gas cost to be 375 + the data gas cost,
// computation to be 375, the history growth to be 8 * 7, and all other gas dimensions to be 0
func TestDimLogLog0ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicNonEmptyData)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log0Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG0")

	var numBytesWritten uint64 = 7

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + LogDataGas*numBytesWritten,
		Computation:           LogStaticCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         LogDataGas * numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log0Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log0Log)
}

// This test deploys a contract that a LOG1 with no data
//
// since it has no data, we expect the gas cost to be
// the static cost of the LOG1 opcode + the topic gas cost.
// Therefore we expect the one dimensional gas cost to be
// 375 + 375, the computation to be 375 + 119,
// the state access to be 0, the state growth to be 0,
// the history growth to be 256, and the state growth refund to be 0
func TestDimLogLog1TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitOneTopicEmptyData)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log1Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG1")
	var numTopics uint64 = 1

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation),
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics * LogTopicGasHistoryGrowth,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log1Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log1Log)
}

// This test deploys a contract that emits a LOG1 with 9 bytes ("hijklmnop") of data.
//
// since it has data, we expect the gas cost to be the static cost of the LOG1 opcode
// plus the 8 * 9 bytes of data cost + the topic gas cost
// split between history growth and computation for 1 topic
// Therefore we expect the one dimensional gas cost to be 375 + 8 * 9 + 256 + 119,
// computation to be 375 + 119, the state access to be 0, the state growth to be 0,
// the history growth to be 256 + 8 * 9, and the state growth refund to be 0
func TestDimLogLog1ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitOneTopicNonEmptyData)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log1Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG1")

	var numBytesWritten uint64 = 9
	var numTopics uint64 = 1

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation) + LogDataGas*numBytesWritten,
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics*LogTopicGasHistoryGrowth + LogDataGas*numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log1Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log1Log)
}

// This test checks the gas cost of a LOG2 with two topics, but no data
//
// since it has no data, we expect the one-dimensional gas cost to be the
// static cost of the LOG2 opcode plus the 2 * 375 for the two topics
// computation to be 375 + 2 * 119, the state access to be 0, the state growth to be 0,
// the history growth to be 2 * 256, and the state growth refund to be 0
func TestDimLogLog2TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitTwoTopics)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log2Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG2")

	var numTopics uint64 = 2

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation),
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics * LogTopicGasHistoryGrowth,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log2Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log2Log)
}

// This test checks the gas cost of a LOG2 with two topics, and emitting an address as extra data
//
// since it has 32 bytes (an address encoded as a uint256) of data,
// we expect the one-dimensional gas cost to be the static cost of the
// LOG2 opcode plus the 2 * 375 for the two topics
// plus the 32 bytes of data cost
// therefore we expect computation to be 375 + 2 * 119,
// the state access to be 0, the state growth to be 0,
// the history growth to be 2 * 256 + 32*8, and the state growth refund to be 0
func TestDimLogLog2ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitTwoTopicsExtraData)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log2Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG2")

	var numBytesWritten uint64 = 32 // address is 20 bytes, but encoded as a uint256 so 32 bytes
	var numTopics uint64 = 2

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation) + LogDataGas*numBytesWritten,
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics*LogTopicGasHistoryGrowth + LogDataGas*numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log2Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log2Log)
}

// This test checks the gas cost of a LOG3 with three topics, but no data
//
// since it has no data, we expect the one-dimensional gas cost to be the
// static cost of the LOG3 opcode plus the 3 * 375 for the three topics
// computation to be 375 + 3 * 119, the state access to be 0, the state growth to be 0,
// the history growth to be 3 * 256, and the state growth refund to be 0
func TestDimLogLog3TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitThreeTopics)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log3Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG3")

	var numTopics uint64 = 3

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation),
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics * LogTopicGasHistoryGrowth,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log3Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log3Log)
}

// This test checks the gas cost of a LOG3 with three topics, and emitting bytes32 as extra data
//
// since it has 32 bytes (a bytes32) of data,
// we expect the one-dimensional gas cost to be the static cost of the
// LOG3 opcode plus the 3 * 375 for the three topics
// plus the 32 bytes of data cost
// therefore we expect computation to be 375 + 3 * 119,
// the state access to be 0, the state growth to be 0,
// the history growth to be 3 * 256 + 32*8, and the state growth refund to be 0
func TestDimLogLog3ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitThreeTopicsExtraData)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log3Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG3")

	var numBytesWritten uint64 = 32
	var numTopics uint64 = 3

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation) + LogDataGas*numBytesWritten,
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics*LogTopicGasHistoryGrowth + LogDataGas*numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log3Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log3Log)
}

// This test checks the gas cost of a LOG4 with four topics, but no data
//
// since it has no data, we expect the one-dimensional gas cost to be the
// static cost of the LOG4 opcode plus the 4 * 375 for the four topics
// computation to be 375 + 4 * 119, the state access to be 0, the state growth to be 0,
// the history growth to be 4 * 256, and the state growth refund to be 0
func TestDimLogLog4TopicsOnlyMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitFourTopics)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log4Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG4")

	var numTopics uint64 = 4

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation),
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics * LogTopicGasHistoryGrowth,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log4Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log4Log)
}

// This test checks the gas cost of a LOG4 with four topics, and emitting bytes32 as extra data
//
// since it has 32 bytes (a bytes32) of data,
// we expect the one-dimensional gas cost to be the static cost of the
// LOG4 opcode plus the 4 * 375 for the four topics
// plus the 32 bytes of data cost
// therefore we expect computation to be 375 + 4 * 119,
// the state access to be 0, the state growth to be 0,
// the history growth to be 4 * 256 + 32*8, and the state growth refund to be 0
func TestDimLogLog4ExtraDataMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitFourTopicsExtraData)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log4Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG4")

	var numBytesWritten uint64 = 32
	var numTopics uint64 = 4

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation) + LogDataGas*numBytesWritten,
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics*LogTopicGasHistoryGrowth + LogDataGas*numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log4Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log4Log)
}

// Comments about the memory expanstion test cases:
// For all of the memory expansion tests, we set the
// offset to the end of the memory and the length to 64.
// The memory size at the time of the LOGN opcode is 96.
// therefore we expect the memory expansion cost to be:
//
// memory_size_word = (memory_byte_size + 31) / 32
// memory_cost = (memory_size_word ** 2) / 512 + (3 * memory_size_word)
// memory_expansion_cost = new_memory_cost - last_memory_cost
//
// so we have 5**2 / 512 + (3 * 5) - (3**2)/512 - (3*3)
// 15 - 9 = 6
var logNMemoryExpansionCost uint64 = 6

// This test checks the gas cost of a LOG0 with no topics, and with
// data that is garbage, since it's doing memory expansion, and reading
// past the end of the memory. In this test, we set up the memory of size
// 96 and tell the LOG0 to read starting at position 96 and read 64 bytes.
//
// We expect the one-dimensional gas cost to be the static cost of the
// LOG0 opcode plus the 64 bytes memory expansion cost, which at memory
// length 96, is 6.
//
// we expect the computation gas cost to be the static cost of the LOG0 opcode
// + the memory expansion cost
// we expect the state access to be 0, the state growth to be 0,
// the history growth to be 64*8, and the state growth refund to be 0
func TestDimLogLog0ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicNonEmptyDataAndMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log0Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG0")

	var numBytesWritten uint64 = 64

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + LogDataGas*numBytesWritten + logNMemoryExpansionCost,
		Computation:           LogStaticCost + logNMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         LogDataGas * numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log0Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log0Log)
}

// This test checks the gas cost of a LOG1 with one topic, and with
// data that is garbage, since it's doing memory expansion, and reading
// past the end of the memory. In this test, we set up the memory of size
// 96 and tell the LOG1 to read starting at position 96 and read 64 bytes.
//
// since it has data, we expect the gas cost to be the static cost of the LOG1 opcode
// plus the 8 * 64 bytes of data cost + the topic gas cost
// split between history growth and computation for 1 topic
// Therefore we expect the one dimensional gas cost to be 375 + 8 * 64 + 256 + 119 + 6,
// computation to be 375 + 119 + 6, the state access to be 0, the state growth to be 0,
// the history growth to be 256 + 8 * 64, and the state growth refund to be 0
func TestDimLogLog1ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitOneTopicNonEmptyDataAndMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log1Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG1")

	var numBytesWritten uint64 = 64
	var numTopics uint64 = 1

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation) + LogDataGas*numBytesWritten + logNMemoryExpansionCost,
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation + logNMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics*LogTopicGasHistoryGrowth + LogDataGas*numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log1Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log1Log)
}

// This test checks the gas cost of a LOG2 with two topics, and with
// data that is garbage, since it's doing memory expansion, and reading
// past the end of the memory. In this test, we set up the memory of size
// 96 and tell the LOG2 to read starting at position 96 and read 64 bytes.
//
// since it has data, we expect the gas cost to be the static cost of the LOG2 opcode
// plus the 8 * 64 bytes of data cost + the topic gas cost
// split between history growth and computation for 2 topics
// Therefore we expect the one dimensional gas cost to be 375 + 8 * 64 + 2*(256 + 119) + 6,
// computation to be 375 + 2*119 + 6, the state access to be 0, the state growth to be 0,
// the history growth to be 2*256 + 8 * 64, and the state growth refund to be 0
func TestDimLogLog2ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitTwoTopicsExtraDataAndMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log2Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG2")

	var numBytesWritten uint64 = 64
	var numTopics uint64 = 2

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation) + LogDataGas*numBytesWritten + logNMemoryExpansionCost,
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation + logNMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics*LogTopicGasHistoryGrowth + LogDataGas*numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log2Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log2Log)
}

// This test checks the gas cost of a LOG3 with three topics, and with
// data that is garbage, since it's doing memory expansion, and reading
// past the end of the memory. In this test, we set up the memory of size
// 96 and tell the LOG3 to read starting at position 96 and read 64 bytes.
//
// since it has data, we expect the gas cost to be the static cost of the LOG3 opcode
// plus the 8 * 64 bytes of data cost + the topic gas cost
// split between history growth and computation for 3 topics
// Therefore we expect the one dimensional gas cost to be 375 + 8 * 64 + 3*(256 + 119) + 6,
// computation to be 375 + 3*119 + 6, the state access to be 0, the state growth to be 0,
// the history growth to be 3*256 + 8 * 64, and the state growth refund to be 0
func TestDimLogLog3ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitThreeTopicsExtraDataAndMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log3Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG3")

	var numBytesWritten uint64 = 64
	var numTopics uint64 = 3

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation) + LogDataGas*numBytesWritten + logNMemoryExpansionCost,
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation + logNMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics*LogTopicGasHistoryGrowth + LogDataGas*numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log3Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log3Log)
}

// This test checks the gas cost of a LOG4 with four topics, and with
// data that is garbage, since it's doing memory expansion, and reading
// past the end of the memory. In this test, we set up the memory of size
// 96 and tell the LOG4 to read starting at position 96 and read 64 bytes.
//
// since it has data, we expect the gas cost to be the static cost of the LOG4 opcode
// plus the 8 * 64 bytes of data cost + the topic gas cost
// split between history growth and computation for 4 topics
// Therefore we expect the one dimensional gas cost to be 375 + 8 * 64 + 4*(256 + 119) + 6,
// computation to be 375 + 4*119 + 6, the state access to be 0, the state growth to be 0,
// the history growth to be 4*256 + 8 * 64, and the state growth refund to be 0
func TestDimLogLog4ExtraDataMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployLogEmitter)
	_, receipt := callOnContract(t, builder, auth, contract.EmitFourTopicsExtraDataAndMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	log4Log := getSpecificDimensionLog(t, traceResult.DimensionLogs, "LOG4")

	var numBytesWritten uint64 = 64
	var numTopics uint64 = 4

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: LogStaticCost + numTopics*(LogTopicGasHistoryGrowth+LogTopicGasComputation) + LogDataGas*numBytesWritten + logNMemoryExpansionCost,
		Computation:           LogStaticCost + numTopics*LogTopicGasComputation + logNMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         numTopics*LogTopicGasHistoryGrowth + LogDataGas*numBytesWritten,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		log4Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log4Log)
}
