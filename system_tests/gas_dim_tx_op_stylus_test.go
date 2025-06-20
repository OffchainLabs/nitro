package arbtest

import (
	"testing"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

// This test runs the tracer on a transaction that includes a call to a
// stylus contract.
func TestDimTxOpStylus(t *testing.T) {
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

// This test calls the Keccak wasm program indirectly via
// a proxy solidity contract in the EVM, testing the
// flow from non-stylus to stylus
func TestDimTxOpStylusKeccakForSloadFromProxy(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, gasDimPrecompileBuilderOpts()...)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, rustFile("keccak"))
	preimage := []byte("hello stylus from evm")
	_, stylusCaller := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployStylusCaller)

	tx, err := stylusCaller.CallKeccak(&auth, program, preimage)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
