package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/params"
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
	tx, receipt := callOnContract(t, builder, auth, contract.NoSpecials)
	intrinsicTxCost := getIntrinsicTxCost(t, tx)
	traceResult := callDebugTraceTransactionWithTxGasDimensionByOpcodeTracer(t, ctx, builder, receipt.TxHash)

	expectedGasUsed := receipt.GasUsedForL2()
	sumOneDimensionalGasCosts, sumAllGasCosts := sumUpDimensionalGasCosts(t, traceResult.Dimensions)
	sumOneDimensionalGasCosts += intrinsicTxCost
	sumAllGasCosts += intrinsicTxCost

	CheckEqual(t, expectedGasUsed, sumOneDimensionalGasCosts,
		fmt.Sprintf("expected gas used: %d, one-dim used: %d\nDimensions:\n%v", expectedGasUsed, sumOneDimensionalGasCosts, traceResult.Dimensions))
	CheckEqual(t, expectedGasUsed, sumAllGasCosts,
		fmt.Sprintf("expected gas used: %d, all used: %d\nDimensions:\n%v", expectedGasUsed, sumAllGasCosts, traceResult.Dimensions))
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
	if traceResult.Gas == 0 {
		Fatal(t, "Expected non-zero gas usage")
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
) (oneDimensionSum, allDimensionsSum uint64) {
	t.Helper()
	oneDimensionSum = 0
	allDimensionsSum = 0
	var sumRefunds int64 = 0
	for _, gasByDimension := range gasesByDimension {
		oneDimensionSum += gasByDimension.OneDimensionalGasCost
		allDimensionsSum += gasByDimension.Computation + gasByDimension.StateAccess + gasByDimension.StateGrowth + gasByDimension.HistoryGrowth
		sumRefunds += gasByDimension.StateGrowthRefund
	}
	if sumRefunds < 0 {
		Fatal(t, "sumRefunds should never be negative: %d", sumRefunds)
	}
	sumRefundsUint64 := uint64(sumRefunds)
	if sumRefundsUint64 > oneDimensionSum {
		Fatal(t, "sumRefunds should never be greater than oneDimensionSum: %d > %d", sumRefundsUint64, oneDimensionSum)
	}
	oneDimensionSum -= sumRefundsUint64
	if sumRefundsUint64 > allDimensionsSum {
		Fatal(t, "sumRefunds should never be greater than allDimensionsSum: %d > %d", sumRefundsUint64, allDimensionsSum)
	}
	allDimensionsSum -= sumRefundsUint64
	return oneDimensionSum, allDimensionsSum
}

func getIntrinsicTxCost(t *testing.T, tx *types.Transaction) uint64 {
	t.Helper()
	data := tx.Data()
	var zeroBytes, nonZeroBytes uint64 = 0, 0
	for _, b := range data {
		if b == 0 {
			zeroBytes++
		} else {
			nonZeroBytes++
		}
	}
	return params.TxDataZeroGas*zeroBytes + params.TxDataNonZeroGasEIP2028*nonZeroBytes + params.TxGas
}
