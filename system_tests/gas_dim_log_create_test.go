package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

// #########################################################################################################
// #########################################################################################################
//	                                             CREATE & CREATE2
// #########################################################################################################
// #########################################################################################################
// CREATE and CREATE2  have permutations:
// Paying vs NoTransfer (is ether value being sent with this call?)
// MemExpansion vs MemUnchanged (does the creation write to new additional memory?)

// #########################################################################################################
//                                              CREATE
// #########################################################################################################

// in this test, we do a CREATE of a new contract with no transfer of value
// and the creation does not write to new additional memory
// Unfortunately it's really hard to show create without using magic numbers
// from staring at debug traces, so you'll just have to trust the magic values below
// we found that the code execution cost for this particular contract is 22477
// that the deployed contract code is 181 bytes
// and that the init code is 359 bytes
//
// So we expect the one dimensional gas to be
// 32000 for the static cost
// + ((359+31)/32) * 2 for the init code cost,
// + 0 for the memory expansion
// + 22477 for the deployed code execution cost
// + 200 * 181 for the code deposit cost
// for compute we expect:
// ((359+31)/32) * 2 for the init code cost,
// + 0 for the memory expansion
// + (32000 - 25000) / 2 for static cost to assign to compute
// the state access cost to be 0
// the state growth cost to be:
// 25000 for the cost of a "new account" (this is taken from the CALL opcode's amount on this)
// + 200 * 181 for the cost of the deployed code size
// + (32000 - 25000) / 2 for static cost to assign to state growth
// the history growth to be 0
// the state growth refund to be 0
func TestDimLogCreateNoTransferMemUnchanged(t *testing.T) {
	t.Parallel()
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreator)

	_, receipt := callOnContract(t, builder, auth, creator.CreateNoTransferMemUnchanged)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	createLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CREATE")

	var initCodeSize uint64 = 359
	var deployedCodeSize uint64 = 181
	var expectedCallExecutionCost uint64 = 22477
	var expectedMemoryExpansionCost uint64 = 0

	var minimumWordSize uint64 = (initCodeSize + 31) / 32

	var expectedInitCodeCost uint64 = 2 * minimumWordSize
	var expectedCodeDepositCost uint64 = 200 * deployedCodeSize

	var expectedStaticCostAssignedToCompute uint64 = (params.CreateGas - params.CallNewAccountGas) / 2
	var expectedStaticCostAssignedToStateGrowth uint64 = params.CreateGas - params.CallNewAccountGas - expectedStaticCostAssignedToCompute

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.CreateGas + expectedInitCodeCost + expectedMemoryExpansionCost + expectedCodeDepositCost,
		Computation:           expectedInitCodeCost + expectedMemoryExpansionCost + expectedStaticCostAssignedToCompute,
		StateAccess:           0,
		StateGrowth:           expectedStaticCostAssignedToStateGrowth + params.CallNewAccountGas + expectedCodeDepositCost,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    expectedCallExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, createLog)
	checkGasDimensionsEqualOneDimensionalGas(t, createLog)
}

// in this test, we do a CREATE of a new contract with no transfer of value
// and the creation writes to new additional memory, causing memory expansion
func TestDimLogCreateNoTransferMemExpansion(t *testing.T) {
	t.Parallel()
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreator)

	_, receipt := callOnContract(t, builder, auth, creator.CreateNoTransferMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	createLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CREATE")

	var initCodeSize uint64 = 416
	var deployedCodeSize uint64 = 181
	var expectedCallExecutionCost uint64 = 2586
	var expectedMemoryExpansionCost uint64 = 6

	var minimumWordSize uint64 = (initCodeSize + 31) / 32

	var expectedInitCodeCost uint64 = 2 * minimumWordSize
	var expectedCodeDepositCost uint64 = 200 * deployedCodeSize

	var expectedStaticCostAssignedToCompute uint64 = (params.CreateGas - params.CallNewAccountGas) / 2
	var expectedStaticCostAssignedToStateGrowth uint64 = params.CreateGas - params.CallNewAccountGas - expectedStaticCostAssignedToCompute

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.CreateGas + expectedInitCodeCost + expectedMemoryExpansionCost + expectedCodeDepositCost,
		Computation:           expectedInitCodeCost + expectedMemoryExpansionCost + expectedStaticCostAssignedToCompute,
		StateAccess:           0,
		StateGrowth:           expectedStaticCostAssignedToStateGrowth + params.CallNewAccountGas + expectedCodeDepositCost,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    expectedCallExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, createLog)
	checkGasDimensionsEqualOneDimensionalGas(t, createLog)
}

// in this test, we do a CREATE to a new contract with a transfer of ether
// and the creation does not write to new additional memory
//
// The gas costs are identical to the case with NoTransfer, see
// the comments for TestDimLogCreateNoTransferMemUnchanged above
func TestDimLogCreatePayingMemUnchanged(t *testing.T) {
	t.Parallel()
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	creatorAddress, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreator)
	// transfer some eth to the creator contract
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", creatorAddress, big.NewInt(1e17), builder.L2Info)

	_, receipt := callOnContract(t, builder, auth, creator.CreatePayableMemUnchanged)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	createLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CREATE")

	var initCodeSize uint64 = 359
	var deployedCodeSize uint64 = 181
	var expectedCallExecutionCost uint64 = 22477
	var expectedMemoryExpansionCost uint64 = 0

	var minimumWordSize uint64 = (initCodeSize + 31) / 32

	var expectedInitCodeCost uint64 = 2 * minimumWordSize
	var expectedCodeDepositCost uint64 = 200 * deployedCodeSize

	var expectedStaticCostAssignedToCompute uint64 = (params.CreateGas - params.CallNewAccountGas) / 2
	var expectedStaticCostAssignedToStateGrowth uint64 = params.CreateGas - params.CallNewAccountGas - expectedStaticCostAssignedToCompute

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.CreateGas + expectedInitCodeCost + expectedMemoryExpansionCost + expectedCodeDepositCost,
		Computation:           expectedInitCodeCost + expectedMemoryExpansionCost + expectedStaticCostAssignedToCompute,
		StateAccess:           0,
		StateGrowth:           expectedStaticCostAssignedToStateGrowth + params.CallNewAccountGas + expectedCodeDepositCost,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    expectedCallExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, createLog)
	checkGasDimensionsEqualOneDimensionalGas(t, createLog)
}

// in this test, we do a CREATE of a new contract with transfer of value
// and the creation writes to new additional memory, causing memory expansion
func TestDimLogCreatePayingMemExpansion(t *testing.T) {
	t.Parallel()
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	creatorAddress, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreator)
	// transfer some eth to the creator contract
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", creatorAddress, big.NewInt(1e17), builder.L2Info)

	_, receipt := callOnContract(t, builder, auth, creator.CreatePayableMemExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	createLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CREATE")

	var initCodeSize uint64 = 416
	var deployedCodeSize uint64 = 181
	var expectedCallExecutionCost uint64 = 2586
	var expectedMemoryExpansionCost uint64 = 6

	var minimumWordSize uint64 = (initCodeSize + 31) / 32

	var expectedInitCodeCost uint64 = 2 * minimumWordSize
	var expectedCodeDepositCost uint64 = 200 * deployedCodeSize

	var expectedStaticCostAssignedToCompute uint64 = (params.CreateGas - params.CallNewAccountGas) / 2
	var expectedStaticCostAssignedToStateGrowth uint64 = params.CreateGas - params.CallNewAccountGas - expectedStaticCostAssignedToCompute

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.CreateGas + expectedInitCodeCost + expectedMemoryExpansionCost + expectedCodeDepositCost,
		Computation:           expectedInitCodeCost + expectedMemoryExpansionCost + expectedStaticCostAssignedToCompute,
		StateAccess:           0,
		StateGrowth:           expectedStaticCostAssignedToStateGrowth + params.CallNewAccountGas + expectedCodeDepositCost,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    expectedCallExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, createLog)
	checkGasDimensionsEqualOneDimensionalGas(t, createLog)
}

// #########################################################################################################
//                                              CREATE2
// #########################################################################################################

// in this test, we do a CREATE2 of a new contract with no transfer of value
// and the creation does not write to new additional memory
// Unfortunately it's really hard to show create without using magic numbers
// from staring at debug traces, so you'll just have to trust the magic values below
// we found that the code execution cost for this particular contract is 22477
// that the deployed contract code is 181 bytes
// and that the init code is 359 bytes
//
// So we expect the one dimensional gas to be
// 32000 for the static cost
// + ((359+31)/32) * 2 for the init code cost,
// + 0 for the memory expansion
// + 22477 for the deployed code execution cost
// + 200 * 181 for the code deposit cost
// + 6 * ((359+31)/32) for the hash cost
// for compute we expect:
// ((359+31)/32) * 2 for the init code cost,
// + 0 for the memory expansion
// + (32000 - 25000) / 2 for static cost to assign to compute
// + 6 * ((359+31)/32) for the hash cost
// the state access cost to be 0
// the state growth cost to be:
// 25000 for the cost of a "new account" (this is taken from the CALL opcode's amount on this)
// + 200 * 181 for the cost of the deployed code size
// + (32000 - 25000) / 2 for static cost to assign to state growth
// the history growth to be 0
// the state growth refund to be 0
func TestDimLogCreate2NoTransferMemUnchanged(t *testing.T) {
	t.Parallel()
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreatorTwo)

	receipt := callOnContractWithOneArg(t, builder, auth, creator.CreateTwoNoTransferMemUnchanged, [32]byte{0x13, 0x37})

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	createLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CREATE2")

	var initCodeSize uint64 = 359
	var deployedCodeSize uint64 = 181
	var expectedCallExecutionCost uint64 = 22477
	var expectedMemoryExpansionCost uint64 = 0

	var minimumWordSize uint64 = (initCodeSize + 31) / 32

	var expectedInitCodeCost uint64 = 2 * minimumWordSize
	var expectedCodeDepositCost uint64 = 200 * deployedCodeSize
	var expectedHashCost uint64 = 6 * minimumWordSize

	var expectedStaticCostAssignedToCompute uint64 = (params.CreateGas - params.CallNewAccountGas) / 2
	var expectedStaticCostAssignedToStateGrowth uint64 = params.CreateGas - params.CallNewAccountGas - expectedStaticCostAssignedToCompute

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.CreateGas + expectedInitCodeCost + expectedMemoryExpansionCost + expectedCodeDepositCost + expectedHashCost,
		Computation:           expectedInitCodeCost + expectedMemoryExpansionCost + expectedStaticCostAssignedToCompute + expectedHashCost,
		StateAccess:           0,
		StateGrowth:           expectedStaticCostAssignedToStateGrowth + params.CallNewAccountGas + expectedCodeDepositCost,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    expectedCallExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, createLog)
	checkGasDimensionsEqualOneDimensionalGas(t, createLog)
}

// in this test, we do a CREATE2 of a new contract with no transfer of value
// and the creation writes to new additional memory, causing memory expansion
func TestDimLogCreate2NoTransferMemExpansion(t *testing.T) {
	t.Parallel()
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreatorTwo)

	receipt := callOnContractWithOneArg(t, builder, auth, creator.CreateTwoNoTransferMemExpansion, [32]byte{0x13, 0x37})

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	createLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CREATE2")

	var initCodeSize uint64 = 416
	var deployedCodeSize uint64 = 181
	var expectedCallExecutionCost uint64 = 2586
	var expectedMemoryExpansionCost uint64 = 6

	var minimumWordSize uint64 = (initCodeSize + 31) / 32

	var expectedInitCodeCost uint64 = 2 * minimumWordSize
	var expectedCodeDepositCost uint64 = 200 * deployedCodeSize
	var expectedHashCost uint64 = 6 * minimumWordSize

	var expectedStaticCostAssignedToCompute uint64 = (params.CreateGas - params.CallNewAccountGas) / 2
	var expectedStaticCostAssignedToStateGrowth uint64 = params.CreateGas - params.CallNewAccountGas - expectedStaticCostAssignedToCompute

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.CreateGas + expectedInitCodeCost + expectedMemoryExpansionCost + expectedCodeDepositCost + expectedHashCost,
		Computation:           expectedInitCodeCost + expectedMemoryExpansionCost + expectedStaticCostAssignedToCompute + expectedHashCost,
		StateAccess:           0,
		StateGrowth:           expectedStaticCostAssignedToStateGrowth + params.CallNewAccountGas + expectedCodeDepositCost,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    expectedCallExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, createLog)
	checkGasDimensionsEqualOneDimensionalGas(t, createLog)
}

// in this test, we do a CREATE2 of a new contract with transfer of value
// and the creation does not write to new additional memory
//
// The gas costs are identical to the case with NoTransfer, see
// the comments for TestDimLogCreateNoTransferMemUnchanged above
func TestDimLogCreate2PayingMemUnchanged(t *testing.T) {
	t.Parallel()
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreatorTwo)

	receipt := callOnContractWithOneArg(t, builder, auth, creator.CreateTwoNoTransferMemUnchanged, [32]byte{0x13, 0x37})

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	createLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CREATE2")

	var initCodeSize uint64 = 359
	var deployedCodeSize uint64 = 181
	var expectedCallExecutionCost uint64 = 22477
	var expectedMemoryExpansionCost uint64 = 0

	var minimumWordSize uint64 = (initCodeSize + 31) / 32

	var expectedInitCodeCost uint64 = 2 * minimumWordSize
	var expectedCodeDepositCost uint64 = 200 * deployedCodeSize
	var expectedHashCost uint64 = 6 * minimumWordSize

	var expectedStaticCostAssignedToCompute uint64 = (params.CreateGas - params.CallNewAccountGas) / 2
	var expectedStaticCostAssignedToStateGrowth uint64 = params.CreateGas - params.CallNewAccountGas - expectedStaticCostAssignedToCompute

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.CreateGas + expectedInitCodeCost + expectedMemoryExpansionCost + expectedCodeDepositCost + expectedHashCost,
		Computation:           expectedInitCodeCost + expectedMemoryExpansionCost + expectedStaticCostAssignedToCompute + expectedHashCost,
		StateAccess:           0,
		StateGrowth:           expectedStaticCostAssignedToStateGrowth + params.CallNewAccountGas + expectedCodeDepositCost,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    expectedCallExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, createLog)
	checkGasDimensionsEqualOneDimensionalGas(t, createLog)
}

// in this test, we do a CREATE2 of a new contract with transfer of value
// and the creation writes to new additional memory, causing memory expansion
func TestDimLogCreate2PayingMemExpansion(t *testing.T) {
	t.Parallel()
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	creatorAddress, creator := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployCreatorTwo)
	// transfer some eth to the creator contract
	_, _ = builder.L2.TransferBalanceTo(t, "Owner", creatorAddress, big.NewInt(1e17), builder.L2Info)

	receipt := callOnContractWithOneArg(t, builder, auth, creator.CreateTwoPayableMemExpansion, [32]byte{0x13, 0x37})

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	createLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "CREATE2")

	var initCodeSize uint64 = 416
	var deployedCodeSize uint64 = 181
	var expectedCallExecutionCost uint64 = 2586
	var expectedMemoryExpansionCost uint64 = 6

	var minimumWordSize uint64 = (initCodeSize + 31) / 32

	var expectedInitCodeCost uint64 = 2 * minimumWordSize
	var expectedCodeDepositCost uint64 = 200 * deployedCodeSize
	var expectedHashCost uint64 = 6 * minimumWordSize

	var expectedStaticCostAssignedToCompute uint64 = (params.CreateGas - params.CallNewAccountGas) / 2
	var expectedStaticCostAssignedToStateGrowth uint64 = params.CreateGas - params.CallNewAccountGas - expectedStaticCostAssignedToCompute

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.CreateGas + expectedInitCodeCost + expectedMemoryExpansionCost + expectedCodeDepositCost + expectedHashCost,
		Computation:           expectedInitCodeCost + expectedMemoryExpansionCost + expectedStaticCostAssignedToCompute + expectedHashCost,
		StateAccess:           0,
		StateGrowth:           expectedStaticCostAssignedToStateGrowth + params.CallNewAccountGas + expectedCodeDepositCost,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    expectedCallExecutionCost,
	}
	checkGasDimensionsMatch(t, expected, createLog)
	checkGasDimensionsEqualOneDimensionalGas(t, createLog)
}
