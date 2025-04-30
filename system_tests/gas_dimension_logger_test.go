package arbtest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
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
	LogStaticCost                  = params.LogGas
	LogDataGas                     = params.LogDataGas
	LogTopicGasHistoryGrowth       = 256
	LogTopicGasComputation         = params.LogTopicGas - LogTopicGasHistoryGrowth
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCounter)
	receipt := callOnContract(t, builder, auth, contract.NoSpecials)
	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)

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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployBalance)
	receipt := callOnContract(t, builder, auth, contract.CallBalanceCold)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	balanceLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "BALANCE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		balanceLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, balanceLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployBalance)
	receipt := callOnContract(t, builder, auth, contract.CallBalanceWarm)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	balanceLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "BALANCE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		balanceLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, balanceLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeSize)
	receipt := callOnContract(t, builder, auth, contract.GetExtCodeSizeCold)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeSizeLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODESIZE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeSizeLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeSizeLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeSize)
	receipt := callOnContract(t, builder, auth, contract.GetExtCodeSizeWarm)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeSizeLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODESIZE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeSizeLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeSizeLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeHash)
	receipt := callOnContract(t, builder, auth, contract.GetExtCodeHashCold)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeHashLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODEHASH")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeHashLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeHashLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeHash)
	receipt := callOnContract(t, builder, auth, contract.GetExtCodeHashWarm)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeHashLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODEHASH")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeHashLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeHashLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySload)
	receipt := callOnContract(t, builder, auth, contract.ColdSload)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sloadLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SLOAD")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdSloadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmSloadCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		sloadLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, sloadLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySload)
	receipt := callOnContract(t, builder, auth, contract.WarmSload)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sloadLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SLOAD")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		sloadLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, sloadLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeCopy)
	receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyColdNoMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeCopyLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODECOPY")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost + extCodeCopyMinimumWordCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeCopyLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeCopyLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeCopy)
	receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyColdMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeCopyLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODECOPY")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdAccountAccessCost + extCodeCopyMemoryExpansionCost + extCodeCopyMinimumWordCost,
		Computation:           WarmStorageReadCost + extCodeCopyMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeCopyLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeCopyLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeCopy)
	receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyWarmNoMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeCopyLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODECOPY")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost + extCodeCopyMinimumWordCost,
		Computation:           WarmStorageReadCost,
		StateAccess:           extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeCopyLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeCopyLog)
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

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeCopy)
	receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyWarmMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeCopyLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODECOPY")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: WarmStorageReadCost + extCodeCopyMemoryExpansionCost + extCodeCopyMinimumWordCost,
		Computation:           WarmStorageReadCost + extCodeCopyMemoryExpansionCost,
		StateAccess:           extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		extCodeCopyLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeCopyLog)
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

func TestGasDimensionLoggerDelegateCallEmptyCold(t *testing.T)    { t.Fail() }
func TestGasDimensionLoggerDelegateCallEmptyWarm(t *testing.T)    { t.Fail() }
func TestGasDimensionLoggerDelegateCallNonEmptyCold(t *testing.T) { t.Fail() }
func TestGasDimensionLoggerDelegateCallNonEmptyWarm(t *testing.T) { t.Fail() }
func TestGasDimensionLoggerStaticCallEmptyCold(t *testing.T)      { t.Fail() }
func TestGasDimensionLoggerStaticCallEmptyWarm(t *testing.T)      { t.Fail() }
func TestGasDimensionLoggerStaticCallNonEmptyCold(t *testing.T)   { t.Fail() }
func TestGasDimensionLoggerStaticCallNonEmptyWarm(t *testing.T)   { t.Fail() }

// ############################################################
//	             LOG0, LOG1, LOG2, LOG3, LOG4
// ############################################################
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

// This test deploys a contract that emits an empty LOG0
//
// since it has no data, we expect the gas cost to just be
// the static cost of the LOG0 opcode, and we assign that to
// computation.
// Therefore we expect the one dimensional gas cost to be
// 375, computation to be 375, and all other gas dimensions to be 0
func TestGasDimensionLoggerLog0Empty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicEmptyData)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog0NonEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicNonEmptyData)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog1Empty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitOneTopicEmptyData)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog1NonEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitOneTopicNonEmptyData)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog2(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitTwoTopics)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog2ExtraData(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitTwoTopicsExtraData)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog3(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitThreeTopics)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog3ExtraData(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitThreeTopicsExtraData)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog4(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitFourTopics)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog4ExtraData(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitFourTopicsExtraData)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog0WithMemoryExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitZeroTopicNonEmptyDataAndMemExpansion)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog1WithMemoryExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitOneTopicNonEmptyDataAndMemExpansion)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog2WithMemoryExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitTwoTopicsExtraDataAndMemExpansion)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog3WithMemoryExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitThreeTopicsExtraDataAndMemExpansion)

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
	checkDimensionLogGasCostsEqual(
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
func TestGasDimensionLoggerLog4WithMemoryExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
	receipt := callOnContract(t, builder, auth, contract.EmitFourTopicsExtraDataAndMemExpansion)

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
	checkDimensionLogGasCostsEqual(
		t,
		expected,
		log4Log,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, log4Log)
}

// ############################################################
//	                    CREATE & CREATE2
// ############################################################
//
// CREATE and CREATE2 only have two permutations, whether or not you
// transfer value with the creation

func TestGasDimensionLoggerCreate(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCreateWithValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCreate2(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCreate2WithValue(t *testing.T) { t.Fail() }

// ############################################################
//                      CALL and CALLCODE
// ############################################################
//
// CALL and CALLCODE have many permutations
// warm or cold
// no value or value transfer with the call
// empty or non-empty code at target address

func TestGasDimensionLoggerCallEmptyColdNoValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallEmptyColdWithValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallEmptyWarmNoValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallEmptyWarmWithValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallNonEmptyColdNoValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallNonEmptyColdWithValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallNonEmptyWarmNoValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallNonEmptyWarmWithValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallCodeEmptyColdNoValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallCodeEmptyColdWithValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallCodeEmptyWarmNoValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallCodeEmptyWarmWithValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallCodeNonEmptyColdNoValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallCodeNonEmptyColdWithValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallCodeNonEmptyWarmNoValue(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerCallCodeNonEmptyWarmWithValue(t *testing.T) { t.Fail() }

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

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to 0
//
// we expect the gas cost of this operation to be 2200, 100 for the base sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 100, state access to be 2000, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreColdZeroToZero(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to a non-zero value
//
// we expect the gas cost of this operation to be 22100, 20000 for the sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 100, state read/write to be 0, state growth to be 22000,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreColdZeroToNonZeroValue(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to 0
//
// we expect the gas cost of this operation to be 5000 and a gas refund of 4800.
// this is from an sstore cost of 2900, and a cold access set cost of 2100.
// we expect computation to be 100, state read/write to be 0, state growth to be 4900,
// history growth to be 0, and state growth refund to be 4800
func TestGasDimensionLoggerSstoreColdNonZeroValueToZero(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// the same non-zero value
//
// we expect the gas cost of this operation to be 2200, 100 for the base sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 100, state read/write to be 0, state growth to be 2100,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreColdNonZeroToSameNonZeroValue(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// a different non-zero value
//
// we expect the gas cost of this operation to be 5000, 2900 for the sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 100, state read/write to be 0, state growth to be 4900,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreColdNonZeroToDifferentNonZeroValue(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to 0
//
// we expect the gas cost of this operation to be 100, 100 for the base sstore cost,
// we expect computation to be 0, state access to be 100, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreWarmZeroToZero(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to a non-zero value
//
// we expect the gas cost of this operation to be 20000, 20000 for the sstore cost,
// we expect computation to be 0, state access to be 0, state growth to be 20000,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreWarmZeroToNonZeroValue(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to 0
//
// we expect the gas cost of this operation to be 2900, with a gas refund of 4800
// This is 2900 just for the sstore cost
// we expect computation to be 0, state read/write to be 0, state growth to be 2900,
// history growth to be 0, and state growth refund to be 4800
func TestGasDimensionLoggerSstoreWarmNonZeroValueToZero(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// the same non-zero value
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreWarmNonZeroToSameNonZeroValue(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// a different non-zero value
//
// we expect the gas cost of this operation to be 2900 for the base sstore cost,
// we expect computation to be 0, state read/write to be 0, state growth to be 2900,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreWarmNonZeroToDifferentNonZeroValue(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from non zero to non zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to a different non zero value
// for example, changing some value from 2->3->4
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreMultipleWarmNonZeroToNonZeroToNonZero(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from non zero to non zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to the same non zero value
// for example, changing some value from 2->3->2
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// and to get a gas refund of 2800
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be 2800
func TestGasDimensionLoggerSstoreMultipleWarmNonZeroToNonZeroToSameNonZero(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from non zero to zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to a non zero value
// for example, changing some value from 2->0->3
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// and to have a NEGATIVE gas refund of -4800, i.e. taking away from the previous gas refund
// that was granted for changing the sstore from non zero to zero
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be -4800
func TestGasDimensionLoggerSstoreMultipleWarmNonZeroToZeroToNonZero(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from non zero to zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to the same non zero value
// for example, changing some value from 2->0->2
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// and to have a NEGATIVE gas refund of -2000, i.e. taking away from the previous gas refund
// that was granted for changing the sstore from non zero to zero
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be -2000
func TestGasDimensionLoggerSstoreMultipleWarmNonZeroToZeroToSameNonZero(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from zero to non zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, to a different non zero value
// for example, changing some value from 0->2->3
//
// We expect the gas cost of this operation to be 100 for the base sstore cost,
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreMultipleWarmZeroToNonZeroToNonZero(t *testing.T) { t.Fail() }

// This test deployes a contract with a local variable that we can SSTORE to
// we test multiple SSTOREs and the interaction for the second SSTORE to an already changed value
// in this test we test a change where the first SSTORE changed a value from zero to non zero
// in the past, and now we're evaluating the cost for this second sstore which is changing the value
// again, back to zero
// for example, changing some value from 0->3->0
//
// We expect the gas cost of this operation to be 100 for the base sstore cost,
// we expect to get a gas refund of 19900
// we expect computation to be 0, state read/write to be 0, state growth to be 100,
// history growth to be 0, and state growth refund to be 19900
func TestGasDimensionLoggerSstoreMultipleWarmZeroToNonZeroBackToZero(t *testing.T) { t.Fail() }

// ############################################################
//                          SELFDESTRUCT
// ############################################################
//
// SELFDESTRUCT has many permutations
// warm or cold
// code at target address
// value transferred or no value transferred

func TestGasDimensionLoggerSelfdestructColdNoValueEmpty(t *testing.T)      { t.Fail() }
func TestGasDimensionLoggerSelfdestructColdNoValueNonEmpty(t *testing.T)   { t.Fail() }
func TestGasDimensionLoggerSelfdestructColdWithValueEmpty(t *testing.T)    { t.Fail() }
func TestGasDimensionLoggerSelfdestructColdWithValueNonEmpty(t *testing.T) { t.Fail() }

func TestGasDimensionLoggerSelfdestructWarmNoValueEmpty(t *testing.T)      { t.Fail() }
func TestGasDimensionLoggerSelfdestructWarmNoValueNonEmpty(t *testing.T)   { t.Fail() }
func TestGasDimensionLoggerSelfdestructWarmWithValueEmpty(t *testing.T)    { t.Fail() }
func TestGasDimensionLoggerSelfdestructWarmWithValueNonEmpty(t *testing.T) { t.Fail() }

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

// deploy the contract we want to deploy for this test
// wait for it to be included
func deployGasDimensionTestContract[C any](
	t *testing.T,
	builder *NodeBuilder,
	auth bind.TransactOpts,
	deployFunc func(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, C, error),
) (
	contract C,
) {
	t.Helper()
	_, tx, contract, err := deployFunc(
		&auth,             // Transaction options
		builder.L2.Client, // Ethereum client
	)
	Require(t, err)

	// 3. Wait for deployment to succeed
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	return contract
}

// call whatever test function is required for the test on the contract
func callOnContract[F func(auth *bind.TransactOpts) (*types.Transaction, error)](
	t *testing.T,
	builder *NodeBuilder,
	auth bind.TransactOpts,
	testFunc F,
) (receipt *types.Receipt) {
	t.Helper()
	tx, err := testFunc(&auth) // For write operations
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

// highlight one specific dimension log you want to get out of the
// dimension logs and return it. Make some basic field validation checks on the
// log while you iterate through it.
func getSpecificDimensionLog(t *testing.T, dimensionLogs []DimensionLogRes, expectedOpcode string) (
	specificDimensionLog *DimensionLogRes,
) {
	t.Helper()
	var expectedOpcodeCount uint64 = 0

	// there should only be one BALANCE in the entire trace
	// go through and grab it and its data
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
			expectedOpcodeCount++
			specificDimensionLog = &log
		}
	}
	if expectedOpcodeCount != 1 {
		Fatal(t, "Expected 1 ", expectedOpcode, " got ", expectedOpcodeCount)
	}
	if specificDimensionLog == nil {
		Fatal(t, "Expected ", expectedOpcode, " log, got nil")
	}
	return specificDimensionLog
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
}

// checks that the one dimensional gas cost is equal to the sum of the other gas dimensions
func checkGasDimensionsEqualOneDimensionalGas(
	t *testing.T,
	l *DimensionLogRes,
) {
	t.Helper()
	if l.OneDimensionalGasCost != l.Computation+l.StateAccess+l.StateGrowth+l.HistoryGrowth {
		Fatal(t, "Expected OneDimensionalGasCost to equal sum of gas dimensions", l.DebugString())
	}
}
