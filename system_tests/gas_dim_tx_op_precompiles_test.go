package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// this test calls the ArbBlockNumber function on the ArbSys precompile
func TestDimTxOpArbSysBlockNumberForSload(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cleanup()
	defer cancel()

	_, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)

	_, precompileTestContract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployPrecompile)

	_, receipt := callOnContract(t, builder, auth, precompileTestContract.TestArbSysArbBlockNumber)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// this test calls a testActivateProgram function on a test contract
// that in turn calls ActivateProgram on the ArbWasm precompile
// this tests that the logic for counting gas works from a proxy contract
func TestDimTxOpArbWasmActivateProgramForSstoreAndCallFromProxy(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, false, gasDimPrecompileBuilderOpts()...)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	filePath := rustFile("keccak")
	wasm, _ := readWasmFile(t, filePath)
	auth.GasLimit = 32000000 // skip gas estimation
	program := deployContract(t, ctx, auth, l2client, wasm)

	_, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	_, precompileTestContract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployPrecompile)

	auth.Value = oneEth
	tx, err := precompileTestContract.TestActivateProgram(&auth, program)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// this test calls the ActivateProgram function on the ArbWasm precompile
// which calls SSTORE and CALL inside the precompile, for this test
func TestDimTxOpActivateProgramForSstoreAndCall(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, false, gasDimPrecompileBuilderOpts()...)
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
	tx, err := arbWasm.ActivateProgram(&auth, program)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// ******************************************************
//                    HELPER FUNCTIONS
// ******************************************************

func gasDimPrecompileBuilderOpts() []func(*NodeBuilder) {
	builderOpts := func(builder *NodeBuilder) {
		// Match gasDimensionTestSetup settings
		builder.execConfig.Caching.Archive = true
		builder.execConfig.Sequencer.MaxRevertGasReject = 0
		builder.WithArbOSVersion(params.MaxArbosVersionSupported)
	}
	return []func(*NodeBuilder){builderOpts}
}
