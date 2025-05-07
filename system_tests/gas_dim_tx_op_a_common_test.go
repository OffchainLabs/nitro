package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/offchainlabs/nitro/solgen/go/gasdimensionsgen"
)

type OpcodeSumTraceResult = native.TxGasDimensionByOpcodeExecutionResult

// #########################################################################################################
// #########################################################################################################
//                          REGULAR COMPUTATION OPCODES (ADD, SWAP, ETC)
// #########################################################################################################
// #########################################################################################################

// this test tests if the tracer calculates the same gas
// used for a transaction as the TX receipt, for
// computation-only opcodes.
func TestDimTxOpComputationOnlyOpcodes(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCounter)
	_, receipt := callOnContract(t, builder, auth, contract.NoSpecials)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// #########################################################################################################
// #########################################################################################################
//                                            HELPER FUNCTIONS
// #########################################################################################################
// #########################################################################################################

func callDebugTraceTransactionWithTxGasDimensionByOpcodeTracer(
	t *testing.T,
	ctx context.Context,
	builder *NodeBuilder,
	txHash common.Hash,
) OpcodeSumTraceResult {
	t.Helper()
	// Call debug_traceTransaction with txGasDimensionLogger tracer
	rpcClient := builder.L2.ConsensusNode.Stack.Attach()
	var result json.RawMessage
	err := rpcClient.CallContext(ctx, &result, "debug_traceTransaction", txHash, map[string]interface{}{
		"tracer": "txGasDimensionByOpcode",
	})
	Require(t, err)

	// Parse the result
	var traceResult OpcodeSumTraceResult
	if err := json.Unmarshal(result, &traceResult); err != nil {
		Fatal(t, err)
	}

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
) (oneDimensionSum, allDimensionsSum uint64) {
	t.Helper()
	oneDimensionSum = intrinsicGas
	allDimensionsSum = intrinsicGas
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
func TxOpTraceAndCheck(t *testing.T, ctx context.Context, builder *NodeBuilder, receipt *types.Receipt) {
	traceResult := callDebugTraceTransactionWithTxGasDimensionByOpcodeTracer(t, ctx, builder, receipt.TxHash)

	CheckEqual(t, receipt.GasUsed, traceResult.GasUsed)
	CheckEqual(t, receipt.GasUsedForL1, traceResult.GasUsedForL1)
	CheckEqual(t, receipt.GasUsedForL2(), traceResult.GasUsedForL2)

	expectedGasUsed := receipt.GasUsedForL2()
	sumOneDimensionalGasCosts, sumAllGasCosts := sumUpDimensionalGasCosts(t, traceResult.Dimensions, traceResult.IntrinsicGas, traceResult.AdjustedRefund)

	CheckEqual(t, expectedGasUsed, sumOneDimensionalGasCosts,
		fmt.Sprintf("expected gas used: %d, one-dim used: %d\nDimensions:\n%v", expectedGasUsed, sumOneDimensionalGasCosts, traceResult.Dimensions))
	CheckEqual(t, expectedGasUsed, sumAllGasCosts,
		fmt.Sprintf("expected gas used: %d, all used: %d\nDimensions:\n%v", expectedGasUsed, sumAllGasCosts, traceResult.Dimensions))
}
