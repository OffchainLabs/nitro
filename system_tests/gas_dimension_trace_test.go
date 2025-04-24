package arbtest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/offchainlabs/nitro/solgen/go/gasdimensionsgen"
)

type DimensionLogRes = native.DimensionLogRes
type TraceResult = native.ExecutionResult

// Run a test where we set up an L2, then send a transaction
// that only has computation-only opcodes. Then call debug_traceTransaction
// with the txGasDimensionLogger tracer.
//
// we expect in this case to get back a json response, with the gas dimension logs
// containing only the computation-only opcodes and that the gas in the computation
// only opcodes is equal to the OneDimensionalGasCost.
func TestGasDimensionLoggerComputationOnlyOpcodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig.Caching.Archive = true
	// For now Archive node should use HashScheme
	builder.execConfig.Caching.StateScheme = rawdb.HashScheme
	cleanup := builder.Build(t)
	defer cleanup()
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployCounter(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	if err != nil {
		t.Fatal(err)
	}

	// 4. Now you can interact with the contract
	tx, err = contract.Increment(&auth) // For write operations
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	if err != nil {
		t.Fatal(err)
	}

	// Call debug_traceTransaction with txGasDimensionLogger tracer
	rpcClient := builder.L2.ConsensusNode.Stack.Attach()
	var result json.RawMessage
	err = rpcClient.CallContext(ctx, &result, "debug_traceTransaction", receipt.TxHash, map[string]interface{}{
		"tracer": "txGasDimensionLogger",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Parse the result
	var traceResult TraceResult
	if err := json.Unmarshal(result, &traceResult); err != nil {
		t.Fatal(err)
	}

	// Validate basic structure
	if traceResult.Gas == 0 {
		t.Error("Expected non-zero gas usage")
	}
	if traceResult.Failed {
		t.Error("Transaction should not have failed")
	}
	if traceResult.TxHash != receipt.TxHash.Hex() {
		t.Errorf("Expected txHash %s, got %s", receipt.TxHash.Hex(), traceResult.TxHash)
	}
	if len(traceResult.DimensionLogs) == 0 {
		t.Error("Expected non-empty dimension logs")
	}

	// Validate each log entry
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Pc == 0 {
			t.Errorf("Log entry %d: Expected non-zero PC", i)
		}
		if log.Op == "" {
			t.Errorf("Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			t.Errorf("Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}

		// Check that OneDimensionalGasCost equals Computation for computation-only opcodes
		if log.OneDimensionalGasCost != log.Computation {
			t.Errorf("Log entry %d: For computation-only opcode %s pc %d, expected OneDimensionalGasCost (%d) to equal Computation (%d): %v",
				i, log.Op, log.Pc, log.OneDimensionalGasCost, log.Computation, log)
		}
		// check that there are only computation-only opcodes
		if log.StateAccess != 0 || log.StateGrowth != 0 || log.HistoryGrowth != 0 {
			t.Errorf("Log entry %d: For computation-only opcode %s pc %d, expected StateAccess (%d), StateGrowth (%d), HistoryGrowth (%d) to be 0: %v",
				i, log.Op, log.Pc, log.StateAccess, log.StateGrowth, log.HistoryGrowth, log)
		}

		// Validate error field
		if log.Err != nil {
			t.Errorf("Log entry %d: Unexpected error: %v", i, log.Err)
		}
	}
}
