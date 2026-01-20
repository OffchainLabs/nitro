// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers/native"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

type OpcodeSumTraceResult = native.TxGasDimensionByOpcodeExecutionResult

// #########################################################################################################
// #########################################################################################################
//                          REGULAR COMPUTATION OPCODES (ADD, SWAP, ETC)
// #########################################################################################################
// #########################################################################################################

// Core test function that tests if the tracer calculates the same gas
// used for a transaction as the TX receipt, for computation-only opcodes.
// Runs on a replica node with the specified execution client mode.
func testDimTxOpComputationOnlyOpcodes(t *testing.T, executionClientMode ExecutionClientMode) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	// Build replica node with specified execution client mode
	replicaConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	replicaParams := &SecondNodeParams{
		nodeConfig:             replicaConfig,
		useExecutionClientOnly: true,
		executionClientMode:    executionClientMode,
	}
	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, replicaParams)
	defer replicaCleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCounter)
	_, receipt := callOnContract(t, builder, auth, contract.NoSpecials)

	// Wait for replica to sync the transaction
	_, err := WaitForTx(ctx, replicaTestClient.Client, receipt.TxHash, time.Second*15)
	Require(t, err)

	// Call trace and check on the replica
	TxOpTraceAndCheck(t, ctx, replicaTestClient, receipt)
}

// TestDimTxOpComputationOnlyOpcodesInternal tests with internal geth execution client on a replica node
func TestDimTxOpComputationOnlyOpcodesInternal(t *testing.T) {
	testDimTxOpComputationOnlyOpcodes(t, ExecutionClientModeInternal)
}

// TestDimTxOpComputationOnlyOpcodesExternal tests with external Nethermind execution client on a replica node
// Requires Nethermind to be running with environment variables:
// - PR_NETH_RPC_CLIENT_URL (default: http://localhost:20545)
// - PR_NETH_WS_CLIENT_URL (default: ws://localhost:28551)
func TestDimTxOpComputationOnlyOpcodesExternal(t *testing.T) {
	testDimTxOpComputationOnlyOpcodes(t, ExecutionClientModeExternal)
}

// TestDimTxOpComputationOnlyOpcodesComparison tests comparing internal geth and external Nethermind on a replica node
// Runs both execution clients in parallel and compares all results for correctness
// Requires Nethermind to be running (same as TestDimTxOpComputationOnlyOpcodesExternal)
func TestDimTxOpComputationOnlyOpcodesComparison(t *testing.T) {
	testDimTxOpComputationOnlyOpcodes(t, ExecutionClientModeComparison)
}

// #########################################################################################################
// #########################################################################################################
//                                            HELPER FUNCTIONS
// #########################################################################################################
// #########################################################################################################

// helper function that automates calling debug_traceTransaction with the txGasDimensionByOpcode tracer
// and does some minimal validation of the result
func callDebugTraceTransactionWithTxGasDimensionByOpcodeTracer(
	t *testing.T,
	ctx context.Context,
	testClient *TestClient,
	txHash common.Hash,
) OpcodeSumTraceResult {
	t.Helper()

	// Call debug_traceTransaction via the execution client (geth, nethermind, or comparison)
	tracerConfig := map[string]interface{}{
		"tracer": "txGasDimensionByOpcode",
		"tracerConfig": map[string]interface{}{
			"debug": true,
		},
	}
	traceResult, err := testClient.DebugTraceTransactionByOpcode(ctx, txHash, tracerConfig)
	Require(t, err)

	// Validate basic structure
	if traceResult.GasUsed == 0 {
		Fatal(t, "Expected non-zero gas usage")
	}
	if traceResult.GasUsedForL1 == 0 {
		Fatal(t, "Expected non-zero gas usage for L1")
	}
	if traceResult.GasUsedForL2 == 0 {
		Fatal(t, "Expected non-zero gas usage for L2")
	}
	if traceResult.IntrinsicGas == 0 {
		Fatal(t, "Expected non-zero intrinsic gas")
	}
	if traceResult.Failed {
		Fatal(t, "Transaction should not have failed")
	}
	txHashHex := txHash.Hex()
	if traceResult.TxHash != txHashHex {
		Fatal(t, "Expected txHash %s, got %s", txHashHex, traceResult.TxHash)
	}
	if len(traceResult.Dimensions) == 0 {
		Fatal(t, "Expected non-empty dimension summary map")
	}
	return traceResult
}

func sumUpDimensionalGasCosts(
	t *testing.T,
	gasesByDimension map[string]native.GasesByDimension,
	intrinsicGas uint64,
	adjustedRefund uint64,
	rootIsPrecompileAdjustment uint64,
	rootIsStylusAdjustment uint64,
) (oneDimensionSum, allDimensionsSum uint64) {
	t.Helper()
	oneDimensionSum = intrinsicGas + rootIsPrecompileAdjustment + rootIsStylusAdjustment
	allDimensionsSum = intrinsicGas + rootIsPrecompileAdjustment + rootIsStylusAdjustment
	for _, gasByDimension := range gasesByDimension {
		oneDimensionSum += gasByDimension.OneDimensionalGasCost
		allDimensionsSum += gasByDimension.Computation + gasByDimension.StateAccess + gasByDimension.StateGrowth + gasByDimension.HistoryGrowth
	}
	if adjustedRefund > oneDimensionSum {
		Fatal(t, "adjustedRefund should never be greater than oneDimensionSum: %d > %d", adjustedRefund, oneDimensionSum)
	}
	oneDimensionSum -= adjustedRefund
	if adjustedRefund > allDimensionsSum {
		Fatal(t, "adjustedRefund should never be greater than allDimensionsSum: %d > %d", adjustedRefund, allDimensionsSum)
	}
	allDimensionsSum -= adjustedRefund
	return oneDimensionSum, allDimensionsSum
}

// basically all of the TxOp tests do the same checks, the only difference is the setup.
func TxOpTraceAndCheck(t *testing.T, ctx context.Context, testClient *TestClient, receipt *types.Receipt) {
	traceResult := callDebugTraceTransactionWithTxGasDimensionByOpcodeTracer(t, ctx, testClient, receipt.TxHash)

	if receipt.Status != traceResult.Status {
		Fatal(t, "Transaction success/failure status mismatch", receipt.Status, traceResult.Status)
	}

	CheckEqual(t, receipt.GasUsed, traceResult.GasUsed)
	CheckEqual(t, receipt.GasUsedForL1, traceResult.GasUsedForL1)
	CheckEqual(t, receipt.GasUsedForL2(), traceResult.GasUsedForL2)

	expectedGasUsed := receipt.GasUsedForL2()
	sumOneDimensionalGasCosts, sumAllGasCosts := sumUpDimensionalGasCosts(
		t,
		traceResult.Dimensions,
		traceResult.IntrinsicGas,
		traceResult.AdjustedRefund,
		traceResult.RootIsPrecompileAdjustment,
		traceResult.RootIsStylusAdjustment,
	)

	CheckEqual(t, expectedGasUsed, sumOneDimensionalGasCosts,
		fmt.Sprintf("expected gas used: %d, one-dim used: %d\nDimensions:\n%v", expectedGasUsed, sumOneDimensionalGasCosts, traceResult.Dimensions))
	CheckEqual(t, expectedGasUsed, sumAllGasCosts,
		fmt.Sprintf("expected gas used: %d, all used: %d\nDimensions:\n%v", expectedGasUsed, sumAllGasCosts, traceResult.Dimensions))
}
