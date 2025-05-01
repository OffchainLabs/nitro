package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployCounter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployBalance)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployBalance)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeSize)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeSize)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeHash)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeHash)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySload)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySload)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeCopy)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeCopy)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeCopy)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployExtCodeCopy)
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
// memory expanded, or no memory expansion
//
// static_gas = 0
// dynamic_gas = memory_expansion_cost + code_execution_cost + address_access_cost
// we do not consider the code_execution_cost as part of the cost of the call itself
// since those costs are counted and incurred by the children of the call.

// this test does the case where the target address being delegatecalled to
// is empty - i.e. no address code at that location. The address being called
// to is also cold, therefore incurring the access list cold read cost.
// the solidity compiler forces no memory expansion for us.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// one-dimensional gas cost to be 2600, for the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerDelegateCallEmptyCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyCold, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the target address being delegatecalled to
// is empty - i.e. no address code at that location.
// the solidity compiler forces no memory expansion for us.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// one-dimensional gas cost to be 100, for the warm access list read,
// computation to be 100 (for the warm access list read),
// state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerDelegateCallEmptyWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyWarm, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// in this test, the target address being delegatecalled to
// is non-empty, there is code at that location that will be executed,
// and the address being called is cold.
// the solidity compiler forces no memory expansion for us.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// call to have a one-dimensional gas cost of 2600 + the child execution gas,
// due to the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerDelegateCallNonEmptyCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyCold, delegateCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	// this is a magic value from the trace of DelegateCallee contract
	var expectedChildGasExecutionCost uint64 = 22712
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// in this test, the target address being delegatecalled to
// is non-empty, there is code at that location that will be executed,
// and the address being called is warm.
// the solidity compiler forces no memory expansion for us.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// call to have a one-dimensional gas cost of 100 + the child execution gas,
// due to the warm access list read cost,
// computation to be 100 (for the warm access list read),
// state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerDelegateCallNonEmptyWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyWarm, delegateCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	// this is a magic value from the trace of DelegateCallee contract
	var expectedChildGasExecutionCost uint64 = 22712
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the target address being delegatecalled to
// is empty - i.e. no address code at that location. The address being called
// to is also cold, therefore incurring the access list cold read cost.
// in this case we force memory expansion for the call in the solidity
// assembly. By staring at the traces and debugging, we find that the
// memory expansion cost is 6.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// one-dimensional gas cost to be 2600, for the access list cold read cost,
// computation to be 100 + 6 (warm access list read + memory expansion),
// state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerDelegateCallEmptyColdMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyColdMemExpansion, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the target address being delegatecalled to
// is empty - i.e. no address code at that location.
// we force memory expansion for the call in the solidity assembly.
// from staring at the traces and debugging,the memory expansion cost is 6.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// one-dimensional gas cost to be 100 + 6, for the warm access list read + memory expansion,
// computation to be 100 + 6 (for the warm access list read + memory expansion),
// state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerDelegateCallEmptyWarmMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallEmptyWarmMemExpansion, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// in this test, the target address being delegatecalled to
// is non-empty, there is code at that location that will be executed,
// and the address being called is cold.
// we force memory expansion for the call in the solidity assembly.
// from staring at the traces and debugging,the memory expansion cost is 6.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// call to have a one-dimensional gas cost of 2600 + 6 + the child execution gas,
// due to the access list cold read cost and memory expansion cost,
// computation to be 100 + 6 (for the warm access list read + memory expansion),
// state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerDelegateCallNonEmptyColdMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyColdMemExpansion, delegateCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	// this is a magic value from the trace of DelegateCallee contract
	var expectedChildGasExecutionCost uint64 = 22712
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// in this test, the target address being delegatecalled to
// is non-empty, there is code at that location that will be executed,
// and the address being called is warm.
// we force memory expansion for the call in the solidity assembly.
// from staring at the traces and debugging,the memory expansion cost is 6.
//
// since it's a call, the call itself does not incur the computation cost
// but rather its children incur various costs. Therefore, we expect the
// call to have a one-dimensional gas cost of 100 + 6 + the child execution gas,
// due to the warm access list read cost and memory expansion cost,
// computation to be 100 + 6 (for the warm access list read + memory expansion),
// state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerDelegateCallNonEmptyWarmMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, delegateCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCaller)
	delegateCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployDelegateCallee)

	receipt := callOnContractWithOneArg(t, builder, auth, delegateCaller.TestDelegateCallNonEmptyWarmMemExpansion, delegateCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	delegateCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "DELEGATECALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	// this is a magic value from the trace of DelegateCallee contract
	var expectedChildGasExecutionCost uint64 = 22712
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, delegateCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, delegateCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the a contract calls another contract via staticcall
// the target address is empty (does not actually have code at that address),
// and the address being called is cold.
// there is no memory expansion in this case.
//
// we expect the one-dimensional gas cost to be 2500, for the cold access list read cost,
// computation to be 100 for the warm access list read cost, state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerStaticCallEmptyCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyCold, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the a contract calls another contract via staticcall
// the target address is empty (does not actually have code at that address),
// and the address being called is warm.
// there is no memory expansion in this case.
//
// we expect the one-dimensional gas cost to be 100, for the warm access list read cost,
// computation to be 100 for the warm access list read cost, state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerStaticCallEmptyWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyWarm, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the a contract calls another contract via staticcall
// the target address is non-empty, so there is code at that location that will be executed,
// and the address being called is cold.
// there is no memory expansion in this case.
//
// we expect the one-dimensional gas cost to be 2500, for the cold access list read cost,
// computation to be 100 for the warm access list read cost, state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerStaticCallNonEmptyCold(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyCold, staticCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 2409
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where the a contract calls another contract via staticcall
// the target address is non-empty, so there is code at that location that will be executed,
// and the address being called is warm.
// there is no memory expansion in this case.
//
// we expect the one-dimensional gas cost to be 100, for the warm access list read cost,
// computation to be 100 for the warm access list read cost, state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerStaticCallNonEmptyWarm(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyWarm, staticCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 2409
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where a contract calls another contract via staticcall
// the target address is empty, so there is no code at that location that will be executed,
// and the address being called is cold.
// memory expansion occurs in this case.
//
// we expect the one-dimensional gas cost to be 2600, for the cold access list read cost,
// plus the memory expansion cost, which via debugging and tracing, we know is 6 gas.
// computation to be 100+6 for the warm access list read cost, state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerStaticCallEmptyColdMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyColdMemExpansion, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where a contract calls another contract via staticcall
// the target address is empty, so there is no code at that location that will be executed,
// and the address being called is warm.
// memory expansion occurs in this case.
//
// we expect the one-dimensional gas cost to be 100, for the warm access list read cost,
// plus the memory expansion cost, which via debugging and tracing, we know is 6 gas.
// computation to be 100+6 for the warm access list read cost, state access to be 0, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerStaticCallEmptyWarmMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")
	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)

	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallEmptyWarmMemExpansion, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	var memoryExpansionCost uint64 = 6

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	var expectedChildGasExecutionCost uint64 = 0
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

// this test does the case where a contract calls another contract via staticcall
// the target address is non-empty, so there is code at that location that will be executed,
// and the address being called is cold.
// memory expansion occurs in this case.
//
// we expect the one-dimensional gas cost to be 2600, for the cold access list read cost,
// plus the memory expansion cost, which via debugging and tracing, we know is 6 gas.
// we also know the child execution gas is 2409 from debugging and tracing.
// computation to be 100+6 for the warm access list read cost, state access to be 2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerStaticCallNonEmptyColdMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyColdMemExpansion, staticCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	var memoryExpansionCost uint64 = 6
	var expectedChildGasExecutionCost uint64 = 2409

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

func TestGasDimensionLoggerStaticCallNonEmptyWarmMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, staticCaller := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCaller)
	staticCalleeAddress, _ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployStaticCallee)
	receipt := callOnContractWithOneArg(t, builder, auth, staticCaller.TestStaticCallNonEmptyWarmMemExpansion, staticCalleeAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	staticCallLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "STATICCALL")

	var memoryExpansionCost uint64 = 6
	var expectedChildGasExecutionCost uint64 = 2409

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		Computation:           params.WarmStorageReadCostEIP2929 + memoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqualCallGas(t, expected, expectedChildGasExecutionCost, staticCallLog)
	checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(t, staticCallLog, expectedChildGasExecutionCost)
}

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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployLogEmitter)
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
//
// Gas dimensionally, SSTORE has one rule basically: if the total gas is
// greater than 20,000 gas, then we know that a write of a 0->non-zero
// occured. That is the only case that actually grows state database size
// all of the other cases are either writing to existing values, or
// writing to a local cache that will eventually be written to the existing
// database values.
// so the majority of cases for SSTORE fall under read/write, except for
// the one case where the database size grows.

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to 0
//
// we expect the gas cost of this operation to be 2200, 100 for the base sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 0, state access to be 2200, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreColdZeroToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreColdZeroToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: ColdMinusWarmSloadCost + params.NetSstoreDirtyGas,
		Computation:           0,
		StateAccess:           ColdMinusWarmSloadCost + params.NetSstoreDirtyGas,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to a non-zero value
//
// we expect the gas cost of this operation to be 22100, 20000 for the sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 0, state read/write to be 2100, state growth to be 20000,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreColdZeroToNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreColdZeroToNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SstoreSetGasEIP2200 + params.ColdSloadCostEIP2929,
		Computation:           0,
		StateAccess:           params.ColdSloadCostEIP2929,
		StateGrowth:           params.SstoreSetGasEIP2200,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to 0
//
// we expect the gas cost of this operation to be 5000 and a gas refund of 4800.
// this is from an sstore cost of 2900, and a cold access set cost of 2100.
// we expect computation to be 100, state read/write to be 0, state growth to be 4900,
// history growth to be 0, and state growth refund to be 4800
func TestGasDimensionLoggerSstoreColdNonZeroValueToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroValueToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SstoreResetGasEIP2200,
		Computation:           0,
		StateAccess:           params.SstoreResetGasEIP2200,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     int64(params.NetSstoreResetRefund),
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// the same non-zero value
//
// we expect the gas cost of this operation to be 2200, 100 for the base sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 0, state read/write to be 2200, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreColdNonZeroToSameNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroToSameNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdSloadCostEIP2929 + params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.ColdSloadCostEIP2929 + params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// a different non-zero value
//
// we expect the gas cost of this operation to be 5000, 2900 for the sstore cost,
// and 2100 for cold access set access.
// we expect computation to be 0, state read/write to be 5000, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreColdNonZeroToDifferentNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreColdNonZeroToDifferentNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SstoreClearGas,
		Computation:           0,
		StateAccess:           params.SstoreClearGas,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to 0
//
// we expect the gas cost of this operation to be 100, 100 for the base sstore cost,
// we expect computation to be 0, state access to be 100, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreWarmZeroToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreWarmZeroToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at 0 and SSTORE it to a non-zero value
//
// we expect the gas cost of this operation to be 20000, 20000 for the sstore cost,
// we expect computation to be 0, state access to be 0, state growth to be 20000,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreWarmZeroToNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreWarmZeroToNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SstoreSetGasEIP2200,
		Computation:           0,
		StateAccess:           0,
		StateGrowth:           params.SstoreSetGasEIP2200,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to 0
//
// we expect the gas cost of this operation to be 2900, with a gas refund of 4800
// This is 2900 just for the sstore cost
// we expect computation to be 0, state read/write to be 2900, state growth to be 0,
// history growth to be 0, and state growth refund to be 4800
func TestGasDimensionLoggerSstoreWarmNonZeroValueToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroValueToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: 2900, // there's no params reference for this, it just is 2900
		Computation:           0,
		StateAccess:           2900,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     int64(params.NetSstoreResetRefund),
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// the same non-zero value
//
// we expect the gas cost of this operation to be 100 for the base sstore cost,
// we expect computation to be 0, state read/write to be 100, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreWarmNonZeroToSameNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroToSameNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// This test deployes a contract with a local variable that we can SSTORE to
// in this test, we SSTORE a variable that starts at a non-zero value and SSTORE it to
// a different non-zero value
//
// we expect the gas cost of this operation to be 2900 for the base sstore cost,
// we expect computation to be 0, state read/write to be 2900, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSstoreWarmNonZeroToDifferentNonZeroValue(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreWarmNonZeroToDifferentNonZeroValue)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: 2900, // there's no params reference for this, it just is 2900
		Computation:           0,
		StateAccess:           2900,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

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
func TestGasDimensionLoggerSstoreMultipleWarmNonZeroToNonZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToNonZeroToNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

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
func TestGasDimensionLoggerSstoreMultipleWarmNonZeroToNonZeroToSameNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToNonZeroToSameNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     2800, // i didn't see anything in params directly for this case
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

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
func TestGasDimensionLoggerSstoreMultipleWarmNonZeroToZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToZeroToNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     -4800,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

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
func TestGasDimensionLoggerSstoreMultipleWarmNonZeroToZeroToSameNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmNonZeroToZeroToSameNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     -2000,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

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
func TestGasDimensionLoggerSstoreMultipleWarmZeroToNonZeroToNonZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmZeroToNonZeroToNonZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

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
func TestGasDimensionLoggerSstoreMultipleWarmZeroToNonZeroBackToZero(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySstore)
	receipt := callOnContract(t, builder, auth, contract.SstoreMultipleWarmZeroToNonZeroBackToZero)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	sstoreLog := getLastOfTwoDimensionLogs(t, traceResult.DimensionLogs, "SSTORE")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929,
		Computation:           0,
		StateAccess:           params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     19900,
	}
	checkDimensionLogGasCostsEqual(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)
}

// ############################################################
//                          SELFDESTRUCT
// ############################################################
//
// SELFDESTRUCT has many permutations
// warm or cold
// code at target address
// value transferred or no value transferred
//
// `value_to_empty_account_cost` is storage growth (25000)
// `address_access_cost` for cold addresses is a read/write cost
// since all this operation does is send ether at this point
// then we assign the static gas of 5000 to read/write,
// since in the non-state-growth case this would be a read/write
// and the access cost is read/write
// in the case where state growth happens due to sending funds to
// a new empty account, then we assign that to state growth.

// in this test case, we self destruct and set the target of funds to be
// an empty address that has no code or value at that address. Normally
// that would trigger state growth, but we also have no money to send
// as part of the selfdestruct, so we don't trigger state growth.
//
// in this case we expect the one-dimensional cost to be 5000 + 2600,
// for the base selfdestruct cost and the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 5000+2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSelfdestructColdNoValueEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// call selfDestructor.warmSelfDestructor(0xdeadbeef)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 + params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that is not empty (i.e. it has code or value).
// but we also self destruct with no funds to send, so it's kind of moot.
//
// in this case we expect the one-dimensional cost to be 5000 + 2600,
// for the base selfdestruct cost and the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 5000+2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSelfdestructColdNoValueNonEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySelfDestructor)
	payableCounterAddress, _ /*payableCounter*/ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployPayableCounter)

	// prefund the selfDestructor and payableCounter with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", payableCounterAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.warmSelfDestructor(payableCounterAddress)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, payableCounterAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 + params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that has no code or value at that address.
// this does trigger state growth.
//
// in this case we expect the one-dimensional cost to be 5000 + 2600 + 25000,
// for the base selfdestruct cost and the access list cold read cost and the
// value to empty account cost,
// so we expect a computation of 100 (for the warm access list read),
// state access to be 5000+2500, state growth to be 25000,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSelfdestructColdWithValueEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.SelfDestruct(emptyAccountAddress) - which is cold
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.CreateBySelfdestructGas + params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 + params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           params.CreateBySelfdestructGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that already has code or value at that address.
// since the address already has code or value, the operation is a state read/write
// rather than a state growth.
//
// in this case we expect the one-dimensional cost to be 5000 + 2600,
// for the base selfdestruct cost and the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 5000+2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSelfdestructColdWithValueNonEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.SelfDestruct(emptyAccountAddress) - which is cold
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.CreateBySelfdestructGas + params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 + params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           params.CreateBySelfdestructGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an empty address that has no code or value at that address. Normally
// that would trigger state growth, but we also have no money to send
// as part of the selfdestruct, so we don't trigger state growth.
//
// in this case we expect the one-dimensional cost to be 5000,
// computation to be 100 (for the warm access list read),
// state access to be 4900, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSelfdestructWarmNoValueEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// call selfDestructor.warmSelfDestructor(0xdeadbeef)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmEmptySelfDestructor, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that has some code at that address and some eth value already,
// but we don't have any funds to send, so we don't send any as part of the
// selfdestruct
//
// for this transaction we expect a one-dimensional cost of 5000
// computation to be 100 (for the warm access list read),
// state access to be 4900, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSelfdestructWarmNoValueNonEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	_ /*selfDestructorAddress*/, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySelfDestructor)
	payableCounterAddress, _ /*payableCounter*/ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployPayableCounter)

	// prefund the payableCounter with some funds, but not the selfDestructor
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", payableCounterAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.warmSelfDestructor(payableCounterAddress)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmSelfDestructor, payableCounterAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case we self destruct and set the target of funds to be
// an address that neither has code nor any eth value yet. Which means
// the resulting selfdestruct will cause state growth, assigning value
// to a new account.
//
// in this case we expect the one-dimensional cost to be 30000,
// 5000 from static cost and 25000 from the value to empty account cost
// 100 for warm access list read
// that gives us a computation of 100, state access of 4900, state growth of 25000,
// history growth of 0, and state growth refund of 0
func TestGasDimensionLoggerSelfdestructWarmWithValueEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.warmSelfDestructor(0xdeadbeef)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmEmptySelfDestructor, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.CreateBySelfdestructGas,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           params.CreateBySelfdestructGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that has some code at that address and some eth value already
//
// for this transaction we expect a one-dimensional cost of 5000
// computation to be 100 (for the warm access list read),
// state access to be 4900, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestGasDimensionLoggerSelfdestructWarmWithValueNonEmpty(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionLoggerSetup(t)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeploySelfDestructor)
	payableCounterAddress, _ /*payableCounter*/ := deployGasDimensionTestContract(t, builder, auth, gasdimensionsgen.DeployPayableCounter)

	// prefund the selfDestructor and payableCounter with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", payableCounterAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.warmSelfDestructor(payableCounterAddress)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmSelfDestructor, payableCounterAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkDimensionLogGasCostsEqual(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

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
) (receipt *types.Receipt) {
	t.Helper()
	tx, err := testFunc(&auth) // For write operations
	Require(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	return receipt
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
}

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
}

// checks that all of the fields of the expected and actual dimension logs are equal
func checkDimensionLogGasCostsEqual(
	t *testing.T,
	expected ExpectedGasCosts,
	actual *DimensionLogRes,
) {
	t.Helper()
	checkGasDimensionsMatch(t, expected, actual)
	if actual.OneDimensionalGasCost != expected.OneDimensionalGasCost {
		Fatal(t, "Expected OneDimensionalGasCost ", expected.OneDimensionalGasCost, " got ", actual.OneDimensionalGasCost, " actual: ", actual.DebugString())
	}
}

// for the special case of opcodes that increase the stack depth,
// the one-dimensional gas cost is the sum of the gas dimension
// and the gas cost of all of the the child opcodes, instead of
// just the gas dimensions
func checkDimensionLogGasCostsEqualCallGas(
	t *testing.T,
	expected ExpectedGasCosts,
	expectedCallChildExecutionGas uint64,
	actual *DimensionLogRes,
) {
	t.Helper()
	checkGasDimensionsMatch(t, expected, actual)
	if actual.OneDimensionalGasCost != expected.OneDimensionalGasCost+expectedCallChildExecutionGas {
		Fatal(t, "Expected OneDimensionalGasCost (", expected.OneDimensionalGasCost, " + ", expectedCallChildExecutionGas, " = ", expected.OneDimensionalGasCost+expectedCallChildExecutionGas, ") got ", actual.OneDimensionalGasCost, " actual: ", actual.DebugString())
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

// checks that the one dimensional gas cost is equal
// to the child execution gas + sum of the other gas dimensions
func checkGasDimensionsEqualOneDimensionalGasWithChildExecutionGas(
	t *testing.T,
	l *DimensionLogRes,
	expectedChildExecutionGas uint64,
) {
	t.Helper()
	if l.OneDimensionalGasCost != l.Computation+l.StateAccess+l.StateGrowth+l.HistoryGrowth+expectedChildExecutionGas {
		Fatal(t, "Expected OneDimensionalGasCost to equal sum of gas dimensions: ", l.DebugString(), " + ", expectedChildExecutionGas)
	}
}
