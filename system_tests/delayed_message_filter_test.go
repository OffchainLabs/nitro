// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	arbosutil "github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/cmd/transaction-filterer/api"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// sendDelayedTx sends a transaction via L1 delayed inbox.
// Returns the L2 tx hash that will be used when sequenced.
func sendDelayedTx(t *testing.T, ctx context.Context, builder *NodeBuilder, tx *types.Transaction) common.Hash {
	t.Helper()
	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)

	txbytes, err := tx.MarshalBinary()
	Require(t, err)
	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)

	l1opts := builder.L1Info.GetDefaultTransactOpts("User", ctx)
	l1tx, err := delayedInbox.SendL2Message(&l1opts, txwrapped)
	Require(t, err)
	_, err = builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)

	return tx.Hash()
}

// sendDelayedBatch sends a batch of transactions via L1 delayed inbox as a single delayed message.
func sendDelayedBatch(t *testing.T, ctx context.Context, builder *NodeBuilder, txes types.Transactions) {
	t.Helper()
	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)

	batchData, err := l2MessageBatchDataFromTxes(txes)
	Require(t, err)

	l1opts := builder.L1Info.GetDefaultTransactOpts("User", ctx)
	l1tx, err := delayedInbox.SendL2Message(&l1opts, batchData)
	Require(t, err)
	_, err = builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
}

// advanceAndWaitForDelayed advances L1 blocks and waits for delayed message processing.
func advanceAndWaitForDelayed(t *testing.T, ctx context.Context, builder *NodeBuilder) {
	t.Helper()
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	<-time.After(time.Second * 2)
}

// waitForDelayedSequencerHaltOnHashes waits until the delayed sequencer is halted on exactly the given hashes.
func waitForDelayedSequencerHaltOnHashes(t *testing.T, ctx context.Context, builder *NodeBuilder, expectedHashes []common.Hash, timeout time.Duration) {
	t.Helper()
	expectedSet := make(map[common.Hash]struct{}, len(expectedHashes))
	for _, h := range expectedHashes {
		expectedSet[h] = struct{}{}
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if builder.L2.ConsensusNode.DelayedSequencer == nil {
			t.Fatal("DelayedSequencer is nil")
		}
		hashes, waiting := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx(t)
		if waiting && len(hashes) == len(expectedHashes) {
			match := true
			for _, h := range hashes {
				if _, ok := expectedSet[h]; !ok {
					match = false
					break
				}
			}
			if match {
				return
			}
		}
		<-time.After(100 * time.Millisecond)
	}
	// Get current state for error message
	hashes, waiting := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx(t)
	t.Fatalf("timeout waiting for delayed sequencer to halt on expected hashes: expected=%v, got=%v, waiting=%v", expectedHashes, hashes, waiting)
}

// waitForDelayedSequencerResume waits until the delayed sequencer is no longer halted.
func waitForDelayedSequencerResume(t *testing.T, ctx context.Context, builder *NodeBuilder, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if builder.L2.ConsensusNode.DelayedSequencer == nil {
			t.Fatal("DelayedSequencer is nil")
		}
		_, waiting := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx(t)
		if !waiting {
			return
		}
		<-time.After(100 * time.Millisecond)
	}
	t.Fatal("timeout waiting for delayed sequencer to resume")
}

func createTransactionFiltererService(t *testing.T, ctx context.Context, builder *NodeBuilder, filtererName string) (*node.Node, *api.TransactionFiltererAPI) {
	t.Helper()

	filtererTxOpts := builder.L2Info.GetDefaultTransactOpts(filtererName, ctx)

	// creates transaction-filterer API server
	transactionFiltererStackConf := api.DefaultStackConfig
	// use arbitrary available ports
	transactionFiltererStackConf.HTTPPort = 0
	transactionFiltererStackConf.WSPort = 0
	transactionFiltererStackConf.AuthPort = 0
	transactionFiltererStack, transactionFiltererAPI, err := api.NewStack(&transactionFiltererStackConf, &filtererTxOpts, nil)
	require.NoError(t, err)
	err = transactionFiltererStack.Start()
	require.NoError(t, err)

	builder.execConfig.Sequencer.TransactionFiltering.TransactionFiltererRPCClient.URL = transactionFiltererStack.HTTPEndpoint()

	return transactionFiltererStack, transactionFiltererAPI
}

// addTxHashToOnChainFilter adds a tx hash to the onchain filter via the precompile.
func addTxHashToOnChainFilter(t *testing.T, ctx context.Context, builder *NodeBuilder, txHash common.Hash, filtererName string) {
	t.Helper()

	filtererTxOpts := builder.L2Info.GetDefaultTransactOpts(filtererName, ctx)

	arbFilteredTxs, err := precompilesgen.NewArbFilteredTransactionsManager(
		types.ArbFilteredTransactionsManagerAddress,
		builder.L2.Client,
	)
	require.NoError(t, err)

	tx, err := arbFilteredTxs.AddFilteredTransaction(&filtererTxOpts, txHash)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)
}

// setupFilteredTxTestBuilder creates a NodeBuilder configured for delayed message filtering tests.
func setupFilteredTxTestBuilder(t *testing.T, ctx context.Context) *NodeBuilder {
	t.Helper()

	// Enable transaction filtering at genesis
	arbOSInit := &params.ArbOSInit{
		TransactionFilteringEnabled: true,
	}

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true). // Need L1 for delayed messages
		WithArbOSVersion(params.ArbosVersion_60).
		WithArbOSInit(arbOSInit)

	builder.isSequencer = true
	builder.nodeConfig.DelayedSequencer.Enable = true
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1

	return builder
}

// TestDelayedMessageFilterHalting verifies that the sequencer halts on a filtered delayed message.
func TestDelayedMessageFilterHalting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")

	// Get initial balance
	initialBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)

	// Set up address filter to block FilteredUser
	filter := newHashedChecker([]common.Address{filteredAddr})

	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare and send delayed tx TO filtered address
	delayedTx := builder.L2Info.PrepareTx("Sender", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on this tx
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash}, 10*time.Second)

	// Verify balance did NOT change (block not created)
	finalBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, initialBalance, finalBalance, "filtered address balance should not change")
}

// TestDelayedMessageFilterBypass verifies that adding tx hash to onchain filter allows tx to proceed.
func TestDelayedMessageFilterBypass(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")

	transactionFiltererStack, transactionFiltererAPI := createTransactionFiltererService(t, ctx, builder, "Filterer")
	defer transactionFiltererStack.Close()

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")

	// Get initial balance
	initialBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Set up address filter to block FilteredUser
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare and send delayed tx TO filtered address
	delayedTx := builder.L2Info.PrepareTx("Sender", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash}, 10*time.Second)

	// Verify balance did NOT change yet
	midBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, initialBalance, midBalance, "balance should not change while halted")

	// Get sender's nonce and balance before bypass
	senderAddr := builder.L2Info.GetAddress("Sender")
	senderNonceBefore, err := builder.L2.Client.NonceAt(ctx, senderAddr, nil)
	require.NoError(t, err)
	senderBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, senderAddr, nil)
	require.NoError(t, err)

	// Set Sequencer client in transactionFiltererAPI, this will eventually add tx hash to onchain filter
	err = transactionFiltererAPI.SetSequencerClient(t, builder.L2.Client)
	require.NoError(t, err)

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Advance L1 again to ensure delayed message is processed
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify filtered address balance did NOT change (tx executed as no-op)
	finalBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, initialBalance, finalBalance, "filtered address should NOT receive funds - tx executed as no-op")

	// Verify the receipt exists and has failed status
	receipt, err := builder.L2.Client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "bypassed tx should have failed receipt status")

	// Verify sender's nonce was incremented
	senderNonceAfter, err := builder.L2.Client.NonceAt(ctx, senderAddr, nil)
	require.NoError(t, err)
	require.Equal(t, senderNonceBefore+1, senderNonceAfter, "sender nonce should be incremented")

	// Verify sender's balance decreased (gas was consumed)
	senderBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, senderAddr, nil)
	require.NoError(t, err)
	require.True(t, senderBalanceAfter.Cmp(senderBalanceBefore) < 0, "sender balance should decrease due to gas consumption")
}

func TestEnableDelayedSequencingFilterDangerousConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)
	// Even though the transaction will touch a filtered address,
	// the sequencer will process the delayed msg, since this config is set to true.
	builder.execConfig.Sequencer.TransactionFiltering.Dangerous.DisableDelayedSequencingFilter = true

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("Sender")

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	senderAddr := builder.L2Info.GetAddress("Sender")

	// Get initial info
	filteredBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	senderNonceBefore, err := builder.L2.Client.NonceAt(ctx, senderAddr, nil)
	require.NoError(t, err)
	senderBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, senderAddr, nil)
	require.NoError(t, err)

	// Set up address filter to block FilteredUser
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare and send delayed tx TO filtered address
	transferAmount := big.NewInt(1e12)
	delayedTx := builder.L2Info.PrepareTx("Sender", "FilteredUser", builder.L2Info.TransferGas, transferAmount, nil)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify filtered address balance changed
	finalBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Add(filteredBalanceBefore, transferAmount), finalBalance, "filtered address should receive funds")

	// Verify the receipt exists and has a successful status
	receipt, err := builder.L2.Client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "bypassed tx should have successful receipt status")

	// Verify sender's nonce was incremented
	senderNonceAfter, err := builder.L2.Client.NonceAt(ctx, senderAddr, nil)
	require.NoError(t, err)
	require.Equal(t, senderNonceBefore+1, senderNonceAfter, "sender nonce should be incremented")

	// Verify sender's balance decreased (gas was consumed)
	senderBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, senderAddr, nil)
	require.NoError(t, err)
	require.True(t, senderBalanceAfter.Cmp(senderBalanceBefore) < 0, "sender balance should decrease due to gas consumption")
}

// TestDelayedMessageFilterBlocksSubsequent verifies that messages behind filtered one are blocked.
func TestDelayedMessageFilterBlocksSubsequent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser1")
	builder.L2Info.GenerateAccount("NormalUser2")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")

	transactionFiltererStack, transactionFiltererAPI := createTransactionFiltererService(t, ctx, builder, "Filterer")
	defer transactionFiltererStack.Close()

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normal1Addr := builder.L2Info.GetAddress("NormalUser1")
	normal2Addr := builder.L2Info.GetAddress("NormalUser2")
	senderAddr := builder.L2Info.GetAddress("Sender")

	// Get initial balances
	filteredInitial, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	normal1Initial, err := builder.L2.Client.BalanceAt(ctx, normal1Addr, nil)
	require.NoError(t, err)
	normal2Initial, err := builder.L2.Client.BalanceAt(ctx, normal2Addr, nil)
	require.NoError(t, err)
	senderBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, senderAddr, nil)
	require.NoError(t, err)

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Set up address filter to block FilteredUser
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Send 3 delayed messages:
	// 1. TO FilteredUser (will be filtered)
	// 2. TO NormalUser1 (should be blocked behind first)
	// 3. TO NormalUser2 (should be blocked behind first)
	delayedTx1 := builder.L2Info.PrepareTx("Sender", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	txHash1 := sendDelayedTx(t, ctx, builder, delayedTx1)

	delayedTx2 := builder.L2Info.PrepareTx("Sender", "NormalUser1", builder.L2Info.TransferGas, big.NewInt(2e12), nil)
	sendDelayedTx(t, ctx, builder, delayedTx2)

	delayedTx3 := builder.L2Info.PrepareTx("Sender", "NormalUser2", builder.L2Info.TransferGas, big.NewInt(3e12), nil)
	sendDelayedTx(t, ctx, builder, delayedTx3)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on first tx
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash1}, 10*time.Second)

	// Verify ALL balances unchanged (all messages blocked)
	filteredMid, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, filteredInitial, filteredMid, "filtered user balance should not change")

	normal1Mid, err := builder.L2.Client.BalanceAt(ctx, normal1Addr, nil)
	require.NoError(t, err)
	require.Equal(t, normal1Initial, normal1Mid, "normal user 1 balance should not change while blocked")

	normal2Mid, err := builder.L2.Client.BalanceAt(ctx, normal2Addr, nil)
	require.NoError(t, err)
	require.Equal(t, normal2Initial, normal2Mid, "normal user 2 balance should not change while blocked")

	// Set Sequencer client in transactionFiltererAPI, this will eventually add tx hash to onchain filter
	err = transactionFiltererAPI.SetSequencerClient(t, builder.L2.Client)
	require.NoError(t, err)

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Advance L1 again to ensure all delayed messages are processed
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify filtered tx executed as no-op (balance unchanged)
	filteredFinal, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, filteredInitial, filteredFinal, "filtered user should NOT receive funds - tx executed as no-op")

	// Verify sender's balance decreased due to gas consumption from all 3 txs
	// (1 no-op filtered tx that burned gas + 2 normal transfers)
	senderBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, senderAddr, nil)
	require.NoError(t, err)
	// Sender sent 2e12 + 3e12 = 5e12 in the two successful transfers, plus gas for all 3 txs
	expectedMaxBalance := new(big.Int).Sub(senderBalanceBefore, big.NewInt(5e12))
	require.True(t, senderBalanceAfter.Cmp(expectedMaxBalance) < 0,
		"sender balance should be less than (before - transfers) due to gas consumption on all txs including no-op")

	// Verify non-filtered messages were processed normally
	normal1Final, err := builder.L2.Client.BalanceAt(ctx, normal1Addr, nil)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Add(normal1Initial, big.NewInt(2e12)), normal1Final, "normal user 1 should receive funds after unblock")

	normal2Final, err := builder.L2.Client.BalanceAt(ctx, normal2Addr, nil)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Add(normal2Initial, big.NewInt(3e12)), normal2Final, "normal user 2 should receive funds after unblock")
}

// TestDelayedMessageFilterBatch verifies that the sequencer correctly handles a batched delayed message
// containing multiple transactions where one (not the first) touches a filtered address.
func TestDelayedMessageFilterBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("User1")
	builder.L2Info.GenerateAccount("User2")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")

	transactionFiltererStack, transactionFiltererAPI := createTransactionFiltererService(t, ctx, builder, "Filterer")
	defer transactionFiltererStack.Close()

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	user1Addr := builder.L2Info.GetAddress("User1")
	user2Addr := builder.L2Info.GetAddress("User2")

	// Get initial balances
	filteredInitial, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	user1Initial, err := builder.L2.Client.BalanceAt(ctx, user1Addr, nil)
	require.NoError(t, err)
	user2Initial, err := builder.L2.Client.BalanceAt(ctx, user2Addr, nil)
	require.NoError(t, err)

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Set up address filter to block FilteredUser
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Create batch of 3 transactions within a single delayed message:
	// tx1: Sender -> User1 (normal transfer, NOT filtered)
	// tx2: Sender -> FilteredUser (will be filtered - note this is NOT the first tx)
	// tx3: Sender -> User2 (normal transfer, NOT filtered)
	transferAmount := big.NewInt(1e15)
	tx1 := builder.L2Info.PrepareTx("Sender", "User1", builder.L2Info.TransferGas, transferAmount, nil)
	tx2 := builder.L2Info.PrepareTx("Sender", "FilteredUser", builder.L2Info.TransferGas, transferAmount, nil)
	tx3 := builder.L2Info.PrepareTx("Sender", "User2", builder.L2Info.TransferGas, transferAmount, nil)

	txBatch := types.Transactions{tx1, tx2, tx3}

	// Send as a single batched delayed message
	sendDelayedBatch(t, ctx, builder, txBatch)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on tx2 (the filtered one, which is NOT the first in the batch)
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{tx2.Hash()}, 10*time.Second)

	// Verify ALL balances unchanged while halted (entire batch is blocked)
	filteredMid, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, filteredInitial, filteredMid, "filtered user balance should not change while halted")

	user1Mid, err := builder.L2.Client.BalanceAt(ctx, user1Addr, nil)
	require.NoError(t, err)
	require.Equal(t, user1Initial, user1Mid, "user1 balance should not change while batch is blocked")

	user2Mid, err := builder.L2.Client.BalanceAt(ctx, user2Addr, nil)
	require.NoError(t, err)
	require.Equal(t, user2Initial, user2Mid, "user2 balance should not change while batch is blocked")

	// Add tx2 hash to onchain filter
	err = transactionFiltererAPI.SetSequencerClient(t, builder.L2.Client)
	require.NoError(t, err)

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Advance L1 again to ensure all delayed messages are processed
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify final balances:
	// tx1: User1 should receive transferAmount
	user1Final, err := builder.L2.Client.BalanceAt(ctx, user1Addr, nil)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Add(user1Initial, transferAmount), user1Final,
		"user1 should receive funds from tx1 after unblock")

	// tx2: FilteredUser should NOT receive (tx executed as no-op)
	filteredFinal, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, filteredInitial, filteredFinal,
		"filtered user should NOT receive funds - tx2 executed as no-op")

	// tx3: User2 should receive transferAmount
	user2Final, err := builder.L2.Client.BalanceAt(ctx, user2Addr, nil)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Add(user2Initial, transferAmount), user2Final,
		"user2 should receive funds from tx3 after unblock")

	// Verify tx2 receipt exists and has failed status
	receipt, err := builder.L2.Client.TransactionReceipt(ctx, tx2.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status,
		"bypassed tx2 should have failed receipt status")
}

// TestDelayedMessageFilterNonFilteredPasses verifies that non-filtered delayed messages pass without issue.
func TestDelayedMessageFilterNonFilteredPasses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalAddr := builder.L2Info.GetAddress("NormalUser")

	// Get initial balance
	initialBalance, err := builder.L2.Client.BalanceAt(ctx, normalAddr, nil)
	require.NoError(t, err)

	// Set up address filter to block FilteredUser (NOT NormalUser)
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare and send delayed tx TO normal (non-filtered) address
	delayedTx := builder.L2Info.PrepareTx("Sender", "NormalUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Give some time for processing
	<-time.After(time.Second)

	// Verify sequencer is NOT halted
	_, waiting := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx(t)
	require.False(t, waiting, "sequencer should not be halted for non-filtered address")

	// Verify balance DID change (message processed normally)
	finalBalance, err := builder.L2.Client.BalanceAt(ctx, normalAddr, nil)
	require.NoError(t, err)
	expectedBalance := new(big.Int).Add(initialBalance, big.NewInt(1e12))
	require.Equal(t, expectedBalance, finalBalance, "normal address should receive funds")
}

// deployAddressFilterTestContractForDelayed deploys the AddressFilterTest contract via regular L2 tx.
func deployAddressFilterTestContractForDelayed(t *testing.T, ctx context.Context, builder *NodeBuilder) (common.Address, *localgen.AddressFilterTest) {
	t.Helper()
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	addr, tx, contract, err := localgen.DeployAddressFilterTest(&auth, builder.L2.Client)
	require.NoError(t, err, "could not deploy AddressFilterTest contract")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)
	return addr, contract
}

// prepareDelayedContractCall prepares a delayed tx that calls a contract method.
func prepareDelayedContractCall(t *testing.T, builder *NodeBuilder, sender string, contract common.Address, data []byte) *types.Transaction {
	t.Helper()
	info := builder.L2Info.GetInfoWithPrivKey(sender)
	nonce := info.Nonce.Add(1) - 1
	chainID := builder.L2Info.Signer.ChainID()

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: big.NewInt(1e9),
		GasFeeCap: big.NewInt(1e12),
		Gas:       500000,
		To:        &contract,
		Value:     big.NewInt(0),
		Data:      data,
	})

	signedTx, err := types.SignTx(tx, types.NewLondonSigner(chainID), info.PrivateKey)
	require.NoError(t, err)
	return signedTx
}

// TestDelayedMessageFilterCall verifies that a delayed message CALLing a filtered contract halts the sequencer.
func TestDelayedMessageFilterCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")

	transactionFiltererStack, transactionFiltererAPI := createTransactionFiltererService(t, ctx, builder, "Filterer")
	defer transactionFiltererStack.Close()

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Deploy caller contract (not filtered)
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	// Deploy target contract (will be filtered)
	targetAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	// Set up filter to block the target contract
	filter := newHashedChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare delayed tx that calls caller.callTarget(targetAddr)
	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	callData, err := callerABI.Pack("callTarget", targetAddr)
	require.NoError(t, err)

	delayedTx := prepareDelayedContractCall(t, builder, "Sender", callerAddr, callData)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on this tx
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash}, 10*time.Second)

	// Set Sequencer client in transactionFiltererAPI, this will eventually add tx hash to onchain filter
	err = transactionFiltererAPI.SetSequencerClient(t, builder.L2.Client)
	require.NoError(t, err)

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Verify the tx was processed as a no-op (failed receipt, no execution)
	receipt, err := builder.L2.Client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "bypassed tx should have failed receipt status")

	// Verify no events were emitted (tx was not executed)
	require.Empty(t, receipt.Logs, "no events should be emitted for no-op execution")
}

// TestDelayedMessageFilterStaticCall verifies that a delayed message STATICCALLing a filtered contract halts the sequencer.
func TestDelayedMessageFilterStaticCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")

	transactionFiltererStack, transactionFiltererAPI := createTransactionFiltererService(t, ctx, builder, "Filterer")
	defer transactionFiltererStack.Close()

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Deploy caller contract (not filtered)
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	// Deploy target contract (will be filtered)
	targetAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	// Set up filter to block the target contract
	filter := newHashedChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare delayed tx that calls caller.staticcallTargetInTx(targetAddr)
	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	callData, err := callerABI.Pack("staticcallTargetInTx", targetAddr)
	require.NoError(t, err)

	delayedTx := prepareDelayedContractCall(t, builder, "Sender", callerAddr, callData)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on this tx
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash}, 10*time.Second)

	// Set Sequencer client in transactionFiltererAPI, this will eventually add tx hash to onchain filter
	err = transactionFiltererAPI.SetSequencerClient(t, builder.L2.Client)
	require.NoError(t, err)

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Verify the tx was processed
	receipt, err := builder.L2.Client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "bypassed tx should have failed receipt status")
}

// TestDelayedMessageFilterCreate verifies that a delayed message CREATing at a filtered address halts the sequencer.
func TestDelayedMessageFilterCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")

	transactionFiltererStack, transactionFiltererAPI := createTransactionFiltererService(t, ctx, builder, "Filterer")
	defer transactionFiltererStack.Close()

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Deploy caller contract (not filtered)
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	// Get the current nonce of the caller contract to compute CREATE address
	nonce, err := builder.L2.Client.NonceAt(ctx, callerAddr, nil)
	require.NoError(t, err)

	// Compute the CREATE address based on caller's address and nonce
	createAddr := crypto.CreateAddress(callerAddr, nonce)

	// Set up filter to block the computed CREATE address
	filter := newHashedChecker([]common.Address{createAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare delayed tx that calls caller.createContract()
	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	callData, err := callerABI.Pack("createContract")
	require.NoError(t, err)

	delayedTx := prepareDelayedContractCall(t, builder, "Sender", callerAddr, callData)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on this tx
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash}, 10*time.Second)

	// Set Sequencer client in transactionFiltererAPI, this will eventually add tx hash to onchain filter
	err = transactionFiltererAPI.SetSequencerClient(t, builder.L2.Client)
	require.NoError(t, err)

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Verify the tx was processed
	receipt, err := builder.L2.Client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "bypassed tx should have failed receipt status")
}

// TestDelayedMessageFilterCreate2 verifies that a delayed message CREATE2ing at a filtered address halts the sequencer.
func TestDelayedMessageFilterCreate2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")

	transactionFiltererStack, transactionFiltererAPI := createTransactionFiltererService(t, ctx, builder, "Filterer")
	defer transactionFiltererStack.Close()

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Deploy caller contract (not filtered)
	callerAddr, caller := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	// Compute the CREATE2 address for a known salt
	salt := [32]byte{1, 2, 3}
	create2Addr, err := caller.ComputeCreate2Address(&bind.CallOpts{Context: ctx}, salt)
	require.NoError(t, err)

	// Set up filter to block the computed CREATE2 address
	filter := newHashedChecker([]common.Address{create2Addr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare delayed tx that calls caller.create2Contract(salt)
	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	callData, err := callerABI.Pack("create2Contract", salt)
	require.NoError(t, err)

	delayedTx := prepareDelayedContractCall(t, builder, "Sender", callerAddr, callData)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on this tx
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash}, 10*time.Second)

	// Set Sequencer client in transactionFiltererAPI, this will eventually add tx hash to onchain filter
	err = transactionFiltererAPI.SetSequencerClient(t, builder.L2.Client)
	require.NoError(t, err)

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Verify the tx was processed
	receipt, err := builder.L2.Client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "bypassed tx should have failed receipt status")
}

// TestDelayedMessageFilterSelfdestruct verifies that a delayed message SELFDESTRUCTing to a filtered beneficiary halts the sequencer.
func TestDelayedMessageFilterSelfdestruct(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
	builder.L2Info.GenerateAccount("FilteredBeneficiary")

	transactionFiltererStack, transactionFiltererAPI := createTransactionFiltererService(t, ctx, builder, "Filterer")
	defer transactionFiltererStack.Close()

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	filteredBeneficiary := builder.L2Info.GetAddress("FilteredBeneficiary")

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Deploy contract that will selfdestruct (not filtered initially)
	contractAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	// Set up filter to block the beneficiary address
	filter := newHashedChecker([]common.Address{filteredBeneficiary})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare delayed tx that calls contract.selfDestructTo(filteredBeneficiary)
	contractABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	callData, err := contractABI.Pack("selfDestructTo", filteredBeneficiary)
	require.NoError(t, err)

	delayedTx := prepareDelayedContractCall(t, builder, "Sender", contractAddr, callData)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on this tx
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash}, 10*time.Second)

	// Set Sequencer client in transactionFiltererAPI, this will eventually add tx hash to onchain filter
	err = transactionFiltererAPI.SetSequencerClient(t, builder.L2.Client)
	require.NoError(t, err)

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Verify the tx was processed
	receipt, err := builder.L2.Client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "bypassed tx should have failed receipt status")
}

// TestDelayedMessageFilterTxHashesUpdateOnchainFilter verifies that TxHashes updates when one tx
// from a batch gets added to the onchain filter.
func TestDelayedMessageFilterTxHashesUpdateOnchainFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)
	// Configure short retry interval so we don't have to wait long
	builder.nodeConfig.DelayedSequencer.FilteredTxFullRetryInterval = 200 * time.Millisecond
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser1")
	builder.L2Info.GenerateAccount("FilteredUser2")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	filteredAddr1 := builder.L2Info.GetAddress("FilteredUser1")
	filteredAddr2 := builder.L2Info.GetAddress("FilteredUser2")

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Set up address filter to block both FilteredUser1 and FilteredUser2
	filter := newHashedChecker([]common.Address{filteredAddr1, filteredAddr2})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Create batch of 2 transactions within a single delayed message:
	// tx1: Sender -> FilteredUser1 (filtered)
	// tx2: Sender -> FilteredUser2 (filtered)
	transferAmount := big.NewInt(1e15)
	tx1 := builder.L2Info.PrepareTx("Sender", "FilteredUser1", builder.L2Info.TransferGas, transferAmount, nil)
	tx2 := builder.L2Info.PrepareTx("Sender", "FilteredUser2", builder.L2Info.TransferGas, transferAmount, nil)

	txBatch := types.Transactions{tx1, tx2}

	// Send as a single batched delayed message
	sendDelayedBatch(t, ctx, builder, txBatch)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on both tx1 and tx2
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{tx1.Hash(), tx2.Hash()}, 10*time.Second)

	// Add tx1 to onchain filter (but NOT tx2)
	addTxHashToOnChainFilter(t, ctx, builder, tx1.Hash(), "Filterer")

	// Wait for full retry to occur and verify TxHashes updated to only tx2
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{tx2.Hash()}, 5*time.Second)

	// Add tx2 to onchain filter
	addTxHashToOnChainFilter(t, ctx, builder, tx2.Hash(), "Filterer")

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Advance L1 again to ensure delayed message is processed
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify both receipts exist and have failed status (executed as no-ops)
	receipt1, err := builder.L2.Client.TransactionReceipt(ctx, tx1.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt1.Status, "tx1 should have failed receipt status")

	receipt2, err := builder.L2.Client.TransactionReceipt(ctx, tx2.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt2.Status, "tx2 should have failed receipt status")
}

// TestDelayedMessageFilterTxHashesUpdateAddressSetChange verifies that TxHashes updates when
// the filtered address set changes.
func TestDelayedMessageFilterTxHashesUpdateAddressSetChange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)
	// Configure short retry interval so we don't have to wait long
	builder.nodeConfig.DelayedSequencer.FilteredTxFullRetryInterval = 200 * time.Millisecond
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("User1")
	builder.L2Info.GenerateAccount("User2")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)

	user1Addr := builder.L2Info.GetAddress("User1")
	user2Addr := builder.L2Info.GetAddress("User2")

	// Get initial balances
	user1Initial, err := builder.L2.Client.BalanceAt(ctx, user1Addr, nil)
	require.NoError(t, err)
	user2Initial, err := builder.L2.Client.BalanceAt(ctx, user2Addr, nil)
	require.NoError(t, err)

	// Set up address filter to block both User1 and User2
	filter := newHashedChecker([]common.Address{user1Addr, user2Addr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Create batch of 2 transactions within a single delayed message:
	// tx1: Sender -> User1 (filtered)
	// tx2: Sender -> User2 (filtered)
	transferAmount := big.NewInt(1e15)
	tx1 := builder.L2Info.PrepareTx("Sender", "User1", builder.L2Info.TransferGas, transferAmount, nil)
	tx2 := builder.L2Info.PrepareTx("Sender", "User2", builder.L2Info.TransferGas, transferAmount, nil)

	txBatch := types.Transactions{tx1, tx2}

	// Send as a single batched delayed message
	sendDelayedBatch(t, ctx, builder, txBatch)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on both tx1 and tx2
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{tx1.Hash(), tx2.Hash()}, 10*time.Second)

	// Change the address filter to only filter User2 (remove User1 from filter)
	newFilter := newHashedChecker([]common.Address{user2Addr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(newFilter)

	// Wait for full retry to occur and verify TxHashes updated to only tx2
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{tx2.Hash()}, 5*time.Second)

	// Change the address filter to filter neither (remove User2 from filter)
	noFilter := newHashedChecker([]common.Address{})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(noFilter)

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Advance L1 again to ensure delayed message is processed
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify tx1 succeeded (User1 received funds - no longer filtered)
	user1Final, err := builder.L2.Client.BalanceAt(ctx, user1Addr, nil)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Add(user1Initial, transferAmount), user1Final,
		"User1 should receive funds - no longer filtered")

	// Verify tx2 also succeeded (User2 received funds - no longer filtered)
	user2Final, err := builder.L2.Client.BalanceAt(ctx, user2Addr, nil)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Add(user2Initial, transferAmount), user2Final,
		"User2 should receive funds - no longer filtered")

	// Verify both receipts exist and have success status
	receipt1, err := builder.L2.Client.TransactionReceipt(ctx, tx1.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt1.Status, "tx1 should have success status")

	receipt2, err := builder.L2.Client.TransactionReceipt(ctx, tx2.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt2.Status, "tx2 should have success status")
}

// TestDelayedMessageFilterAliasedSender verifies that when an unsigned delayed
// message is sent via sendUnsignedTransaction, the original (non-aliased) L1
// sender address is checked against the address filter. Without de-aliasing,
// the filter would only see the aliased L2 address and miss the restricted
// original address.
func TestDelayedMessageFilterAliasedSender(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)
	cleanup := builder.Build(t)
	defer cleanup()

	// The L1 sender whose ORIGINAL address will be filtered
	l1SenderAddr := builder.L1Info.GetAddress("User")

	// Compute the aliased L2 address (what the sender appears as on L2 for unsigned txs)
	aliasedSenderAddr := arbosutil.RemapL1Address(l1SenderAddr)

	// Fund the aliased address on L2 so the unsigned tx can pay for gas
	builder.L2.TransferBalanceTo(t, "Owner", aliasedSenderAddr, big.NewInt(1e18), builder.L2Info)

	// Create a recipient account (not filtered)
	builder.L2Info.GenerateAccount("Recipient")
	recipientAddr := builder.L2Info.GetAddress("Recipient")

	// Get initial balance
	initialBalance, err := builder.L2.Client.BalanceAt(ctx, recipientAddr, nil)
	require.NoError(t, err)

	// Set up address filter to block the ORIGINAL L1 address (NOT the aliased one).
	// Sanctions lists contain original addresses, not aliased derivatives.
	filter := newHashedChecker([]common.Address{l1SenderAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Get nonce for the aliased address on L2
	nonce, err := builder.L2.Client.NonceAt(ctx, aliasedSenderAddr, nil)
	require.NoError(t, err)

	// Create unsigned tx (ArbitrumUnsignedTx) - matches what L2 node creates from delayed msg
	unsignedTx := types.NewTx(&types.ArbitrumUnsignedTx{
		ChainId:   builder.L2Info.Signer.ChainID(),
		From:      aliasedSenderAddr,
		Nonce:     nonce,
		GasFeeCap: builder.L2Info.GasPrice,
		Gas:       builder.L2Info.TransferGas,
		To:        &recipientAddr,
		Value:     big.NewInt(1e12),
		Data:      nil,
	})

	// Send via L1 delayed inbox using sendUnsignedTransaction.
	// The bridge will alias the sender in _deliverToBridge, so on L2 the
	// sender is aliasedSenderAddr (not l1SenderAddr).
	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	require.NoError(t, err)

	l1opts := builder.L1Info.GetDefaultTransactOpts("User", ctx)
	l1tx, err := delayedInbox.SendUnsignedTransaction(
		&l1opts,
		arbmath.UintToBig(unsignedTx.Gas()),
		unsignedTx.GasFeeCap(),
		arbmath.UintToBig(unsignedTx.Nonce()),
		*unsignedTx.To(),
		unsignedTx.Value(),
		unsignedTx.Data(),
	)
	require.NoError(t, err)
	_, err = builder.L1.EnsureTxSucceeded(l1tx)
	require.NoError(t, err)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on this tx.
	// The filter catches the original L1 address via de-aliasing in PostTxFilter.
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{unsignedTx.Hash()}, 10*time.Second)

	// Verify recipient balance did NOT change (tx was not processed)
	finalBalance, err := builder.L2.Client.BalanceAt(ctx, recipientAddr, nil)
	require.NoError(t, err)
	require.Equal(t, initialBalance, finalBalance, "recipient balance should not change - sender is filtered")
}
