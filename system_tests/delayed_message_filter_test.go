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

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/retryables"
	arbosutil "github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/transaction-filterer/api"
	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
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

// advanceL1ForDelayed advances L1 blocks so the delayed sequencer picks up pending messages.
// Callers should use their own wait mechanism (WaitForTx, waitForDelayedSequencerHaltOnHashes, etc.)
// rather than relying on a sleep.
func advanceL1ForDelayed(t *testing.T, ctx context.Context, builder *NodeBuilder) {
	t.Helper()
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
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

	builder.execConfig.TransactionFiltering.TransactionFiltererRPCClient.URL = transactionFiltererStack.HTTPEndpoint()

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

// verifyCascadingRedeemFiltered verifies the checkpoint-and-revert behavior for
// a cascading redeem filter. The delayed sequencer halts because the auto-redeem
// touched a filtered address. After the operator adds the tx hash to the onchain
// filter, the submission re-processes as filtered (no auto-redeem). Asserts:
//  1. Delayed sequencer halts on the ticketId hash.
//  2. After onchain filter entry, sequencer resumes.
//  3. Submission receipt has failed status (retryable created with redirected beneficiary, auto-redeem skipped).
//  4. Retryable ticket exists with the expected beneficiary.
//  5. No ArbitrumRetryTx in the submission block.
func verifyCascadingRedeemFiltered(t *testing.T, ctx context.Context, builder *NodeBuilder, ticketId common.Hash, filtererName string, expectedBeneficiary common.Address) *types.Receipt {
	t.Helper()

	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{ticketId}, 10*time.Second)
	addTxHashToOnChainFilter(t, ctx, builder, ticketId, filtererName)
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)
	advanceL1ForDelayed(t, ctx, builder)

	submissionReceipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, submissionReceipt.Status)

	arbRetryable, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	Require(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketId)
	require.NoError(t, err, "retryable ticket should still exist")

	beneficiary, err := arbRetryable.GetBeneficiary(&bind.CallOpts{}, ticketId)
	require.NoError(t, err)
	require.Equal(t, expectedBeneficiary, beneficiary, "beneficiary should be redirected to filteredFundsRecipient")

	block, err := builder.L2.Client.BlockByNumber(ctx, submissionReceipt.BlockNumber)
	Require(t, err)
	redeemCount := 0
	for _, btx := range block.Transactions() {
		if btx.Type() == types.ArbitrumRetryTxType {
			redeemCount++
		}
	}
	require.Equal(t, 0, redeemCount, "no redeem should exist - submission was filtered on retry")

	return submissionReceipt
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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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
	advanceL1ForDelayed(t, ctx, builder)

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

// retryableFilterTestParams holds common params for retryable filter tests.
type retryableFilterTestParams struct {
	builder            *NodeBuilder
	ctx                context.Context
	delayedInbox       *bridgegen.Inbox
	lookupL2Tx         func(*types.Receipt) *types.Transaction
	filtererName       string
	fundsRecipientAddr common.Address
}

// setupRetryableFilterTest sets up a node for retryable filtering tests.
// It creates a Filterer account with the transaction-filterer role.
// When setFundsRecipient is true, it also sets the filteredFundsRecipient.
func setupRetryableFilterTest(t *testing.T, ctx context.Context, setFundsRecipient bool, eventFilterRules []eventfilter.EventRule) (*retryableFilterTestParams, func()) {
	t.Helper()
	builder := setupFilteredTxTestBuilder(t, ctx)
	if eventFilterRules != nil {
		builder.WithEventFilterRules(eventFilterRules)
	}
	cleanup := builder.Build(t)

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	require.NoError(t, err)

	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
	require.NoError(t, err)

	lookupL2Tx := func(l1Receipt *types.Receipt) *types.Transaction {
		messages, err := delayedBridge.LookupMessagesInRange(ctx, l1Receipt.BlockNumber, l1Receipt.BlockNumber, nil)
		require.NoError(t, err)
		require.NotEmpty(t, messages, "no delayed messages found")
		var submissionTxs []*types.Transaction
		for _, message := range messages {
			if message.Message.Header.Kind != arbostypes.L1MessageType_SubmitRetryable {
				continue
			}
			txs, err := arbos.ParseL2Transactions(message.Message, chaininfo.ArbitrumDevTestChainConfig().ChainID, params.MaxDebugArbosVersionSupported)
			require.NoError(t, err)
			for _, tx := range txs {
				if tx.Type() == types.ArbitrumSubmitRetryableTxType {
					submissionTxs = append(submissionTxs, tx)
				}
			}
		}
		require.Len(t, submissionTxs, 1, "expected exactly 1 retryable submission tx")
		return submissionTxs[0]
	}

	builder.L2Info.GenerateAccount("Filterer")
	builder.L2Info.GenerateAccount("FundsRecipient")
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	fundsRecipientAddr := builder.L2Info.GetAddress("FundsRecipient")
	if setFundsRecipient {
		tx, err = arbOwner.SetFilteredFundsRecipient(&ownerTxOpts, fundsRecipientAddr)
		require.NoError(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		require.NoError(t, err)
	}

	return &retryableFilterTestParams{
		builder:            builder,
		ctx:                ctx,
		delayedInbox:       delayedInbox,
		lookupL2Tx:         lookupL2Tx,
		filtererName:       "Filterer",
		fundsRecipientAddr: fundsRecipientAddr,
	}, cleanup
}

// submitRetryableViaL1 submits a retryable ticket via the L1 delayed inbox.
// Returns the L1 receipt and the L2 submission tx hash (ticketId).
func submitRetryableViaL1(
	t *testing.T,
	p *retryableFilterTestParams,
	l1Sender string,
	destAddr common.Address,
	callValue *big.Int,
	beneficiary common.Address,
	feeRefundAddr common.Address,
	data []byte,
) (*types.Receipt, common.Hash) {
	t.Helper()

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	maxSubmissionCost := big.NewInt(1e16)
	gasLimit := big.NewInt(100000)
	maxFeePerGas := big.NewInt(l2pricing.InitialBaseFeeWei * 2)

	l1opts := p.builder.L1Info.GetDefaultTransactOpts(l1Sender, p.ctx)
	l1opts.Value = deposit
	l1tx, err := p.delayedInbox.CreateRetryableTicket(
		&l1opts,
		destAddr,
		callValue,
		maxSubmissionCost,
		feeRefundAddr,
		beneficiary,
		gasLimit,
		maxFeePerGas,
		data,
	)
	require.NoError(t, err)

	l1Receipt, err := p.builder.L1.EnsureTxSucceeded(l1tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, l1Receipt.Status)

	l2Tx := p.lookupL2Tx(l1Receipt)
	return l1Receipt, l2Tx.Hash()
}

func TestFilteredRetryableRedirectWithExplicitRecipient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("Destination")

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	destAddr := builder.L2Info.GetAddress("Destination")

	// Set up address filter to block FilteredUser
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Record initial balance of filtered address
	filteredInitialBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)

	// Submit retryable with filtered beneficiary and feeRefundAddr
	_, ticketId := submitRetryableViaL1(t, p, "Faucet", destAddr, common.Big0, filteredAddr, filteredAddr, nil)

	// Advance L1 to trigger delayed message processing
	advanceL1ForDelayed(t, ctx, builder)

	// Sequencer should halt because PostTxFilter touches beneficiary/feeRefundAddr
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{ticketId}, 10*time.Second)

	// Verify filtered address balance did not change while halted
	filteredMidBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, filteredInitialBalance, filteredMidBalance, "balance should not change while halted")

	// Add tx hash to onchain filter to authorize processing
	addTxHashToOnChainFilter(t, ctx, builder, ticketId, "Filterer")

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Advance L1 again to ensure delayed message is processed
	advanceL1ForDelayed(t, ctx, builder)

	// Wait for the L2 receipt
	receipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "filtered retryable should have failed receipt status")

	// Verify retryable was created with redirected beneficiary
	arbRetryable, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	beneficiary, err := arbRetryable.GetBeneficiary(&bind.CallOpts{}, ticketId)
	require.NoError(t, err)
	require.Equal(t, p.fundsRecipientAddr, beneficiary, "retryable beneficiary should be redirected to FundsRecipient")

	// Verify filtered address balance did not increase
	filteredFinalBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, filteredInitialBalance, filteredFinalBalance, "filtered address should not receive any funds")

	// Verify redirect address received fee refunds
	redirectBalance, err := builder.L2.Client.BalanceAt(ctx, p.fundsRecipientAddr, nil)
	require.NoError(t, err)
	require.True(t, redirectBalance.Sign() > 0, "redirect address should have received fee refunds")

	// Verify sequencer is not re-halted
	_, waiting := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx(t)
	require.False(t, waiting, "sequencer should not be re-halted after processing filtered retryable")
}

func TestFilteredRetryableRedirectFallbackToNetworkFee(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, false, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("Destination")

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	destAddr := builder.L2Info.GetAddress("Destination")

	// Get the networkFeeAccount for comparison
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, builder.L2.Client)
	require.NoError(t, err)
	networkFeeAccount, err := arbOwnerPublic.GetNetworkFeeAccount(&bind.CallOpts{})
	require.NoError(t, err)

	// Verify filteredFundsRecipient is zero
	configuredRecipient, err := arbOwnerPublic.GetFilteredFundsRecipient(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, common.Address{}, configuredRecipient, "filteredFundsRecipient should be zero")

	// Set up address filter
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	filteredInitialBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)

	// Snapshot networkFeeAccount balance before retryable processing
	nfaBefore, err := builder.L2.Client.BalanceAt(ctx, networkFeeAccount, nil)
	require.NoError(t, err)

	// Submit retryable with filtered beneficiary
	_, ticketId := submitRetryableViaL1(t, p, "Faucet", destAddr, common.Big0, filteredAddr, filteredAddr, nil)

	// Advance L1 to trigger delayed message processing
	advanceL1ForDelayed(t, ctx, builder)

	// Sequencer should halt on filtered beneficiary/feeRefundAddr
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{ticketId}, 10*time.Second)

	// Add tx hash to onchain filter to authorize processing
	addTxHashToOnChainFilter(t, ctx, builder, ticketId, "Filterer")

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Advance L1 again to ensure delayed message is processed
	advanceL1ForDelayed(t, ctx, builder)

	// Wait for receipt
	receipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

	// Verify beneficiary fell back to networkFeeAccount
	arbRetryable, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	beneficiary, err := arbRetryable.GetBeneficiary(&bind.CallOpts{}, ticketId)
	require.NoError(t, err)
	require.Equal(t, networkFeeAccount, beneficiary, "retryable beneficiary should fallback to networkFeeAccount")

	// Verify filtered address untouched
	filteredFinalBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, filteredInitialBalance, filteredFinalBalance, "filtered address should not receive any funds")

	// Verify networkFeeAccount balance increased from fee refunds
	nfaAfter, err := builder.L2.Client.BalanceAt(ctx, networkFeeAccount, nil)
	require.NoError(t, err)
	require.True(t, nfaAfter.Cmp(nfaBefore) > 0, "network fee account should have received fee refunds")
}

func TestFilteredRetryableNoRedirectWhenNotFiltered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalBeneficiary")
	builder.L2Info.GenerateAccount("Destination")

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	normalBeneficiary := builder.L2Info.GetAddress("NormalBeneficiary")
	destAddr := builder.L2Info.GetAddress("Destination")

	// Set up address filter to block FilteredUser only (NOT NormalBeneficiary)
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Submit retryable with non-filtered beneficiary
	_, ticketId := submitRetryableViaL1(t, p, "Faucet", destAddr, common.Big0, normalBeneficiary, normalBeneficiary, nil)

	advanceL1ForDelayed(t, ctx, builder)

	// Wait for the L2 receipt — this should succeed normally.
	// A successful receipt proves no filtering happened (filtered retryables get ReceiptStatusFailed).
	// We can't check GetBeneficiary because the auto-redeem runs immediately
	// and deletes the retryable ticket on success.
	receipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "non-filtered retryable should succeed")

	// Verify sequencer is NOT halted
	_, waiting := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx(t)
	require.False(t, waiting, "sequencer should not be halted for non-filtered retryable")
}

func TestFilteredRetryableWithCallValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("Destination")

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	destAddr := builder.L2Info.GetAddress("Destination")

	// Set up address filter
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	callValue := big.NewInt(1e6)

	// Submit retryable with call value and filtered beneficiary
	_, ticketId := submitRetryableViaL1(t, p, "Faucet", destAddr, callValue, filteredAddr, filteredAddr, nil)

	// Advance L1 to trigger delayed message processing
	advanceL1ForDelayed(t, ctx, builder)

	// Sequencer should halt on filtered beneficiary/feeRefundAddr
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{ticketId}, 10*time.Second)

	// Add tx hash to onchain filter to authorize processing
	addTxHashToOnChainFilter(t, ctx, builder, ticketId, "Filterer")

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Advance L1 again to ensure delayed message is processed
	advanceL1ForDelayed(t, ctx, builder)

	// Wait for receipt
	receipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

	// Verify retryable beneficiary is redirected
	arbRetryable, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	beneficiary, err := arbRetryable.GetBeneficiary(&bind.CallOpts{}, ticketId)
	require.NoError(t, err)
	require.Equal(t, p.fundsRecipientAddr, beneficiary, "retryable beneficiary should be redirected")

	// Verify escrow holds the call value
	escrowAddr := retryables.RetryableEscrowAddress(ticketId)
	state, err := builder.L2.ExecNode.ArbInterface.BlockChain().State()
	require.NoError(t, err)
	escrowBalance := state.GetBalance(escrowAddr)
	require.Equal(t, callValue, escrowBalance.ToBig(), "escrow should hold the call value")

	// Verify filtered address did not receive anything
	filteredBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.True(t, filteredBalance.Sign() == 0, "filtered address should have zero balance")

	// Verify redirect address received fee refunds
	redirectBalance, err := builder.L2.Client.BalanceAt(ctx, p.fundsRecipientAddr, nil)
	require.NoError(t, err)
	require.True(t, redirectBalance.Sign() > 0, "redirect address should have received fee refunds")
}

func TestFilteredRetryableSequencerDoesNotReHalt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("Destination")
	builder.L2Info.GenerateAccount("NormalRecipient")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	destAddr := builder.L2Info.GetAddress("Destination")
	normalRecipientAddr := builder.L2Info.GetAddress("NormalRecipient")

	// Set up address filter
	filter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Record initial balance of normal recipient
	normalInitialBalance, err := builder.L2.Client.BalanceAt(ctx, normalRecipientAddr, nil)
	require.NoError(t, err)

	// Submit filtered retryable via L1
	_, ticketId := submitRetryableViaL1(t, p, "Faucet", destAddr, common.Big0, filteredAddr, filteredAddr, nil)

	// Submit a normal delayed transfer behind it
	transferAmount := big.NewInt(1e12)
	delayedTx := builder.L2Info.PrepareTx("Sender", "NormalRecipient", builder.L2Info.TransferGas, transferAmount, nil)
	delayedTxHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceL1ForDelayed(t, ctx, builder)

	// Sequencer should halt on the filtered retryable
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{ticketId}, 10*time.Second)

	// Add tx hash to onchain filter to authorize processing
	addTxHashToOnChainFilter(t, ctx, builder, ticketId, "Filterer")

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Advance L1 again to ensure all delayed messages are processed
	advanceL1ForDelayed(t, ctx, builder)

	// Verify filtered retryable processed with redirect
	receipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "filtered retryable should have failed receipt")

	// Wait for the normal delayed transfer to also be processed. The delayed sequencer
	// may not have sequenced it in the same iteration as the retryable if the inbox
	// reader hadn't indexed it yet when the sequencer resumed.
	_, err = WaitForTx(ctx, builder.L2.Client, delayedTxHash, time.Second*10)
	require.NoError(t, err)

	arbRetryable, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	beneficiary, err := arbRetryable.GetBeneficiary(&bind.CallOpts{}, ticketId)
	require.NoError(t, err)
	require.Equal(t, p.fundsRecipientAddr, beneficiary, "retryable beneficiary should be redirected")

	// Verify the normal delayed transfer behind it also processed (no re-halt)
	normalFinalBalance, err := builder.L2.Client.BalanceAt(ctx, normalRecipientAddr, nil)
	require.NoError(t, err)
	expectedBalance := new(big.Int).Add(normalInitialBalance, transferAmount)
	require.Equal(t, expectedBalance, normalFinalBalance, "normal transfer should be processed after filtered retryable")

	// Verify redirect address received fee refunds
	redirectBalance, err := builder.L2.Client.BalanceAt(ctx, p.fundsRecipientAddr, nil)
	require.NoError(t, err)
	require.True(t, redirectBalance.Sign() > 0, "redirect address should have received fee refunds")

	// Verify sequencer is NOT halted
	_, waiting := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx(t)
	require.False(t, waiting, "sequencer should not be re-halted after processing")
}

// TestRetryableAutoRedeemCallsFilteredAddress verifies the checkpoint-and-revert
// path when an auto-redeem CALLs a filtered address. The retryable's outer
// fields are clean so submission initially succeeds and schedules an auto-redeem.
// RedeemFilter detects the CALL to the filtered target, triggering a group
// revert of the entire submission+redeem. The delayed sequencer halts, the
// operator adds the tx hash to the onchain filter, and the submission
// re-processes with redirected beneficiary/feeRefundAddr and no auto-redeem.
func TestRetryableAutoRedeemCallsFilteredAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy two AddressFilterTest contracts: caller (clean) and target (filtered)
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	targetAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	filter := newHashedChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	retryData, err := callerABI.Pack("callTarget", targetAddr)
	require.NoError(t, err)

	_, ticketId := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, retryData,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketId, p.filtererName, p.fundsRecipientAddr)
}

// TestRetryableAutoRedeemCreatesAtFilteredAddress verifies the checkpoint-and-revert
// path when an auto-redeem CREATEs a contract at a filtered address. The group
// is reverted, the submission re-processes with redirected beneficiary and no
// auto-redeem. No contract is deployed.
func TestRetryableAutoRedeemCreatesAtFilteredAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	nonce, err := builder.L2.Client.NonceAt(ctx, callerAddr, nil)
	require.NoError(t, err)
	createAddr := crypto.CreateAddress(callerAddr, nonce)

	filter := newHashedChecker([]common.Address{createAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	retryData, err := callerABI.Pack("createContract")
	require.NoError(t, err)

	_, ticketId := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, retryData,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketId, p.filtererName, p.fundsRecipientAddr)

	// Verify no contract was created at the filtered address
	code, err := builder.L2.Client.CodeAt(ctx, createAddr, nil)
	require.NoError(t, err)
	require.Empty(t, code, "no contract should exist at filtered address after group revert")
}

// TestRetryableAutoRedeemSelfDestructsToFilteredAddress verifies the
// checkpoint-and-revert path when an auto-redeem SELFDESTRUCTs to a filtered
// beneficiary. The group is reverted, so the filtered beneficiary receives no
// ETH. The submission re-processes with redirected beneficiary and no
// auto-redeem. The retryable ticket survives.
func TestRetryableAutoRedeemSelfDestructsToFilteredAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("FilteredBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")
	filteredBeneficiary := builder.L2Info.GetAddress("FilteredBeneficiary")

	contractAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	filter := newHashedChecker([]common.Address{filteredBeneficiary})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	filteredInitial, err := builder.L2.Client.BalanceAt(ctx, filteredBeneficiary, nil)
	require.NoError(t, err)

	contractABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	retryData, err := contractABI.Pack("selfDestructTo", filteredBeneficiary)
	require.NoError(t, err)

	callValue := big.NewInt(1e16)
	_, ticketId := submitRetryableViaL1(
		t, p, "Faucet", contractAddr, callValue, cleanBeneficiary, cleanBeneficiary, retryData,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketId, p.filtererName, p.fundsRecipientAddr)

	// Verify filtered beneficiary did NOT receive ETH (group was reverted)
	filteredFinal, err := builder.L2.Client.BalanceAt(ctx, filteredBeneficiary, nil)
	require.NoError(t, err)
	require.True(t, filteredFinal.Cmp(filteredInitial) == 0,
		"filtered beneficiary should not have received funds from reverted selfdestruct redeem")
}

// TestRetryableAutoRedeemStaticCallsFilteredAddress verifies the
// checkpoint-and-revert path when an auto-redeem STATICCALLs a filtered
// address. The group is reverted, the submission re-processes with redirected
// beneficiary and no auto-redeem. The retryable ticket survives.
func TestRetryableAutoRedeemStaticCallsFilteredAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	targetAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	filter := newHashedChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	retryData, err := callerABI.Pack("staticcallTargetInTx", targetAddr)
	require.NoError(t, err)

	_, ticketId := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, retryData,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketId, p.filtererName, p.fundsRecipientAddr)
}

// TestRetryableAutoRedeemEmitsTransferToFilteredAddress verifies the
// checkpoint-and-revert path when an auto-redeem emits a Transfer event with
// a filtered address in a topic. The event filter inside RedeemFilter detects
// the filtered address, triggering a group revert. The submission re-processes
// with redirected beneficiary and no auto-redeem. The retryable ticket survives.
func TestRetryableAutoRedeemEmitsTransferToFilteredAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	selector, _, err := eventfilter.CanonicalSelectorFromEvent("Transfer(address,address,uint256)")
	require.NoError(t, err)
	rules := []eventfilter.EventRule{{
		Event:          "Transfer(address,address,uint256)",
		Selector:       selector,
		TopicAddresses: []int{1, 2},
	}}

	p, cleanup := setupRetryableFilterTest(t, ctx, true, rules)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	builder.L2Info.GenerateAccount("FilteredTarget")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")
	filteredAddr := builder.L2Info.GetAddress("FilteredTarget")

	contractAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	addrFilter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(addrFilter)

	contractABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	retryData, err := contractABI.Pack("emitTransfer", cleanBeneficiary, filteredAddr)
	require.NoError(t, err)

	_, ticketId := submitRetryableViaL1(
		t, p, "Faucet", contractAddr, common.Big0, cleanBeneficiary, cleanBeneficiary, retryData,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketId, p.filtererName, p.fundsRecipientAddr)
}

// TestManualRedeemGroupRevert verifies the FullSequencingHooks path for
// checkpoint-and-revert. A retryable is submitted via L1 with gasLimit=0 so
// no auto-redeem is scheduled and the ticket survives. Then the address
// filter is set and a manual redeem is sent as a regular L2 tx. The redeem's
// inner execution touches the filtered address, triggering a group revert via
// FullSequencingHooks.ReportGroupRevert. The manual redeem tx is dropped and
// the retryable ticket survives.
func TestManualRedeemGroupRevert(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy caller and target contracts
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	targetAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	retryData, err := callerABI.Pack("callTarget", targetAddr)
	require.NoError(t, err)

	// Submit retryable with gasLimit=0 so no auto-redeem is scheduled.
	// The ticket survives for later manual redemption.
	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	maxSubmissionCost := big.NewInt(1e16)
	maxFeePerGas := big.NewInt(l2pricing.InitialBaseFeeWei * 2)
	l1opts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1opts.Value = deposit
	l1tx, err := p.delayedInbox.CreateRetryableTicket(
		&l1opts,
		callerAddr,
		common.Big0,
		maxSubmissionCost,
		cleanBeneficiary,
		cleanBeneficiary,
		common.Big0, // gasLimit=0: no auto-redeem
		maxFeePerGas,
		retryData,
	)
	require.NoError(t, err)
	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	require.NoError(t, err)

	l2Tx := p.lookupL2Tx(l1Receipt)
	ticketId := l2Tx.Hash()
	advanceL1ForDelayed(t, ctx, builder)

	// Wait for submission receipt (successful - outer fields are clean)
	receipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status,
		"retryable submission should succeed")

	// Verify retryable ticket exists (no auto-redeem was scheduled)
	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketId)
	require.NoError(t, err, "retryable ticket should exist")

	// Record target contract balance before enabling filter
	targetBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, targetAddr, nil)
	require.NoError(t, err)

	// NOW set address filter to include the target
	filter := newHashedChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Send manual redeem via L2 tx (goes through FullSequencingHooks).
	// The redeem's inner execution calls targetAddr which is now filtered.
	// Group revert fires: FullSequencingHooks.ReportGroupRevert replaces
	// txErrors[last], causing the sequencer to return an error.
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	_, err = arbRetryable.Redeem(&redeemOpts, ticketId)
	require.ErrorContains(t, err, "cascading redeem filtered",
		"manual redeem should fail with cascading redeem filter error")

	// Retryable ticket should STILL exist (submission was in a previous block,
	// unaffected by the group revert of the manual redeem)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketId)
	require.NoError(t, err, "retryable ticket should survive manual redeem group revert")

	// Target contract state should be unchanged (group revert rolled back all effects)
	targetBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, targetAddr, nil)
	require.NoError(t, err)
	require.Equal(t, targetBalanceBefore, targetBalanceAfter,
		"target contract balance should be unchanged after group revert")

	// Clear filter and do a successful manual redeem to verify numTries was
	// rolled back. If IncrementNumTries had leaked, SequenceNum would be 1.
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(nil)
	redeemOpts2 := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	redeemTx, err := arbRetryable.Redeem(&redeemOpts2, ticketId)
	require.NoError(t, err)
	redeemReceipt, err := builder.L2.EnsureTxSucceeded(redeemTx)
	require.NoError(t, err)

	arbRetryableFilterer, err := precompilesgen.NewArbRetryableTxFilterer(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	foundEvent := false
	for _, log := range redeemReceipt.Logs {
		event, err := arbRetryableFilterer.ParseRedeemScheduled(*log)
		if err != nil {
			continue
		}
		require.Equal(t, uint64(0), event.SequenceNum,
			"numTries should be 0: IncrementNumTries from the reverted manual redeem was rolled back")
		foundEvent = true
		break
	}
	require.True(t, foundEvent, "successful redeem should emit RedeemScheduled event")
}

// TestDelayedManualRedeemGroupRevert exercises the path where a signed L2 tx
// sent via the delayed inbox calls ArbRetryableTx.redeem(), and the redeem's
// inner execution touches a filtered address. Unlike cascading-redeem tests
// where ticketId == the originating delayed tx hash, here the L2 tx hash that
// wraps the redeem differs from the ticketId. The group revert fires with the
// L2 tx hash (NOT the ticketId), so the delayed sequencer halts on l2TxHash.
func TestDelayedManualRedeemGroupRevert(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("ManualRedeemer")
	builder.L2.TransferBalance(t, "Owner", "ManualRedeemer", big.NewInt(1e18), builder.L2Info)
	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy caller and target contracts
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTargetAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	retryData, err := callerABI.Pack("callTarget", filteredTargetAddr)
	require.NoError(t, err)

	// Phase 2: Submit clean retryable (no address filter yet) with gasLimit=0
	// so no auto-redeem is scheduled. The ticket survives for later manual redemption.
	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	maxSubmissionCost := big.NewInt(1e16)
	maxFeePerGas := big.NewInt(l2pricing.InitialBaseFeeWei * 2)
	l1opts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1opts.Value = deposit
	l1tx, err := p.delayedInbox.CreateRetryableTicket(
		&l1opts,
		callerAddr,
		common.Big0,
		maxSubmissionCost,
		cleanBeneficiary,
		cleanBeneficiary,
		common.Big0, // gasLimit=0: no auto-redeem
		maxFeePerGas,
		retryData,
	)
	require.NoError(t, err)
	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	require.NoError(t, err)

	l2Tx := p.lookupL2Tx(l1Receipt)
	ticketId := l2Tx.Hash()
	advanceL1ForDelayed(t, ctx, builder)

	receipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status,
		"retryable submission should succeed (no filter yet)")

	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketId)
	require.NoError(t, err, "retryable ticket should exist")

	// Record target contract balance before enabling filter
	targetBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, filteredTargetAddr, nil)
	require.NoError(t, err)

	// Phase 3: Enable filter and send delayed manual redeem
	filter := newHashedChecker([]common.Address{filteredTargetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	redeemCallData, err := arbRetryableABI.Pack("redeem", ticketId)
	require.NoError(t, err)

	arbRetryableTxAddr := types.ArbRetryableTxAddress
	signedL2Tx := prepareDelayedContractCall(t, builder, "ManualRedeemer", arbRetryableTxAddr, redeemCallData)
	l2TxHash := sendDelayedTx(t, ctx, builder, signedL2Tx)
	advanceL1ForDelayed(t, ctx, builder)

	// Phase 4: Verify group revert fires on L2 tx hash (NOT ticketId)
	require.NotEqual(t, ticketId, l2TxHash,
		"L2 tx hash must differ from ticketId")
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{l2TxHash}, 10*time.Second)

	// Phase 5: Resolve and verify
	addTxHashToOnChainFilter(t, ctx, builder, l2TxHash, p.filtererName)
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)
	advanceL1ForDelayed(t, ctx, builder)

	redeemReceipt, err := WaitForTx(ctx, builder.L2.Client, l2TxHash, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, redeemReceipt.Status,
		"delayed manual redeem should fail (filtered)")

	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketId)
	require.NoError(t, err, "retryable ticket should still exist")

	beneficiary, err := arbRetryable.GetBeneficiary(&bind.CallOpts{}, ticketId)
	require.NoError(t, err)
	require.Equal(t, cleanBeneficiary, beneficiary,
		"beneficiary should be unchanged (submission was in a prior block)")

	targetBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, filteredTargetAddr, nil)
	require.NoError(t, err)
	require.Equal(t, targetBalanceBefore, targetBalanceAfter,
		"filtered target should be untouched")

	// Phase 6: Verify numTries rollback
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(nil)

	builder.L2Info.GenerateAccount("CleanRedeemer")
	builder.L2.TransferBalance(t, "Owner", "CleanRedeemer", big.NewInt(1e18), builder.L2Info)
	cleanRedeemOpts := builder.L2Info.GetDefaultTransactOpts("CleanRedeemer", ctx)
	cleanRedeemTx, err := arbRetryable.Redeem(&cleanRedeemOpts, ticketId)
	require.NoError(t, err)
	cleanRedeemReceipt, err := builder.L2.EnsureTxSucceeded(cleanRedeemTx)
	require.NoError(t, err)

	arbRetryableFilterer, err := precompilesgen.NewArbRetryableTxFilterer(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	foundEvent := false
	for _, log := range cleanRedeemReceipt.Logs {
		event, err := arbRetryableFilterer.ParseRedeemScheduled(*log)
		if err != nil {
			continue
		}
		require.Equal(t, uint64(0), event.SequenceNum,
			"numTries should be 0: IncrementNumTries from the reverted delayed redeem was rolled back")
		foundEvent = true
		break
	}
	require.True(t, foundEvent, "successful redeem should emit RedeemScheduled event")
}

// TestRetryableGroupRevertDoesNotAffectCleanRetryable verifies that a clean
// retryable processed before a dirty one is unaffected by the dirty one's
// group revert. The two retryables arrive as separate delayed messages and
// are processed in separate blocks (via separate ProduceBlockAdvanced calls),
// so this tests sequential-block safety: the clean retryable's block is fully
// committed before the dirty retryable's block starts processing. The dirty
// group revert cannot affect a prior committed block.
func TestRetryableGroupRevertDoesNotAffectCleanRetryable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy caller, clean target (not filtered), and dirty target (filtered)
	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	cleanTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	dirtyTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	filter := newHashedChecker([]common.Address{dirtyTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)

	// Clean retryable: callTarget(cleanTarget) - no filtered address touched
	cleanRetryData, err := callerABI.Pack("callTarget", cleanTarget)
	require.NoError(t, err)
	_, cleanTicketId := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0,
		cleanBeneficiary, cleanBeneficiary, cleanRetryData,
	)

	// Dirty retryable: callTarget(dirtyTarget) - touches filtered address
	dirtyRetryData, err := callerABI.Pack("callTarget", dirtyTarget)
	require.NoError(t, err)
	_, dirtyTicketId := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0,
		cleanBeneficiary, cleanBeneficiary, dirtyRetryData,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// Clean retryable should succeed (processed first, group finalized before dirty)
	cleanReceipt, err := WaitForTx(ctx, builder.L2.Client, cleanTicketId, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, cleanReceipt.Status,
		"clean retryable submission should succeed")

	// Clean retryable's auto-redeem should have succeeded (ticket deleted)
	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, cleanTicketId)
	require.Error(t, err, "clean retryable should be deleted after successful auto-redeem")

	// Dirty retryable triggers cascading revert and halt/resume flow
	verifyCascadingRedeemFiltered(t, ctx, builder, dirtyTicketId,
		p.filtererName, p.fundsRecipientAddr)
}

// TestSequentialRetryableGroupReverts verifies that two dirty retryables
// submitted in sequence each trigger their own group revert independently.
// The first retryable halts the delayed sequencer, gets resolved, and then
// the second retryable triggers another halt/resolve cycle.
func TestSequentialRetryableGroupReverts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget1, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget2, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	filter := newHashedChecker([]common.Address{filteredTarget1, filteredTarget2})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)

	retryData1, err := callerABI.Pack("callTarget", filteredTarget1)
	require.NoError(t, err)
	_, ticketId1 := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0,
		cleanBeneficiary, cleanBeneficiary, retryData1,
	)

	retryData2, err := callerABI.Pack("callTarget", filteredTarget2)
	require.NoError(t, err)
	_, ticketId2 := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0,
		cleanBeneficiary, cleanBeneficiary, retryData2,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// First retryable: group revert -> halt -> resolve
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{ticketId1}, 10*time.Second)
	addTxHashToOnChainFilter(t, ctx, builder, ticketId1, p.filtererName)

	// After resolving ticketId1, the sequencer briefly resumes, processes
	// ticketId1 as filtered (redirected beneficiary, no auto-redeem), then immediately encounters ticketId2
	// and halts again. We skip waitForDelayedSequencerResume here because
	// the resume-to-halt transition is too fast for polling to catch.
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{ticketId2}, 10*time.Second)
	addTxHashToOnChainFilter(t, ctx, builder, ticketId2, p.filtererName)
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)
	advanceL1ForDelayed(t, ctx, builder)

	receipt1, err := WaitForTx(ctx, builder.L2.Client, ticketId1, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt1.Status)

	receipt2, err := WaitForTx(ctx, builder.L2.Client, ticketId2, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt2.Status)

	// Both retryables exist with redirected beneficiary
	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)

	beneficiary1, err := arbRetryable.GetBeneficiary(&bind.CallOpts{}, ticketId1)
	require.NoError(t, err)
	require.Equal(t, p.fundsRecipientAddr, beneficiary1)

	beneficiary2, err := arbRetryable.GetBeneficiary(&bind.CallOpts{}, ticketId2)
	require.NoError(t, err)
	require.Equal(t, p.fundsRecipientAddr, beneficiary2)
}

// TestRetryableGroupRevertSkipFinaliseSafety verifies that state changes from
// tentative execution (before group revert) are fully rolled back. The auto-
// redeem calls staticcallTargetInTx which increments the contract's "dummy"
// storage variable before STATICCALLing the filtered target. After the group
// revert and filtered re-processing, the dummy counter should be unchanged,
// proving that the tentative auto-redeem's storage writes were rolled back.
//
// Note: the dummy++ assertion specifically verifies per-redeem snapshot
// rollback (RevertToSnapshot). The skipFinalise flag's correctness is
// implicitly tested by verifyCascadingRedeemFiltered succeeding: if
// skipFinalise were broken, the tentative retryable creation would leak
// into pending storage via Finalise(), and the filtered re-processing's
// CreateRetryable would produce incorrect state.
func TestRetryableGroupRevertSkipFinaliseSafety(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	callerAddr, callerContract := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Record initial dummy counter value
	dummyBefore, err := callerContract.Dummy(&bind.CallOpts{})
	require.NoError(t, err)
	filteredInitialBalance, err := builder.L2.Client.BalanceAt(ctx, filteredTarget, nil)
	require.NoError(t, err)

	// Use staticcallTargetInTx: it does dummy++ then STATICCALL(target).
	// The dummy++ modifies storage, and the STATICCALL touches the filtered address.
	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	retryData, err := callerABI.Pack("staticcallTargetInTx", filteredTarget)
	require.NoError(t, err)

	_, ticketId := submitRetryableViaL1(
		t, p, "Faucet", callerAddr, common.Big0,
		cleanBeneficiary, cleanBeneficiary, retryData,
	)

	advanceL1ForDelayed(t, ctx, builder)
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketId,
		p.filtererName, p.fundsRecipientAddr)

	// Verify dummy counter was NOT incremented (tentative auto-redeem's
	// storage write was rolled back by group revert)
	dummyAfter, err := callerContract.Dummy(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, dummyBefore, dummyAfter,
		"dummy counter should be unchanged after group revert (tentative storage writes rolled back)")

	// Verify filtered target was not touched
	filteredFinalBalance, err := builder.L2.Client.BalanceAt(ctx, filteredTarget, nil)
	require.NoError(t, err)
	require.Equal(t, filteredInitialBalance, filteredFinalBalance,
		"filtered target balance should be unchanged after group revert")
}

// TestRetryableGroupRevertWithChainedRedeems verifies that the group revert
// correctly handles chained redeems: retryable A's auto-redeem calls
// ArbRetryableTx.redeem(ticketB), which schedules redeem-B. Redeem-B's
// inner execution touches a filtered address. The entire group (A's
// submission + A's redeem + B's redeem) is reverted. The delayed sequencer
// halts on A's ticketId, and after resolution, A re-processes with redirected
// beneficiary and no auto-redeem. B's ticket remains intact.
func TestRetryableGroupRevertWithChainedRedeems(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p, cleanup := setupRetryableFilterTest(t, ctx, true, nil)
	defer cleanup()

	builder := p.builder

	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Deploy contract for B's inner execution target
	innerTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	filter := newHashedChecker([]common.Address{filteredTarget})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)

	// Submit retryable B first with gasLimit=0 (no auto-redeem).
	// B's inner execution calls filteredTarget.
	bRetryData, err := callerABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	maxSubmissionCost := big.NewInt(1e16)
	maxFeePerGas := big.NewInt(l2pricing.InitialBaseFeeWei * 2)

	l1opts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1opts.Value = deposit
	l1txB, err := p.delayedInbox.CreateRetryableTicket(
		&l1opts,
		innerTarget,
		common.Big0,
		maxSubmissionCost,
		cleanBeneficiary,
		cleanBeneficiary,
		common.Big0, // gasLimit=0: no auto-redeem
		maxFeePerGas,
		bRetryData,
	)
	require.NoError(t, err)
	l1ReceiptB, err := builder.L1.EnsureTxSucceeded(l1txB)
	require.NoError(t, err)

	l2TxB := p.lookupL2Tx(l1ReceiptB)
	ticketIdB := l2TxB.Hash()

	// Process B's submission
	advanceL1ForDelayed(t, ctx, builder)
	receiptB, err := WaitForTx(ctx, builder.L2.Client, ticketIdB, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receiptB.Status,
		"retryable B submission should succeed")

	// Verify B's ticket exists
	arbRetryable, err := precompilesgen.NewArbRetryableTx(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketIdB)
	require.NoError(t, err, "retryable B ticket should exist")

	// Submit retryable A whose inner execution calls ArbRetryableTx.redeem(ticketB).
	// A's auto-redeem will chain into B's redeem which touches the filtered address.
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)
	aRetryData, err := arbRetryableABI.Pack("redeem", ticketIdB)
	require.NoError(t, err)

	arbRetryableTxAddr := common.HexToAddress("6e")
	_, ticketIdA := submitRetryableViaL1(
		t, p, "Faucet", arbRetryableTxAddr, common.Big0,
		cleanBeneficiary, cleanBeneficiary, aRetryData,
	)

	advanceL1ForDelayed(t, ctx, builder)

	// A's auto-redeem chains into B's redeem which touches filtered address.
	// Group revert fires for A's group, delayed sequencer halts on A's ticketId.
	verifyCascadingRedeemFiltered(t, ctx, builder, ticketIdA,
		p.filtererName, p.fundsRecipientAddr)

	// B's ticket should STILL exist (its submission was in a prior block,
	// and B's chained redeem was rolled back along with A's group)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketIdB)
	require.NoError(t, err, "retryable B ticket should survive the chained group revert")

	// Verify B's numTries is still 0: the chained redeem called
	// IncrementNumTries on B, but the group revert rolled it back.
	// Clear filter and do a successful manual redeem of B to check.
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(nil)
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	redeemTx, err := arbRetryable.Redeem(&redeemOpts, ticketIdB)
	require.NoError(t, err)
	redeemReceipt, err := builder.L2.EnsureTxSucceeded(redeemTx)
	require.NoError(t, err)

	arbRetryableFilterer, err := precompilesgen.NewArbRetryableTxFilterer(
		common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	foundEvent := false
	for _, log := range redeemReceipt.Logs {
		event, err := arbRetryableFilterer.ParseRedeemScheduled(*log)
		if err != nil {
			continue
		}
		require.Equal(t, uint64(0), event.SequenceNum,
			"B's numTries should be 0: IncrementNumTries from the chained redeem was rolled back")
		foundEvent = true
		break
	}
	require.True(t, foundEvent, "successful redeem of B should emit RedeemScheduled event")
}

// TestDelayedMessageFilterCatchesEventFilter verifies that the delayed
// message PostTxFilter runs the event filter. A normal delayed tx that
// emits a Transfer event naming a filtered address causes the sequencer
// to halt. After adding the tx hash to the onchain filter, the sequencer
// resumes and the tx is processed.
func TestDelayedMessageFilterCatchesEventFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	selector, _, err := eventfilter.CanonicalSelectorFromEvent("Transfer(address,address,uint256)")
	require.NoError(t, err)
	rules := []eventfilter.EventRule{{
		Event:          "Transfer(address,address,uint256)",
		Selector:       selector,
		TopicAddresses: []int{1, 2},
	}}

	arbOSInit := &params.ArbOSInit{
		TransactionFilteringEnabled: true,
	}
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithArbOSVersion(params.ArbosVersion_60).
		WithArbOSInit(arbOSInit).
		WithEventFilterRules(rules)
	builder.isSequencer = true
	builder.nodeConfig.DelayedSequencer.Enable = true
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
	builder.L2Info.GenerateAccount("FilteredTarget")
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

	senderAddr := builder.L2Info.GetAddress("Sender")
	filteredAddr := builder.L2Info.GetAddress("FilteredTarget")

	contractAddr, _ := deployAddressFilterTestContractForDelayed(t, ctx, builder)

	addrFilter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(addrFilter)

	contractABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	callData, err := contractABI.Pack("emitTransfer", senderAddr, filteredAddr)
	require.NoError(t, err)

	delayedTx := prepareDelayedContractCall(t, builder, "Sender", contractAddr, callData)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	advanceL1ForDelayed(t, ctx, builder)

	// Sequencer should halt because event filter detects the Transfer event
	// with the filtered address in a topic
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash}, 10*time.Second)

	// Add tx hash to onchain filter to allow it through
	addTxHashToOnChainFilter(t, ctx, builder, txHash, "Filterer")

	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	advanceL1ForDelayed(t, ctx, builder)

	// Tx should now be processed (as a filtered no-op with failed receipt)
	receipt, err := WaitForTx(ctx, builder.L2.Client, txHash, time.Second*10)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status,
		"filtered tx should have failed receipt status")
}

// TestFilteredArbitrumDepositTx verifies that an L1->L2 ETH deposit (ArbitrumDepositTx)
// to a filtered address is handled correctly when the tx hash is in the onchain filter.
// The deposit funds should be redirected to FilteredFundsRecipient (or networkFeeAccount
// as default fallback) instead of the filtered address.
func TestFilteredArbitrumDepositTx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("Filterer")
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	// The Faucet on L1 will be the depositor.
	// depositEth() sends to msg.sender for EOAs, so both From and To are the Faucet address.
	faucetAddr := builder.L1Info.GetAddress("Faucet")

	// Get the networkFeeAccount (default FilteredFundsRecipient fallback)
	arbOwnerPub, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, builder.L2.Client)
	require.NoError(t, err)
	networkFeeAccount, err := arbOwnerPub.GetNetworkFeeAccount(nil)
	require.NoError(t, err)

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Set up address filter to block the Faucet's address on L2
	addrFilter := newHashedChecker([]common.Address{faucetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(addrFilter)

	// Record initial balances
	faucetL2BalanceBefore, err := builder.L2.Client.BalanceAt(ctx, faucetAddr, nil)
	require.NoError(t, err)
	networkFeeBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, networkFeeAccount, nil)
	require.NoError(t, err)

	// Send ETH deposit from L1
	depositAmount := big.NewInt(1e16)
	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	require.NoError(t, err)
	txOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	txOpts.Value = depositAmount
	l1tx, err := delayedInbox.DepositEth439370b1(&txOpts)
	require.NoError(t, err)
	_, err = builder.L1.EnsureTxSucceeded(l1tx)
	require.NoError(t, err)

	advanceL1ForDelayed(t, ctx, builder)

	// Wait for delayed sequencer to halt on the filtered deposit
	var depositTxHash common.Hash
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		hashes, waiting := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx(t)
		if waiting && len(hashes) > 0 {
			depositTxHash = hashes[0]
			break
		}
		<-time.After(100 * time.Millisecond)
	}
	require.NotEqual(t, common.Hash{}, depositTxHash, "sequencer should halt on filtered deposit")

	addTxHashToOnChainFilter(t, ctx, builder, depositTxHash, "Filterer")

	waitForDelayedSequencerResume(t, ctx, builder, 30*time.Second)

	advanceL1ForDelayed(t, ctx, builder)

	receipt, err := WaitForTx(ctx, builder.L2.Client, depositTxHash, 30*time.Second)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status,
		"filtered deposit should have failed receipt status")

	// Verify the filtered address (Faucet) did NOT receive the deposit
	faucetL2BalanceAfter, err := builder.L2.Client.BalanceAt(ctx, faucetAddr, nil)
	require.NoError(t, err)
	require.Equal(t, faucetL2BalanceBefore, faucetL2BalanceAfter,
		"filtered address should NOT receive deposit funds")

	// Verify the networkFeeAccount (default FilteredFundsRecipient) received the deposit
	networkFeeBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, networkFeeAccount, nil)
	require.NoError(t, err)
	expectedMinBalance := new(big.Int).Add(networkFeeBalanceBefore, depositAmount)
	require.True(t, networkFeeBalanceAfter.Cmp(expectedMinBalance) >= 0,
		"networkFeeAccount balance should increase by at least deposit amount: before=%s, after=%s, deposit=%s",
		networkFeeBalanceBefore, networkFeeBalanceAfter, depositAmount)
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
	advanceL1ForDelayed(t, ctx, builder)

	// Verify sequencer is halted on this tx.
	// The filter catches the original L1 address via de-aliasing in PostTxFilter.
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{unsignedTx.Hash()}, 10*time.Second)

	// Verify recipient balance did NOT change (tx was not processed)
	finalBalance, err := builder.L2.Client.BalanceAt(ctx, recipientAddr, nil)
	require.NoError(t, err)
	require.Equal(t, initialBalance, finalBalance, "recipient balance should not change - sender is filtered")
}
