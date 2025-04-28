package arbtest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/gasdimensionsgen"
)

type DimensionLogRes = native.DimensionLogRes
type TraceResult = native.ExecutionResult

const (
	ColdMinusWarmAccountAccessCost = params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929
	ColdMinusWarmSloadCost         = params.ColdSloadCostEIP2929 - params.WarmStorageReadCostEIP2929
	ColdAccountAccessCost          = params.ColdAccountAccessCostEIP2929
	ColdSloadCost                  = params.ColdSloadCostEIP2929
	WarmStorageReadCost            = params.WarmStorageReadCostEIP2929
)

// ############################################################
//      REGULAR COMPUTATION OPCODES (ADD, SWAP, ETC)
// ############################################################

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

// ############################################################
// SIMPLE STATE ACCESS OPCODES (BALANCE, EXTCODESIZE, EXTCODEHASH)
// ############################################################

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

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, balanceLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
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

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, balanceLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
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

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeSizeLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
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

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeSizeLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
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

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeHashLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
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

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeHashLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeHashLog,
	)
}

// ############################################################
//                        SLOAD
// ############################################################

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

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdSloadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmSloadCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, sloadLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
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

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, sloadLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		sloadLog,
	)
}

// ############################################################
//                        EXTCODECOPY
// ############################################################

// EXTCODECOPY reads from state and copies code to memory
// for gas dimensions, we don't care about expanding memory, but
// we do care about the cost being correct
//
// EXTCODECOPY has three components to its gas cost:
// 1. minimum_word_size = (size + 31) / 32
// 2. memory_expansion_cost
// 3. address_access_cost - the access set.
// gas for extcodecopy is 3 * minimum_word_size + memory_expansion_cost + address_access_cost
// 3*minimum_word_size is always state access
//
// Here is the blob of code for the contract that we are copying:
// "608060405234801561000f575f5ffd5b506004361061004a575f3560e01c8063",
// "3fb5c1cb1461004e578063822ec8611461006a5780638381f58a146100745780",
// "63d09de08a14610092575b5f5ffd5b6100686004803603810190610063919061",
// "011b565b61009c565b005b6100726100a5565b005b61007c6100b1565b604051",
// "6100899190610155565b60405180910390f35b61009a6100b6565b005b805f81",
// "90555050565b61696961133701602081f35b5f5481565b5f5f54905060018161",
// "00c8919061019b565b90505f5f54905080826100db919061019b565b5f819055",
// "505050565b5f5ffd5b5f819050919050565b6100fa816100e8565b8114610104",
// "575f5ffd5b50565b5f81359050610115816100f1565b92915050565b5f602082",
// "840312156101305761012f6100e4565b5b5f61013d84828501610107565b9150",
// "5092915050565b61014f816100e8565b82525050565b5f602082019050610168",
// "5f830184610146565b92915050565b7f4e487b71000000000000000000000000",
// "000000000000000000000000000000005f52601160045260245ffd5b5f6101a5",
// "826100e8565b91506101b0836100e8565b92508282019050808211156101c857",
// "6101c761016e565b5b9291505056fea264697066735822122056d73a5a32faf2",
// "0913b0a82eef9159812447f4e5e86362af90bcb20669ddf7bc64736f6c634300",
// "081c003300000000000000000000000000000000000000000000000000000000"
//
// observe that the code size is 516 bytes, and there are 17 256-bit (32 byte)
// long words of data in of this code thus, the minimum word size is 17
var extCodeCopyWordSize uint64 = 17

// the minimum word cost is the minimum word size * 3, and it is always
// read-write state access since this cost is associated with the copying
var extCodeCopyMinimumWordCost uint64 = extCodeCopyWordSize * 3

// Above we show the contract code that is copied.
// the memory size at time of copy for all of the test cases
// is 704 bytes (22 words).
// In the memory expansion cases, we copy starting at offset 703
// out of 704, forcing the memory to expand. It expands from
// 704 bytes to 1248 bytes, because the code size is 516 bytes
// (1219 bytes) which then gets pushed out to 39 words - 1248 bytes.
//
// the formula for memory expansion is:
// memory_size_word = (memory_byte_size + 31) / 32
// memory_cost = (memory_size_word ** 2) / 512 + (3 * memory_size_word)
// memory_expansion_cost = new_memory_cost - last_memory_cost
//
// we care about the last_memory_cost, that happens at PC 309
// when the CALLDATACOPY is executed for the
// line of solidity: bytes memory localCode = new bytes(codeSize);
// in that case the memory size increased from 160 to 704 bytes
// 704 bytes is 22 words.
//
// so we have memory_expansion_cost =
// (39 ** 2) / 512 + (3 * 39) - (22 ** 2) / 512 - (3 * 22)
// = 119 - 66 = 53
var extCodeCopyMemoryExpansionCost uint64 = 53

// EXTCODECOPY reads from state and copies code to memory
// for gas dimensions, we don't care about expanding memory, but
// we do care about the cost being correct
//
// this test checks the cost of EXTCODECOPY when the code is cold
// and there is no memory expansion. We expect the cost to be
// be 2600, the computation to be 100, the state access to be
// 2500 + the minimum word cost,
// and all other gas dimensions to be 0
func TestGasDimensionLoggerExtCodeCopyColdNoMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployExtCodeCopy(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.ExtCodeCopyColdNoMemExpansion(&auth)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var extCodeCopyCount uint64 = 0
	var extCodeCopyLog *DimensionLogRes

	// there should only be one EXTCODECOPY in the entire trace
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
		if log.Op == "EXTCODECOPY" {
			extCodeCopyCount++
			extCodeCopyLog = &log
		}
	}
	if extCodeCopyCount != 1 {
		Fatal(t, "Expected 1 EXTCODECOPY, got %d", extCodeCopyCount)
	}
	if extCodeCopyLog == nil {
		Fatal(t, "Expected EXTCODECOPY log, got nil")
	}

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost + extCodeCopyMinimumWordCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeCopyLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeCopyLog,
	)
}

// EXTCODECOPY reads from state and copies code to memory
// for gas dimensions, we don't care about expanding memory, but
// we do care about the cost being correct
//
// this test checks the cost of EXTCODECOPY when the code is cold
// and there is memory expansion. We expect the cost to be
// be 2600 + whatever the memory expansion cost happens to be,
// + the minimum word cost,
// the computation to be 100 + the memory expansion cost,
// the state access to be 2500 + the minimum word cost,
// and all other gas dimensions to be 0
func TestGasDimensionLoggerExtCodeCopyColdMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployExtCodeCopy(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.ExtCodeCopyColdMemExpansion(&auth)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var extCodeCopyCount uint64 = 0
	var extCodeCopyLog *DimensionLogRes

	// there should only be one EXTCODECOPY in the entire trace
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
		if log.Op == "EXTCODECOPY" {
			extCodeCopyCount++
			extCodeCopyLog = &log
		}
	}
	if extCodeCopyCount != 1 {
		Fatal(t, "Expected 1 EXTCODECOPY, got %d", extCodeCopyCount)
	}
	if extCodeCopyLog == nil {
		Fatal(t, "Expected EXTCODECOPY log, got nil")
	}

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost + extCodeCopyMemoryExpansionCost + extCodeCopyMinimumWordCost,
		Computation:           WarmStorageReadCost + extCodeCopyMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeCopyLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeCopyLog,
	)
}

// EXTCODECOPY reads from state and copies code to memory
// for gas dimensions, we don't care about expanding memory, but
// we do care about the cost being correct
//
// this test checks the cost of EXTCODECOPY when the code is warm
// and there is no memory expansion. We expect the cost to be
// be 100, the computation to be 100, the state access to be
// just the minimum word cost,
// and all other gas dimensions to be 0
func TestGasDimensionLoggerExtCodeCopyWarmNoMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployExtCodeCopy(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.ExtCodeCopyWarmNoMemExpansion(&auth)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var extCodeCopyCount uint64 = 0
	var extCodeCopyLog *DimensionLogRes

	// there should only be one EXTCODECOPY in the entire trace
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
		if log.Op == "EXTCODECOPY" {
			extCodeCopyCount++
			extCodeCopyLog = &log
		}
	}
	if extCodeCopyCount != 1 {
		Fatal(t, "Expected 1 EXTCODECOPY, got %d", extCodeCopyCount)
	}
	if extCodeCopyLog == nil {
		Fatal(t, "Expected EXTCODECOPY log, got nil")
	}

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost + extCodeCopyMinimumWordCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeCopyLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeCopyLog,
	)
}

// EXTCODECOPY reads from state and copies code to memory
// for gas dimensions, we don't care about expanding memory, but
// we do care about the cost being correct
//
// this test checks the cost of EXTCODECOPY when the code is warm
// and there is memory expansion. We expect the cost to be
// be 100 + whatever the memory expansion cost happens to be,
// + the minimum word cost,
// the computation to be 100 + the memory expansion cost,
// the state access to be the minimum word cost,
// and all other gas dimensions to be 0
func TestGasDimensionLoggerExtCodeCopyWarmMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	// 2. Deploy the contract
	_, tx, contract, err := gasdimensionsgen.DeployExtCodeCopy(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 4. Now you can interact with the contract
	tx, err = contract.ExtCodeCopyWarmMemExpansion(&auth)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	traceResult := callDebugTraceTransaction(t, ctx, builder, receipt.TxHash)
	var extCodeCopyCount uint64 = 0
	var extCodeCopyLog *DimensionLogRes

	// there should only be one EXTCODECOPY in the entire trace
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
		if log.Op == "EXTCODECOPY" {
			extCodeCopyCount++
			extCodeCopyLog = &log
		}
	}
	if extCodeCopyCount != 1 {
		Fatal(t, "Expected 1 EXTCODECOPY, got %d", extCodeCopyCount)
	}
	if extCodeCopyLog == nil {
		Fatal(t, "Expected EXTCODECOPY log, got nil")
	}

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost + extCodeCopyMemoryExpansionCost + extCodeCopyMinimumWordCost,
		Computation:           WarmStorageReadCost + extCodeCopyMemoryExpansionCost,
		StateAccess:           extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeCopyLog)
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeCopyLog,
	)
}

// ############################################################
//
//	DELEGATECALL & STATICCALL
//
// ############################################################
//
// DELEGATECALL and STATICCALL have many permutations
// warm or cold
// empty or non-empty code at target address

func TestGasDimensionLoggerDelegateCallEmptyCold(t *testing.T) {
}
func TestGasDimensionLoggerDelegateCallEmptyWarm(t *testing.T) {
}
func TestGasDimensionLoggerDelegateCallNonEmptyCold(t *testing.T) {
}
func TestGasDimensionLoggerDelegateCallNonEmptyWarm(t *testing.T) {
}
func TestGasDimensionLoggerStaticCallEmptyCold(t *testing.T) {
}
func TestGasDimensionLoggerStaticCallEmptyWarm(t *testing.T) {
}
func TestGasDimensionLoggerStaticCallNonEmptyCold(t *testing.T) {
}
func TestGasDimensionLoggerStaticCallNonEmptyWarm(t *testing.T) {
}

// ############################################################
//	             LOG0, LOG1, LOG2, LOG3, LOG4
// ############################################################

func TestGasDimensionLoggerLog0Empty(t *testing.T) {
}

func TestGasDimensionLoggerLog0NonEmpty(t *testing.T) {
}

func TestGasDimensionLoggerLog1Empty(t *testing.T) {
}

func TestGasDimensionLoggerLog1NonEmpty(t *testing.T) {
}

func TestGasDimensionLoggerLog2(t *testing.T) {}

func TestGasDimensionLoggerLog2ExtraData(t *testing.T) {}

func TestGasDimensionLoggerLog3(t *testing.T) {}

func TestGasDimensionLoggerLog3ExtraData(t *testing.T) {}

func TestGasDimensionLoggerLog4(t *testing.T) {}

func TestGasDimensionLoggerLog4ExtraData(t *testing.T) {}

// ############################################################
//	                    CREATE & CREATE2
// ############################################################
//
// CREATE and CREATE2 only have two permutations, whether or not you
// transfer value with the creation

func TestGasDimensionLoggerCreate(t *testing.T) {}

func TestGasDimensionLoggerCreateWithValue(t *testing.T) {}

func TestGasDimensionLoggerCreate2(t *testing.T) {}

func TestGasDimensionLoggerCreate2WithValue(t *testing.T) {}

// ############################################################
//                      CALL and CALLCODE
// ############################################################
//
// CALL and CALLCODE have many permutations
// warm or cold
// no value or value transfer with the call
// empty or non-empty code at target address

func TestGasDimensionLoggerCallEmptyColdNoValue(t *testing.T) {}

func TestGasDimensionLoggerCallEmptyColdWithValue(t *testing.T) {}

func TestGasDimensionLoggerCallEmptyWarmNoValue(t *testing.T) {}

func TestGasDimensionLoggerCallEmptyWarmWithValue(t *testing.T) {}

func TestGasDimensionLoggerCallNonEmptyColdNoValue(t *testing.T) {}

func TestGasDimensionLoggerCallNonEmptyColdWithValue(t *testing.T) {}

func TestGasDimensionLoggerCallNonEmptyWarmNoValue(t *testing.T) {}

func TestGasDimensionLoggerCallNonEmptyWarmWithValue(t *testing.T) {}

func TestGasDimensionLoggerCallCodeEmptyColdNoValue(t *testing.T) {}

func TestGasDimensionLoggerCallCodeEmptyColdWithValue(t *testing.T) {}

func TestGasDimensionLoggerCallCodeEmptyWarmNoValue(t *testing.T) {}

func TestGasDimensionLoggerCallCodeEmptyWarmWithValue(t *testing.T) {}

func TestGasDimensionLoggerCallCodeNonEmptyColdNoValue(t *testing.T) {}

func TestGasDimensionLoggerCallCodeNonEmptyColdWithValue(t *testing.T) {}

func TestGasDimensionLoggerCallCodeNonEmptyWarmNoValue(t *testing.T) {}

func TestGasDimensionLoggerCallCodeNonEmptyWarmWithValue(t *testing.T) {}

// ############################################################
//                           SSTORE
// ############################################################
//
// SSTORE has many permutations
// warm or cold
// 0 -> 0
// 0 -> non-zero
// non-zero -> 0
// non-zero -> non-zero (same value)
// non-zero -> non-zero (different value)

func TestGasDimensionLoggerSstoreColdZeroToZero(t *testing.T)                     {}
func TestGasDimensionLoggerSstoreColdZeroToNonZeroValue(t *testing.T)             {}
func TestGasDimensionLoggerSstoreColdNonZeroValueToZero(t *testing.T)             {}
func TestGasDimensionLoggerSstoreColdNonZeroToSameNonZeroValue(t *testing.T)      {}
func TestGasDimensionLoggerSstoreColdNonZeroToDifferentNonZeroValue(t *testing.T) {}

func TestGasDimensionLoggerSstoreWarmZeroToZero(t *testing.T)                     {}
func TestGasDimensionLoggerSstoreWarmZeroToNonZeroValue(t *testing.T)             {}
func TestGasDimensionLoggerSstoreWarmNonZeroValueToZero(t *testing.T)             {}
func TestGasDimensionLoggerSstoreWarmNonZeroToSameNonZeroValue(t *testing.T)      {}
func TestGasDimensionLoggerSstoreWarmNonZeroToDifferentNonZeroValue(t *testing.T) {}

// ############################################################
//                          SELFDESTRUCT
// ############################################################
//
// SELFDESTRUCT has many permutations
// warm or cold
// code at target address
// value transferred or no value transferred

func TestGasDimensionLoggerSelfdestructColdNoValueEmpty(t *testing.T)      {}
func TestGasDimensionLoggerSelfdestructColdNoValueNonEmpty(t *testing.T)   {}
func TestGasDimensionLoggerSelfdestructColdWithValueEmpty(t *testing.T)    {}
func TestGasDimensionLoggerSelfdestructColdWithValueNonEmpty(t *testing.T) {}

func TestGasDimensionLoggerSelfdestructWarmNoValueEmpty(t *testing.T)      {}
func TestGasDimensionLoggerSelfdestructWarmNoValueNonEmpty(t *testing.T)   {}
func TestGasDimensionLoggerSelfdestructWarmWithValueEmpty(t *testing.T)    {}
func TestGasDimensionLoggerSelfdestructWarmWithValueNonEmpty(t *testing.T) {}

// ############################################################
//                         HELPER FUNCTIONS
// ############################################################

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

// checks that all of the fields of the expected and actual dimension logs are equal
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

// checks that the one dimensional gas cost is equal to the sum of the other gas dimensions
func checkGasDimensionsEqualOneDimensionalGas(
	t *testing.T,
	l *DimensionLogRes,
) {
	t.Helper()
	if l.OneDimensionalGasCost != l.Computation+l.StateAccess+l.StateGrowth+l.HistoryGrowth {
		Fatal(t, "Expected OneDimensionalGasCost to equal sum of gas dimensions", l)
	}
}
