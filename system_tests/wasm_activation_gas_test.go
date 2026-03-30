// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// activationGasTest holds the common fixtures for wasm-activation-gas tests.
type activationGasTest struct {
	auth     bind.TransactOpts
	ctx      context.Context
	l2client *ethclient.Client
	arbOwner *precompilesgen.ArbOwner
	arbWasm  *precompilesgen.ArbWasm
	ensure   func(*types.Transaction, error) *types.Receipt
	cleanup  func()
}

// setupActivationGasTest spins up a node at ArbosVersion_StylusActivationGas and wires
// up the ArbOwner / ArbWasm bindings used by the activation-gas test suite.
func setupActivationGasTest(t *testing.T) activationGasTest {
	t.Helper()
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithArbOSVersion(params.ArbosVersion_StylusActivationGas)
	})
	ctx := builder.ctx
	l2client := builder.L2.Client

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	return activationGasTest{auth: auth, ctx: ctx, l2client: l2client, arbOwner: arbOwner, arbWasm: arbWasm, ensure: ensure, cleanup: cleanup}
}

// deployUnactivatedWasm deploys raw wasm bytecode at a new address without activating it.
// An explicit 32M gas limit bypasses estimation, which would fail for unactivated bytecode.
func deployUnactivatedWasm(t *testing.T, env activationGasTest, wasmFile string) common.Address {
	t.Helper()
	wasm, _ := readWasmFile(t, wasmFile)
	env.auth.GasLimit = 32_000_000
	return deployContract(t, env.ctx, env.auth, env.l2client, wasm)
}

// requireActivationGas asserts that ArbWasm.ActivationGas() returns the expected value.
func requireActivationGas(t *testing.T, arbWasm *precompilesgen.ArbWasm, want uint64) {
	t.Helper()
	got, err := arbWasm.ActivationGas(nil)
	Require(t, err)
	if got != want {
		Fatal(t, "expected activation gas", want, "got", got)
	}
}

// requireTxReverts asserts that a submitted transaction is mined but fails.
func requireTxReverts(t *testing.T, ctx context.Context, l2client *ethclient.Client, tx *types.Transaction) {
	t.Helper()
	_, err := EnsureTxSucceeded(ctx, l2client, tx)
	if err == nil {
		Fatal(t, "expected transaction to revert")
	}
}

// TestWasmActivationGasBlocking verifies that:
//   - SetWasmActivationGas / ActivationGas form a correct round-trip, and
//   - setting a blocking activation gas value prevents contract activation while
//     resetting it to zero restores normal activation.
func TestWasmActivationGasBlocking(t *testing.T) {
	env := setupActivationGasTest(t)
	defer env.cleanup()

	program := deployUnactivatedWasm(t, env, rustFile("keccak"))

	requireActivationGas(t, env.arbWasm, 0)

	// set a value no transaction can ever afford
	env.ensure(env.arbOwner.SetWasmActivationGas(&env.auth, math.MaxUint64))
	requireActivationGas(t, env.arbWasm, math.MaxUint64)

	// activation must fail OOG; explicit gas limit skips estimation which would
	// itself fail on a transaction that will definitely OOG
	env.auth.GasLimit = 32_000_000
	env.auth.Value = oneEth
	tx, err := env.arbWasm.ActivateProgram(&env.auth, program)
	Require(t, err)
	requireTxReverts(t, env.ctx, env.l2client, tx)

	// reset activation gas to zero and restore defaults
	env.auth.GasLimit = 0
	env.auth.Value = big.NewInt(0)
	env.ensure(env.arbOwner.SetWasmActivationGas(&env.auth, 0))

	// activation must succeed now
	activateWasm(t, env.ctx, env.auth, env.l2client, program, "keccak")
}

// TestWasmActivationGasVersionGating verifies that SetWasmActivationGas and ActivationGas
// are unavailable on ArbOS versions prior to ArbosVersion_StylusActivationGas.
func TestWasmActivationGasVersionGating(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithArbOSVersion(params.ArbosVersion_51)
	})
	defer cleanup()
	ctx := builder.ctx
	l2client := builder.L2.Client

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	// SetWasmActivationGas must revert; bypass gas estimation with an explicit limit
	// since estimation itself fails on a reverting call.
	auth.GasLimit = 32_000_000
	tx, err := arbOwner.SetWasmActivationGas(&auth, 1_000_000)
	Require(t, err)
	requireTxReverts(t, ctx, l2client, tx)
	auth.GasLimit = 0

	// ActivationGas getter must also revert
	_, err = arbWasm.ActivationGas(nil)
	if err == nil {
		Fatal(t, "expected ActivationGas to fail on pre-v60 chain")
	}
}

// TestWasmActivationGasCharge verifies that a non-zero activation gas is actually
// deducted on top of the normal activation cost when activating a Stylus contract.
func TestWasmActivationGasCharge(t *testing.T) {
	const (
		extraGas            = uint64(1_000_000)
		fixedActivationCost = uint64(1_659_168)
	)

	env := setupActivationGasTest(t)
	defer env.cleanup()

	env.ensure(env.arbOwner.SetWasmActivationGas(&env.auth, extraGas))
	requireActivationGas(t, env.arbWasm, extraGas)

	program := deployUnactivatedWasm(t, env, rustFile("keccak"))

	env.auth.GasLimit = 32_000_000
	env.auth.Value = oneEth
	receipt := env.ensure(env.arbWasm.ActivateProgram(&env.auth, program))
	env.auth.GasLimit = 0
	env.auth.Value = big.NewInt(0)

	// gas used must include both the configurable charge and the fixed activation cost
	if receipt.GasUsedForL2() < extraGas+fixedActivationCost {
		Fatal(t, "expected gas used >=", extraGas+fixedActivationCost, "got", receipt.GasUsedForL2())
	}
}
