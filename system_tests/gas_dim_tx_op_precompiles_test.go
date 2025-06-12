package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	pgen "github.com/offchainlabs/nitro/solgen/go/precompilesgen"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

// this test calls the ArbBlockNumber function on the ArbSys precompile
func TestDimTxOpArbSysBlockNumberForSload(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	builder, auth, cleanup := setupProgramTest(t, false, gasDimPrecompileBuilderOpts()...)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	filePath := rustFile("keccak")
	wasm, _ := readWasmFile(t, filePath)
	auth.GasLimit = 32000000 // skip gas estimation
	program := deployContract(t, ctx, auth, l2client, wasm)

	_, err := pgen.NewArbWasm(types.ArbWasmAddress, l2client)
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
	t.Parallel()
	builder, auth, cleanup := setupProgramTest(t, false, gasDimPrecompileBuilderOpts()...)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	filePath := rustFile("keccak")
	wasm, _ := readWasmFile(t, filePath)
	auth.GasLimit = 32000000 // skip gas estimation
	program := deployContract(t, ctx, auth, l2client, wasm)

	arbWasm, err := pgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, program)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// This test runs the tracer on a transaction that includes a call to a
// stylus contract.
func TestDimTxOpStylus(t *testing.T) {
	t.Parallel()
	builder, auth, cleanup := setupProgramTest(t, true, gasDimPrecompileBuilderOpts()...)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, rustFile("keccak"))

	preimage := []byte("hello world, ok")
	keccakArgs := []byte{0x01} // keccak the preimage once
	keccakArgs = append(keccakArgs, preimage...)

	tx := builder.L2Info.PrepareTxTo("Owner", &program, 1000000, nil, keccakArgs)
	Require(t, l2client.SendTransaction(ctx, tx))
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
		builder.execConfig.Caching.StateScheme = rawdb.HashScheme
		builder.execConfig.Sequencer.MaxRevertGasReject = 0
		builder.WithArbOSVersion(params.MaxArbosVersionSupported)
	}
	return []func(*NodeBuilder){builderOpts}
}
