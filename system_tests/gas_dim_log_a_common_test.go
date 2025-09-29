// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

type DimensionLogRes = native.DimensionLogRes
type TraceResult = native.ExecutionResult

const (
	ColdMinusWarmAccountAccessCost = params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929
	ColdMinusWarmSloadCost         = params.ColdSloadCostEIP2929 - params.WarmStorageReadCostEIP2929
)

// #########################################################################################################
// #########################################################################################################
//                          REGULAR COMPUTATION OPCODES (ADD, SWAP, ETC)
// #########################################################################################################
// #########################################################################################################

// Run a test where we set up an L2, then send a transaction
// that only has computation-only opcodes. Then call debug_traceTransaction
// with the txGasDimensionLogger tracer.
//
// we expect in this case to get back a json response, with the gas dimension logs
// containing only the computation-only opcodes and that the gas in the computation
// only opcodes is equal to the OneDimensionalGasCost.
func TestDimLogComputationOnlyOpcodes(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCounter)
	_, receipt := callOnContract(t, builder, auth, contract.NoSpecials)
	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)

	// Validate each log entry
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}

		// Check that OneDimensionalGasCost equals Computation for computation-only opcodes
		if log.OneDimensionalGasCost != log.Computation {
			Fatal(t, "Log entry %d: For computation-only opcode %s pc %d, expected OneDimensionalGasCost (%d) to equal Computation (%d): %v",
				i, log.Op, log.Pc, log.OneDimensionalGasCost, log.Computation, log)
		}
		// check that there are only computation-only opcodes
		if log.StateAccess != 0 || log.StateGrowth != 0 || log.HistoryGrowth != 0 {
			Fatal(t, "Log entry %d: For computation-only opcode %s pc %d, expected StateAccess (%d), StateGrowth (%d), HistoryGrowth (%d) to be 0: %v",
				i, log.Op, log.Pc, log.StateAccess, log.StateGrowth, log.HistoryGrowth, log)
		}

		// Validate error field
		Require(t, log.Err, "Log entry %d: Unexpected error: %v", i, log.Err)
	}
}

// #########################################################################################################
// #########################################################################################################
//                                            HELPER FUNCTIONS
// #########################################################################################################
// #########################################################################################################

// common setup for all gas_dimension_logger tests
func gasDimensionTestSetup(t *testing.T, expectRevert bool) (
	ctx context.Context,
	cancel context.CancelFunc,
	builder *NodeBuilder,
	auth bind.TransactOpts,
	cleanup func(),
) {
	t.Helper()
	ctx, cancel = context.WithCancel(context.Background())
	builder = NewNodeBuilder(ctx).DefaultConfig(t, true).WithPreBoldDeployment()
	builder.execConfig.Caching.Archive = true
	if expectRevert {
		builder.execConfig.Sequencer.MaxRevertGasReject = 0
	}
	cleanup = builder.Build(t)
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	return ctx, cancel, builder, auth, cleanup
}

// deploy the contract we want to deploy for this test
// wait for it to be included
func deployGasDimensionTestContract[C any](
	t *testing.T,
	builder *NodeBuilder,
	auth bind.TransactOpts,
	deployFunc func(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, C, error),
) (
	address common.Address,
	contract C,
) {
	t.Helper()
	address, tx, contract, err := deployFunc(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	return address, contract
}

// call whatever test function is required for the test on the contract
func callOnContract[F func(auth *bind.TransactOpts) (*types.Transaction, error)](
	t *testing.T,
	builder *NodeBuilder,
	auth bind.TransactOpts,
	testFunc F,
) (tx *types.Transaction, receipt *types.Receipt) {
	t.Helper()
	tx, err := testFunc(&auth) // For write operations
	Require(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	return tx, receipt
}

// call whatever test function is required for the test on the contract
// pass in the argument provided to the test function call as its first argument
func callOnContractWithOneArg[A any, F func(auth *bind.TransactOpts, arg1 A) (*types.Transaction, error)](
	t *testing.T,
	builder *NodeBuilder,
	auth bind.TransactOpts,
	testFunc F,
	arg1 A,
) (receipt *types.Receipt) {
	t.Helper()
	tx, err := testFunc(&auth, arg1)
	Require(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	return receipt
}

// call debug_traceTransaction with txGasDimensionLogger tracer
// do very light sanity checks on the result
func callDebugTraceTransactionWithLogger(
	t *testing.T,
	ctx context.Context,
	builder *NodeBuilder,
	txHash common.Hash,
) TraceResult {
	t.Helper()
	// Call debug_traceTransaction with txGasDimensionLogger tracer
	rpcClient := builder.L2.ConsensusNode.Stack.Attach()
	var result json.RawMessage
	err := rpcClient.CallContext(ctx, &result, "debug_traceTransaction", txHash, map[string]interface{}{
		"tracer": "txGasDimensionLogger",
		"tracerConfig": map[string]interface{}{
			"debug": true,
		},
	})
	Require(t, err)

	// Parse the result
	var traceResult TraceResult
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
	if len(traceResult.DimensionLogs) == 0 {
		Fatal(t, "Expected non-empty dimension logs")
	}
	return traceResult
}

// get dimension log at position index of that opcode
// desiredIndex is 0-indexed
func getSpecificDimensionLogAtIndex(
	t *testing.T,
	dimensionLogs []DimensionLogRes,
	expectedOpcode string,
	expectedCount uint64,
	desiredIndex uint64,
) (
	specificDimensionLog *DimensionLogRes,
) {
	t.Helper()
	specificDimensionLog = nil
	var observedOpcodeCount uint64 = 0

	for i, log := range dimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry ", i, " Expected non-empty opcode")
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry ", i, " Expected depth >= 1, got", log.Depth)
		}
		if log.Err != nil {
			Fatal(t, "Log entry ", i, " Unexpected error:", log.Err)
		}
		if log.Op == expectedOpcode {
			if observedOpcodeCount == desiredIndex {
				specificDimensionLog = &log
			}
			observedOpcodeCount++
		}
	}
	if observedOpcodeCount != expectedCount {
		Fatal(t, "Expected ", expectedCount, " ", expectedOpcode, " got ", observedOpcodeCount)
	}
	if specificDimensionLog == nil {
		Fatal(t, "Expected to find log at index ", desiredIndex, " of ", expectedOpcode, " got nil")
	}
	return specificDimensionLog
}

// highlight one specific dimension log you want to get out of the
// dimension logs and return it. Make some basic field validation checks on the
// log while you iterate through it.
func getSpecificDimensionLog(t *testing.T, dimensionLogs []DimensionLogRes, expectedOpcode string) (
	specificDimensionLog *DimensionLogRes,
) {
	t.Helper()
	return getSpecificDimensionLogAtIndex(t, dimensionLogs, expectedOpcode, 1, 0)
}

// for the sstore multiple tests, we need to get the second sstore in the transaction
// but there should only ever be two sstores in the transaction.
func getLastOfTwoDimensionLogs(t *testing.T, dimensionLogs []DimensionLogRes, expectedOpcode string) (
	specificDimensionLog *DimensionLogRes,
) {
	t.Helper()
	return getSpecificDimensionLogAtIndex(t, dimensionLogs, expectedOpcode, 2, 1)
}

// just to reduce visual clutter in parameters
type ExpectedGasCosts struct {
	OneDimensionalGasCost uint64
	Computation           uint64
	StateAccess           uint64
	StateGrowth           uint64
	HistoryGrowth         uint64
	StateGrowthRefund     int64
	ChildExecutionCost    uint64
}

// checks that all of the fields of the expected and actual dimension logs are equal
func checkGasDimensionsMatch(t *testing.T, expected ExpectedGasCosts, actual *DimensionLogRes) {
	t.Helper()
	if actual.Computation != expected.Computation {
		Fatal(t, "Expected Computation ", expected.Computation, " got ", actual.Computation, " actual: ", actual.DebugString())
	}
	if actual.StateAccess != expected.StateAccess {
		Fatal(t, "Expected StateAccess ", expected.StateAccess, " got ", actual.StateAccess, " actual: ", actual.DebugString())
	}
	if actual.StateGrowth != expected.StateGrowth {
		Fatal(t, "Expected StateGrowth ", expected.StateGrowth, " got ", actual.StateGrowth, " actual: ", actual.DebugString())
	}
	if actual.HistoryGrowth != expected.HistoryGrowth {
		Fatal(t, "Expected HistoryGrowth ", expected.HistoryGrowth, " got ", actual.HistoryGrowth, " actual: ", actual.DebugString())
	}
	if actual.StateGrowthRefund != expected.StateGrowthRefund {
		Fatal(t, "Expected StateGrowthRefund ", expected.StateGrowthRefund, " got ", actual.StateGrowthRefund, " actual: ", actual.DebugString())
	}
	if actual.OneDimensionalGasCost != expected.OneDimensionalGasCost {
		Fatal(t, "Expected OneDimensionalGasCost ", expected.OneDimensionalGasCost, " got ", actual.OneDimensionalGasCost, " actual: ", actual.DebugString())
	}
	if actual.ChildExecutionCost != expected.ChildExecutionCost {
		Fatal(t, "Expected ChildExecutionCost ", expected.ChildExecutionCost, " got ", actual.ChildExecutionCost, " actual: ", actual.DebugString())
	}
}

// check that the one dimensional gas cost is equal to the sum of the other gas dimensions
func checkGasDimensionsEqualOneDimensionalGas(
	t *testing.T,
	l *DimensionLogRes,
) {
	t.Helper()
	if l.OneDimensionalGasCost != l.Computation+l.StateAccess+l.StateGrowth+l.HistoryGrowth {
		Fatal(t, "Expected OneDimensionalGasCost to equal sum of gas dimensions", l.DebugString())
	}
}
