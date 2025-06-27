// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

// #########################################################################################################
// #########################################################################################################
//                                               EXTCODECOPY
// #########################################################################################################
// #########################################################################################################

// EXTCODECOPY reads from state and copies code to memory
// for gas dimensions, we don't care about expanding memory, but
// we do care about the cost being correct
//
// EXTCODECOPY has two axes of variation:
// Warm vs Cold (in the access list)
// MemExpansion or MemUnchanged (memory expansion or not)
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
func TestDimLogExtCodeCopyColdMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeCopy)
	_, receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyColdNoMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeCopyLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODECOPY")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + extCodeCopyMinimumWordCost,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           ColdMinusWarmAccountAccessCost + extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
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
func TestDimLogExtCodeCopyColdMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeCopy)
	_, receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyColdMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeCopyLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODECOPY")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.ColdAccountAccessCostEIP2929 + extCodeCopyMemoryExpansionCost + extCodeCopyMinimumWordCost,
		Computation:           params.WarmStorageReadCostEIP2929 + extCodeCopyMemoryExpansionCost,
		StateAccess:           ColdMinusWarmAccountAccessCost + extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
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
func TestDimLogExtCodeCopyWarmMemUnchanged(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeCopy)
	_, receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyWarmNoMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeCopyLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODECOPY")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + extCodeCopyMinimumWordCost,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
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
func TestDimLogExtCodeCopyWarmMemExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployExtCodeCopy)
	_, receipt := callOnContract(t, builder, auth, contract.ExtCodeCopyWarmMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	extCodeCopyLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "EXTCODECOPY")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.WarmStorageReadCostEIP2929 + extCodeCopyMemoryExpansionCost + extCodeCopyMinimumWordCost,
		Computation:           params.WarmStorageReadCostEIP2929 + extCodeCopyMemoryExpansionCost,
		StateAccess:           extCodeCopyMinimumWordCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		extCodeCopyLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, extCodeCopyLog)
}
