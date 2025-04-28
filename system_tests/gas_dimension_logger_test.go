package arbtest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
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
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployCounter(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.NoSpecials(&auth) // For write operations
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)

	// Validate each log entry
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
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

// BALANCE, EXTCODESIZE, EXTCODEHASH are all read-only operations on state access
// this test deployes a contract that calls BALANCE on a cold access list address
//
// on the cold BALANCE, we expect the total one-dimensional gas cost to be 2600
// the computation to be 100 (for the warm access cost of the address)
// and the state access to be 2500 (for the cold access cost of the address)
// and all other gas dimensions to be 0
func TestGasDimensionLoggerBalanceCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployBalance(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.CallBalanceCold(&auth) // For write operations
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var balanceCount uint64 = 0
	var balanceLog *DimensionLogRes

	// there should only be one BALANCE in the entire trace
	// go through and grab it and its data
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}
		if log.Err != nil {
			Fatal(t, "Log entry %d: Unexpected error: %v", i, log.Err)
		}
		if log.Op == "BALANCE" {
			balanceCount++
			balanceLog = &log
		}
	}
	if balanceCount != 1 {
		Fatal(t, "Expected 1 BALANCE, got %d", balanceCount)
	}
	if balanceLog == nil {
		Fatal(t, "Expected BALANCE log, got nil")
	}

	checkDimensionLogGasCostsEqual(
		t,
		ExpectedGasCosts{
			OneDimensionalGasCost: 2600,
			Computation:           100,
			StateAccess:           2500,
			StateGrowth:           0,
			HistoryGrowth:         0,
			StateGrowthRefund:     0,
		},
		balanceLog,
	)
}

// BALANCE, EXTCODESIZE, EXTCODEHASH are all read-only operations on state access
// this test deployes a contract that calls BALANCE on a warm access list address
//
// on the warm BALANCE, we expect the total one-dimensional gas cost to be 100
// the computation to be 100 (for the warm access cost of the address)
// and all other gas dimensions to be 0
func TestGasDimensionLoggerBalanceWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployBalance(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.CallBalanceWarm(&auth) // For write operations
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var balanceCount uint64 = 0
	var balanceLog *DimensionLogRes

	// there should only be one BALANCE in the entire trace
	// go through and grab it and its data
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}
		if log.Err != nil {
			Fatal(t, "Log entry %d: Unexpected error: %v", i, log.Err)
		}
		if log.Op == "BALANCE" {
			balanceCount++
			balanceLog = &log
		}
	}
	if balanceCount != 1 {
		Fatal(t, "Expected 1 BALANCE, got %d", balanceCount)
	}
	if balanceLog == nil {
		Fatal(t, "Expected BALANCE log, got nil")
	}

	checkDimensionLogGasCostsEqual(
		t,
		ExpectedGasCosts{
			OneDimensionalGasCost: 100,
			Computation:           100,
			StateAccess:           0,
			StateGrowth:           0,
			HistoryGrowth:         0,
			StateGrowthRefund:     0,
		},
		balanceLog,
	)
}

// BALANCE, EXTCODESIZE, EXTCODEHASH are all read-only operations on state access
// this test deployes a contract that calls EXTCODESIZE on a cold access list address
//
// on the cold EXTCODESIZE, we expect the total one-dimensional gas cost to be 2600
// the computation to be 100 (for the warm access cost of the address)
// and the state access to be 2500 (for the cold access cost of the address)
// and all other gas dimensions to be 0
func TestGasDimensionLoggerExtCodeSizeCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployExtCodeSize(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.GetExtCodeSizeCold(&auth) // For write operations
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var extCodeSizeCount uint64 = 0
	var extCodeSizeLog *DimensionLogRes

	// there should only be one EXTCODESIZE in the entire trace
	// go through and grab it and its data
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}
		if log.Err != nil {
			Fatal(t, "Log entry %d: Unexpected error: %v", i, log.Err)
		}
		if log.Op == "EXTCODESIZE" {
			extCodeSizeCount++
			extCodeSizeLog = &log
		}
	}
	if extCodeSizeCount != 1 {
		Fatal(t, "Expected 1 EXTCODESIZE, got %d", extCodeSizeCount)
	}
	if extCodeSizeLog == nil {
		Fatal(t, "Expected EXTCODESIZE log, got nil")
	}

	checkDimensionLogGasCostsEqual(
		t,
		ExpectedGasCosts{
			OneDimensionalGasCost: 2600,
			Computation:           100,
			StateAccess:           2500,
			StateGrowth:           0,
			HistoryGrowth:         0,
			StateGrowthRefund:     0,
		},
		extCodeSizeLog,
	)
}

// BALANCE, EXTCODESIZE, EXTCODEHASH are all read-only operations on state access
// this test deployes a contract that calls EXTCODESIZE on a warm access list address
//
// on the warm EXTCODESIZE, we expect the total one-dimensional gas cost to be 100
// the computation to be 100 (for the warm access cost of the address)
// and all other gas dimensions to be 0
func TestGasDimensionLoggerExtCodeSizeWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployExtCodeSize(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.GetExtCodeSizeWarm(&auth) // For write operations
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var extCodeSizeCount uint64 = 0
	var extCodeSizeLog *DimensionLogRes

	// there should only be one EXTCODESIZE in the entire trace
	// go through and grab it and its data
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}
		if log.Err != nil {
			Fatal(t, "Log entry %d: Unexpected error: %v", i, log.Err)
		}
		if log.Op == "EXTCODESIZE" {
			extCodeSizeCount++
			extCodeSizeLog = &log
		}
	}
	if extCodeSizeCount != 1 {
		Fatal(t, "Expected 1 EXTCODESIZE, got %d", extCodeSizeCount)
	}
	if extCodeSizeLog == nil {
		Fatal(t, "Expected EXTCODESIZE log, got nil")
	}

	checkDimensionLogGasCostsEqual(
		t,
		ExpectedGasCosts{
			OneDimensionalGasCost: 100,
			Computation:           100,
			StateAccess:           0,
			StateGrowth:           0,
			HistoryGrowth:         0,
			StateGrowthRefund:     0,
		},
		extCodeSizeLog,
	)
}

// BALANCE, EXTCODESIZE, EXTCODEHASH are all read-only operations on state access
// this test deployes a contract that calls EXTCODEHASH on a cold access list address
//
// on the cold EXTCODEHASH, we expect the total one-dimensional gas cost to be 2600
// the computation to be 100 (for the warm access cost of the address)
// and the state access to be 2500 (for the cold access cost of the address)
// and all other gas dimensions to be 0
func TestGasDimensionLoggerExtCodeHashCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployExtCodeHash(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.GetExtCodeHashCold(&auth) // For write operations
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var extCodeHashCount uint64 = 0
	var extCodeHashLog *DimensionLogRes

	// there should only be one EXTCODEHASH in the entire trace
	// go through and grab it and its data
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}
		if log.Err != nil {
			Fatal(t, "Log entry %d: Unexpected error: %v", i, log.Err)
		}
		if log.Op == "EXTCODEHASH" {
			extCodeHashCount++
			extCodeHashLog = &log
		}
	}
	if extCodeHashCount != 1 {
		Fatal(t, "Expected 1 EXTCODEHASH, got %d", extCodeHashCount)
	}
	if extCodeHashLog == nil {
		Fatal(t, "Expected EXTCODEHASH log, got nil")
	}

	checkDimensionLogGasCostsEqual(
		t,
		ExpectedGasCosts{
			OneDimensionalGasCost: 2600,
			Computation:           100,
			StateAccess:           2500,
			StateGrowth:           0,
			HistoryGrowth:         0,
			StateGrowthRefund:     0,
		},
		extCodeHashLog,
	)
}

// BALANCE, EXTCODESIZE, EXTCODEHASH are all read-only operations on state access
// this test deployes a contract that calls EXTCODEHASH on a warm access list address
//
// on the warm EXTCODEHASH, we expect the total one-dimensional gas cost to be 100
// the computation to be 100 (for the warm access cost of the address)
// and all other gas dimensions to be 0
func TestGasDimensionLoggerExtCodeHashWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployExtCodeHash(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.GetExtCodeHashWarm(&auth) // For write operations
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var extCodeHashCount uint64 = 0
	var extCodeHashLog *DimensionLogRes

	// there should only be one EXTCODEHASH in the entire trace
	// go through and grab it and its data
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}
		if log.Err != nil {
			Fatal(t, "Log entry %d: Unexpected error: %v", i, log.Err)
		}
		if log.Op == "EXTCODEHASH" {
			extCodeHashCount++
			extCodeHashLog = &log
		}
	}
	if extCodeHashCount != 1 {
		Fatal(t, "Expected 1 EXTCODEHASH, got %d", extCodeHashCount)
	}
	if extCodeHashLog == nil {
		Fatal(t, "Expected EXTCODEHASH log, got nil")
	}

	checkDimensionLogGasCostsEqual(
		t,
		ExpectedGasCosts{
			OneDimensionalGasCost: 100,
			Computation:           100,
			StateAccess:           0,
			StateGrowth:           0,
			HistoryGrowth:         0,
			StateGrowthRefund:     0,
		},
		extCodeHashLog,
	)
}

// In this test we deploy a contract with a function that all it does
// is perform an sload on a cold slot that has not been touched yet
//
// on the cold sload, we expect the total one-dimensional gas cost to be 2100
// the computation to be 100 (for the warm base access cost)
// the state access to be 2000 (for the cold sload cost)
// all others zero
func TestGasDimensionLoggerSloadCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeploySload(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.ColdSload(&auth)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var sloadCount uint64 = 0
	var sloadLog *DimensionLogRes

	// there should only be one SLOAD in the entire trace
	// go through and grab it and its data
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}
		if log.Err != nil {
			Fatal(t, "Log entry %d: Unexpected error: %v", i, log.Err)
		}
		if log.Op == "SLOAD" {
			sloadCount++
			sloadLog = &log
		}
	}
	if sloadCount != 1 {
		Fatal(t, "Expected 1 SLOAD, got %d", sloadCount)
	}
	if sloadLog == nil {
		Fatal(t, "Expected SLOAD log, got nil")
	}

	checkDimensionLogGasCostsEqual(
		t,
		ExpectedGasCosts{
			OneDimensionalGasCost: 2100,
			Computation:           100,
			StateAccess:           2000,
			StateGrowth:           0,
			HistoryGrowth:         0,
			StateGrowthRefund:     0,
		},
		sloadLog,
	)
}

// In this test we deploy a contract with a function that all it does
// is perform an sload on an already warm slot (by SSTORE-ing to the slot first)
//
// on the warm sload, we expect the total one-dimensional gas cost to be 100
// the computation to be 100 (for the warm base access cost)
// all others zero
func TestGasDimensionLoggerSloadWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeploySload(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.WarmSload(&auth)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var sloadCount uint64 = 0
	var sloadLog *DimensionLogRes

	// there should only be one SLOAD in the entire trace
	// go through and grab it and its data
	for i, log := range traceResult.DimensionLogs {
		// Basic field validation
		if log.Op == "" {
			Fatal(t, "Log entry %d: Expected non-empty opcode", i)
		}
		if log.Depth < 1 {
			Fatal(t, "Log entry %d: Expected depth >= 1, got %d", i, log.Depth)
		}
		if log.Err != nil {
			Fatal(t, "Log entry %d: Unexpected error: %v", i, log.Err)
		}
		if log.Op == "SLOAD" {
			sloadCount++
			sloadLog = &log
		}
	}
	if sloadCount != 1 {
		Fatal(t, "Expected 1 SLOAD, got %d", sloadCount)
	}
	if sloadLog == nil {
		Fatal(t, "Expected SLOAD log, got nil")
	}

	// on the warm sload, we expect the total one-dimensional gas cost to be 100
	// the computation to be 100 (for the warm base access cost)
	// all others zero
	checkDimensionLogGasCostsEqual(
		t,
		ExpectedGasCosts{
			OneDimensionalGasCost: 100,
			Computation:           100,
			StateAccess:           0,
			StateGrowth:           0,
			HistoryGrowth:         0,
			StateGrowthRefund:     0,
		},
		sloadLog,
	)
}

// ############################################################################
//                                HELPER FUNCTIONS
// ############################################################################

// common setup for all gas_dimension_logger tests
func gasDimensionLoggerSetup(t *testing.T) (
	ctx context.Context,
	cancel context.CancelFunc,
	builder *NodeBuilder,
	auth bind.TransactOpts,
	cleanup func(),
) {
	t.Helper()
	ctx, cancel = context.WithCancel(context.Background())
	builder = NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig.Caching.Archive = true
	// For now Archive node should use HashScheme
	builder.execConfig.Caching.StateScheme = rawdb.HashScheme
	cleanup = builder.Build(t)
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	return ctx, cancel, builder, auth, cleanup
}

// call debug_traceTransaction with txGasDimensionLogger tracer
// do very light sanity checks on the result
func callDebugTraceTransaction(
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
	})
	Require(t, err)

	// Parse the result
	var traceResult TraceResult
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
	if len(traceResult.DimensionLogs) == 0 {
		Fatal(t, "Expected non-empty dimension logs")
	}
	return traceResult
}

// just to reduce visual clutter in parameters
type ExpectedGasCosts struct {
	OneDimensionalGasCost uint64
	Computation           uint64
	StateAccess           uint64
	StateGrowth           uint64
	HistoryGrowth         uint64
	StateGrowthRefund     int64
}

func checkDimensionLogGasCostsEqual(
	t *testing.T,
	expected ExpectedGasCosts,
	actual *DimensionLogRes,
) {
	t.Helper()
	if actual.OneDimensionalGasCost != expected.OneDimensionalGasCost {
		Fatal(t, "Expected OneDimensionalGasCost ", expected.OneDimensionalGasCost, " got ", actual.OneDimensionalGasCost)
	}
	if actual.Computation != expected.Computation {
		Fatal(t, "Expected Computation ", expected.Computation, " got ", actual.Computation)
	}
	if actual.StateAccess != expected.StateAccess {
		Fatal(t, "Expected StateAccess ", expected.StateAccess, " got ", actual.StateAccess)
	}
	if actual.StateGrowth != expected.StateGrowth {
		Fatal(t, "Expected StateGrowth ", expected.StateGrowth, " got ", actual.StateGrowth)
	}
	if actual.HistoryGrowth != expected.HistoryGrowth {
		Fatal(t, "Expected HistoryGrowth ", expected.HistoryGrowth, " got ", actual.HistoryGrowth)
	}
	if actual.StateGrowthRefund != expected.StateGrowthRefund {
		Fatal(t, "Expected StateGrowthRefund ", expected.StateGrowthRefund, " got ", actual.StateGrowthRefund)
	}
}
