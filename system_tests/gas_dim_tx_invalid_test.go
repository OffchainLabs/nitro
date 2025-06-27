// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
	"github.com/offchainlabs/nitro/solgen/go/yulgen"
)

// test the invalid opcode
// it should cause the tx to have a failure
// but the tracer should not fail
// and the gas should still make sense
func TestDimTxOpInvalid(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployInvalid)

	// Create transact opts with NoSend=true and explicit gas limit
	opts := &bind.TransactOpts{
		From:     auth.From,
		Signer:   auth.Signer,
		NoSend:   true, // This will prevent the transaction from being sent
		Context:  ctx,
		GasLimit: 1000000, // Set an explicit gas limit to bypass estimation
	}

	// Get the signed transaction without sending it
	tx, err := contract.Invalid(opts)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	// Now manually send the transaction
	err = builder.L2.Client.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatalf("Failed to send transaction: %v", err)
	}

	// Wait for the transaction to be mined and check receipt
	receipt := EnsureTxFailed(t, ctx, builder.L2.Client, tx)
	CheckEqual(t, receipt.Status, types.ReceiptStatusFailed)

	// Check the trace result
	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// this test tests a transaction that has a revert
// but the revert does not stop the entire transaction
// execution, it's inside a try/catch
func TestDimTxOpRevertInTryCatch(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployInvalid)
	_, receipt := callOnContract(t, builder, auth, contract.RevertInTryCatch)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// this test tests a transaction that has a revert
// but the revert does not stop the entire transaction
// execution, it's inside a try/catch
// and the revert has a memory expansion
func TestDimTxOpRevertInTryCatchWithMemoryExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployInvalid)
	_, receipt := callOnContract(t, builder, auth, contract.RevertInTryCatchWithMemoryExpansion)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// this test should force the revert opcode to be used
// the tx should fail
// but the tracer should not fail
// and the gas should still make sense
func TestDimTxOpRevert(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, true)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployInvalid)

	// Create transact opts with NoSend=true and explicit gas limit
	opts := &bind.TransactOpts{
		From:     auth.From,
		Signer:   auth.Signer,
		NoSend:   true, // This will prevent the transaction from being sent
		Context:  ctx,
		GasLimit: 1000000, // Set an explicit gas limit to bypass estimation
	}
	tx, err := contract.RevertNoMessage(opts)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	err = builder.L2.Client.SendTransaction(ctx, tx)
	// We expect a revert, but the tx should still be mined without error
	Require(t, err)

	receipt := EnsureTxFailed(t, ctx, builder.L2.Client, tx)

	// Check the receipt status
	CheckEqual(t, receipt.Status, types.ReceiptStatusFailed)

	// Check the trace result
	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// this test should force the revert opcode to be used
// the tx should fail
// but the tracer should not fail
// and the gas should still make sense
func TestDimTxOpRevertWithMessage(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, true)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployInvalid)

	// Create transact opts with NoSend=true and explicit gas limit
	opts := &bind.TransactOpts{
		From:     auth.From,
		Signer:   auth.Signer,
		NoSend:   true, // This will prevent the transaction from being sent
		Context:  ctx,
		GasLimit: 1000000, // Set an explicit gas limit to bypass estimation
	}
	tx, err := contract.RevertWithMessage(opts)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	err = builder.L2.Client.SendTransaction(ctx, tx)
	// We expect a revert, but the tx should still be mined without error
	Require(t, err)

	receipt := EnsureTxFailed(t, ctx, builder.L2.Client, tx)

	// Check the receipt status
	CheckEqual(t, receipt.Status, types.ReceiptStatusFailed)

	// Check the trace result
	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

func TestDimTxOpRevertWithMemoryExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, true)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployInvalid)

	// Create transact opts with NoSend=true and explicit gas limit
	opts := &bind.TransactOpts{
		From:     auth.From,
		Signer:   auth.Signer,
		NoSend:   true, // This will prevent the transaction from being sent
		Context:  ctx,
		GasLimit: 1000000, // Set an explicit gas limit to bypass estimation
	}
	tx, err := contract.RevertWithMemoryExpansion(opts)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	err = builder.L2.Client.SendTransaction(ctx, tx)
	// We expect a revert, but the tx should still be mined without error
	Require(t, err)

	receipt := EnsureTxFailed(t, ctx, builder.L2.Client, tx)

	// Check the receipt status
	CheckEqual(t, receipt.Status, types.ReceiptStatusFailed)

	// Check the trace result
	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// this test will cause an invalid jump destination error
// the transaction should fail
// but the tracer should not fail
// and the gas should still make sense
func TestDimTxOpInvalidJump(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	contractAddress, _ := deployGasDimensionTestContract(t, builder, auth, yulgen.DeployInvalidJump)

	tx := builder.L2Info.PrepareTxTo(
		"Owner",
		&contractAddress,
		1e9,
		big.NewInt(0),
		[]byte{0xc5, 0x9e, 0x9b, 0xfd}, // invalidJump()
	)

	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt := EnsureTxFailed(t, ctx, builder.L2.Client, tx)

	CheckEqual(t, receipt.Status, types.ReceiptStatusFailed)
	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
