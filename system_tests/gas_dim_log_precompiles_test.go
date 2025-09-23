package arbtest

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// This test calls the ArbBlockNumber function on the ArbSys precompile
// which calls SLOAD inside the precompile, for this test
func TestDimLogArbSysBlockNumberForSload(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cleanup()
	defer cancel()

	_, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)

	_, precompileTestContract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployPrecompile)

	_, receipt := callOnContract(t, builder, auth, precompileTestContract.TestArbSysArbBlockNumber)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)

	// We expect an SLOAD with 0 gas, which is the SLOAD fired inside the precompile

	sloadLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SLOAD")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: 0,
		Computation:           0,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, sloadLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sloadLog)
}

// this test calls the ActivateProgram function on the ArbWasm precompile
// which calls SSTORE and CALL inside the precompile, for this test
func TestDimLogActivateProgramForSstoreAndCall(t *testing.T) {
	builderOpts := []func(*NodeBuilder){
		func(builder *NodeBuilder) {
			// Match gasDimensionTestSetup settings
			builder.execConfig.Caching.Archive = true
			builder.execConfig.Sequencer.MaxRevertGasReject = 0
			builder.WithArbOSVersion(params.MaxArbosVersionSupported)
		},
	}
	builder, auth, cleanup := setupProgramTest(t, false, builderOpts...)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	filePath := rustFile("keccak")
	wasm, _ := readWasmFile(t, filePath)
	auth.GasLimit = 32000000 // skip gas estimation
	program := deployContract(t, ctx, auth, l2client, wasm)

	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	auth.Value = oneEth
	fmt.Println("transaction sender", auth.From)
	tx, err := arbWasm.ActivateProgram(&auth, program)
	Require(t, err)
	fmt.Println("tx recipient", tx.To())
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)

	// We expect an SSTORE with 0 gas, which is the SSTORE fired inside the precompile
	sstoreLog := getSpecificDimensionLogAtIndex(t, traceResult.DimensionLogs, "SSTORE", 4, 1)

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: 0,
		Computation:           0,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
		ChildExecutionCost:    0,
	}
	checkGasDimensionsMatch(t, expected, sstoreLog)
	checkGasDimensionsEqualOneDimensionalGas(t, sstoreLog)

	// check the calls inside the precompile are free
	callLog := getSpecificDimensionLogAtIndex(t, traceResult.DimensionLogs, "CALL", 2, 1)
	checkGasDimensionsMatch(t, expected, callLog)
	checkGasDimensionsEqualOneDimensionalGas(t, callLog)
}
