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

// the invalid opcode uses all of the remaining gas in a transaction
// and halts transaction execution
func TestDimLogInvalid(t *testing.T) {
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

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	invalidLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "INVALID")

	// the invalid log should be the absolute last log in the trace result
	lastLog := traceResult.DimensionLogs[len(traceResult.DimensionLogs)-1]
	if lastLog.Op != "INVALID" {
		t.Fatalf("lastLog is not the invalid opcode")
	}

	receiptGasUsed := receipt.GasUsedForL2()

	if traceResult.IntrinsicGas >= receiptGasUsed {
		t.Fatalf("traceResult.IntrinsicGas %d is greater/equal than receiptGasUsed %d", traceResult.IntrinsicGas, receiptGasUsed)
	}
	receiptGasUsed -= traceResult.IntrinsicGas

	var summedGas uint64 = 0
	for _, log := range traceResult.DimensionLogs {
		if log.Op != "INVALID" {
			summedGas += log.OneDimensionalGasCost
		}
	}
	if summedGas >= receiptGasUsed {
		t.Fatalf("summedGas %d is greater/equal than receiptGasUsed %d", summedGas, receiptGasUsed)
	}

	invalidGasUsed := receiptGasUsed - summedGas

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: invalidGasUsed,
		Computation:           invalidGasUsed,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(
		t,
		expected,
		invalidLog,
	)
	checkGasDimensionsEqualOneDimensionalGas(t, invalidLog)
}

// this test tests a transaction that has a revert
// but the revert does not stop the entire transaction
// execution, it's inside a try/catch
func TestDimLogInvalidInTryCatch(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployInvalid)
	_, receipt := callOnContract(t, builder, auth, contract.RevertInTryCatch)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	revertLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "REVERT")

	// revert has memory expansion cost but it has 0 base cost
	expected := ExpectedGasCosts{
		OneDimensionalGasCost: 0,
		Computation:           0,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, revertLog)
	checkGasDimensionsEqualOneDimensionalGas(t, revertLog)
}

// do a revert but do it in a try/catch
// and do it in a way that it has a memory expansion
func TestDimLogInvalidInTryCatchWithMemoryExpansion(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, contract := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployInvalid)
	_, receipt := callOnContract(t, builder, auth, contract.RevertInTryCatchWithMemoryExpansion)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	revertLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "REVERT")

	// the memory expansion cost is 12
	var expectedMemoryExpansionCost uint64 = 12

	// revert has memory expansion cost but it has 0 base cost
	expected := ExpectedGasCosts{
		OneDimensionalGasCost: expectedMemoryExpansionCost,
		Computation:           expectedMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, revertLog)
	checkGasDimensionsEqualOneDimensionalGas(t, revertLog)
}

func TestDimLogRevert(t *testing.T) {
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

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	revertLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "REVERT")

	// revert has memory expansion cost but it has 0 base cost
	expected := ExpectedGasCosts{
		OneDimensionalGasCost: 0,
		Computation:           0,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, revertLog)
	checkGasDimensionsEqualOneDimensionalGas(t, revertLog)
}

func TestDimLogRevertWithMessage(t *testing.T) {
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

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	revertLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "REVERT")

	// revert has memory expansion cost but it has 0 base cost
	expected := ExpectedGasCosts{
		OneDimensionalGasCost: 0,
		Computation:           0,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, revertLog)
	checkGasDimensionsEqualOneDimensionalGas(t, revertLog)
}

func TestDimLogRevertWithMemoryExpansion(t *testing.T) {
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

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	revertLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "REVERT")

	// the memory expansion cost is 12
	var expectedMemoryExpansionCost uint64 = 12

	// revert has memory expansion cost but it has 0 base cost
	expected := ExpectedGasCosts{
		OneDimensionalGasCost: expectedMemoryExpansionCost,
		Computation:           expectedMemoryExpansionCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, revertLog)
	checkGasDimensionsEqualOneDimensionalGas(t, revertLog)
}

// this test will cause an invalid jump destination error
// the transaction should fail
// but the tracer should not fail
// and the gas should still make sense
func TestDimLogInvalidJump(t *testing.T) {
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

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	// the invalid log should be the absolute last log in the trace result
	lastLog := traceResult.DimensionLogs[len(traceResult.DimensionLogs)-1]
	if lastLog.Op != "JUMP" {
		t.Fatalf("lastLog is not the invalid opcode")
	}

	// we expect the gas cost of the invalid jump
	// to be all of the remaining gas in the tx.
	summedGas := uint64(0)
	for _, log := range traceResult.DimensionLogs[:len(traceResult.DimensionLogs)-1] {
		summedGas += log.OneDimensionalGasCost
	}
	expectedGasCost := receipt.GasUsedForL2() - traceResult.IntrinsicGas - summedGas

	// jump always has a base cost of 8
	expected := ExpectedGasCosts{
		OneDimensionalGasCost: expectedGasCost,
		Computation:           expectedGasCost,
		StateAccess:           0,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, &lastLog)
	checkGasDimensionsEqualOneDimensionalGas(t, &lastLog)
}
