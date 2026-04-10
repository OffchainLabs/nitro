// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// race detection makes things slow and miss timeouts
//go:build !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

// TestEagerRecordingBasicHashDB tests that eager recording works correctly with
// HashDB (the traditional scheme). This ensures the eager recording path doesn't
// break existing functionality.
func TestEagerRecordingBasicHashDB(t *testing.T) {
	testEagerRecordingBasic(t, rawdb.HashScheme)
}

// TestEagerRecordingBasicPathDB tests that eager recording enables block validation
// with PathDB, which was previously unsupported.
func TestEagerRecordingBasicPathDB(t *testing.T) {
	testEagerRecordingBasic(t, rawdb.PathScheme)
}

func testEagerRecordingBasic(t *testing.T, stateScheme string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.RequireScheme(t, stateScheme)

	// Enable block validation
	builder.nodeConfig.BlockValidator.Enable = true

	// Explicitly enable eager recording
	builder.execConfig.EagerRecording = gethexec.EagerBlockRecorderConfig{
		Enable:          true,
		RetentionBlocks: 1000,
		CacheSize:       100,
	}

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	// Send some transactions
	perTransfer := big.NewInt(1e12)
	for i := 0; i < 5; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, perTransfer, nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	// Also send a delayed message
	delayedTx := builder.L2Info.PrepareTx("Owner", "User2", 30002, perTransfer, nil)
	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, delayedTx, builder.L1Info, "User", 100000),
	})

	// Create L1 blocks to get the delayed message picked up
	for i := 0; i < 30; i++ {
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err := WaitForTx(ctx, builder.L2.Client, delayedTx.Hash(), time.Second*30)
	Require(t, err)

	// Find the last block with useful transactions
	lastBlock, err := builder.L2.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	for {
		usefulBlock := false
		for _, tx := range lastBlock.Transactions() {
			if tx.Type() != types.ArbitrumInternalTxType {
				usefulBlock = true
				break
			}
		}
		if usefulBlock {
			break
		}
		lastBlock, err = builder.L2.Client.BlockByHash(ctx, lastBlock.ParentHash())
		Require(t, err)
	}

	t.Log("waiting for validation of block:", lastBlock.NumberU64())
	timeout := getDeadlineTimeout(t, time.Minute*10)
	if !builder.L2.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(lastBlock.NumberU64()), timeout) {
		Fatal(t, "did not validate all blocks")
	}

	// Verify the balance is correct
	l2balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	// 5 direct transfers + 1 delayed = 6 transfers
	expectedBalance := new(big.Int).Mul(perTransfer, big.NewInt(6))
	if l2balance.Cmp(expectedBalance) != 0 {
		Fatal(t, "Unexpected balance:", l2balance, "expected:", expectedBalance)
	}
}

// TestEagerRecordingWithContracts tests eager recording with contract deployment
// and interaction to exercise more complex state trie operations.
func TestEagerRecordingWithContracts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// Don't force a scheme - use whatever the test environment specifies

	// Enable block validation
	builder.nodeConfig.BlockValidator.Enable = true

	// Explicitly enable eager recording
	builder.execConfig.EagerRecording = gethexec.EagerBlockRecorderConfig{
		Enable:          true,
		RetentionBlocks: 1000,
		CacheSize:       100,
	}

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	// Deploy a small contract
	contractCode := []byte{byte(vm.PUSH0)}
	contractCode = append(contractCode, byte(vm.PUSH0))
	contractCode = append(contractCode, byte(vm.PUSH1))
	contractCode = append(contractCode, 8) // the prelude length
	contractCode = append(contractCode, byte(vm.PUSH0))
	contractCode = append(contractCode, byte(vm.CODECOPY))
	contractCode = append(contractCode, byte(vm.PUSH0))
	contractCode = append(contractCode, byte(vm.BLOBHASH))
	contractCode = append(contractCode, byte(vm.RETURN))

	tx := builder.L2Info.PrepareTxTo("Owner", nil, builder.L2Info.TransferGas*2, big.NewInt(0), contractCode)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Send a few more eth transfers
	for i := 0; i < 3; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	// Find the last block with useful transactions
	lastBlock, err := builder.L2.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	for {
		usefulBlock := false
		for _, tx := range lastBlock.Transactions() {
			if tx.Type() != types.ArbitrumInternalTxType {
				usefulBlock = true
				break
			}
		}
		if usefulBlock {
			break
		}
		lastBlock, err = builder.L2.Client.BlockByHash(ctx, lastBlock.ParentHash())
		Require(t, err)
	}

	t.Log("waiting for validation of block:", lastBlock.NumberU64())
	timeout := getDeadlineTimeout(t, time.Minute*10)
	if !builder.L2.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(lastBlock.NumberU64()), timeout) {
		Fatal(t, "did not validate all blocks")
	}
}

// TestEagerRecordingAutoEnablePathDB verifies that eager recording is auto-enabled
// when PathDB is the state scheme.
func TestEagerRecordingAutoEnablePathDB(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.RequireScheme(t, rawdb.PathScheme)

	// Enable block validation but do NOT explicitly enable eager recording
	builder.nodeConfig.BlockValidator.Enable = true

	// EagerRecording.Enable is left at default (false) — it should auto-enable for PathDB

	cleanup := builder.Build(t)
	defer cleanup()

	// Verify the eager recorder was created
	if builder.L2.ExecNode.EagerRecorder == nil {
		Fatal(t, "eager recorder should have been auto-enabled for PathDB")
	}

	builder.L2Info.GenerateAccount("User2")

	// Send a transaction and validate it
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	blockNum := receipt.BlockNumber.Uint64()
	t.Log("waiting for validation of block:", blockNum)
	timeout := getDeadlineTimeout(t, time.Minute*10)
	if !builder.L2.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(blockNum), timeout) {
		Fatal(t, "did not validate block", blockNum)
	}
}
