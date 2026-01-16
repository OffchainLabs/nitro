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
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/txfilter"
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

// advanceAndWaitForDelayed advances L1 blocks and waits for delayed message processing.
func advanceAndWaitForDelayed(t *testing.T, ctx context.Context, builder *NodeBuilder) {
	t.Helper()
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	<-time.After(time.Second * 2)
}

// waitForDelayedSequencerHalt waits until the delayed sequencer is halted on a filtered tx.
// Returns the tx hash being waited on.
func waitForDelayedSequencerHalt(t *testing.T, ctx context.Context, builder *NodeBuilder, timeout time.Duration) common.Hash {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if builder.L2.ConsensusNode.DelayedSequencer == nil {
			t.Fatal("DelayedSequencer is nil")
		}
		hash := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx()
		if hash != nil {
			return *hash
		}
		<-time.After(100 * time.Millisecond)
	}
	t.Fatal("timeout waiting for delayed sequencer to halt")
	return common.Hash{}
}

// waitForDelayedSequencerResume waits until the delayed sequencer is no longer halted.
func waitForDelayedSequencerResume(t *testing.T, ctx context.Context, builder *NodeBuilder, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if builder.L2.ConsensusNode.DelayedSequencer == nil {
			t.Fatal("DelayedSequencer is nil")
		}
		hash := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx()
		if hash == nil {
			return
		}
		<-time.After(100 * time.Millisecond)
	}
	t.Fatal("timeout waiting for delayed sequencer to resume")
}

// addTxHashToOnChainFilter adds a tx hash to the on-chain filter via the precompile.
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare and send delayed tx TO filtered address
	delayedTx := builder.L2Info.PrepareTx("Sender", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted on this tx
	haltedOnTx := waitForDelayedSequencerHalt(t, ctx, builder, 10*time.Second)
	require.Equal(t, txHash, haltedOnTx, "sequencer should be halted on the filtered tx")

	// Verify balance did NOT change (block not created)
	finalBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, initialBalance, finalBalance, "filtered address balance should not change")
}

// TestDelayedMessageFilterBypass verifies that adding tx hash to on-chain filter allows tx to proceed.
func TestDelayedMessageFilterBypass(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare and send delayed tx TO filtered address
	delayedTx := builder.L2Info.PrepareTx("Sender", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Verify sequencer is halted
	haltedOnTx := waitForDelayedSequencerHalt(t, ctx, builder, 10*time.Second)
	require.Equal(t, txHash, haltedOnTx)

	// Verify balance did NOT change yet
	midBalance, err := builder.L2.Client.BalanceAt(ctx, filteredAddr, nil)
	require.NoError(t, err)
	require.Equal(t, initialBalance, midBalance, "balance should not change while halted")

	// Add tx hash to on-chain filter to bypass
	addTxHashToOnChainFilter(t, ctx, builder, txHash, "Filterer")

	// Get sender's initial nonce and balance before bypass
	senderAddr := builder.L2Info.GetAddress("Sender")
	senderNonceBefore, err := builder.L2.Client.NonceAt(ctx, senderAddr, nil)
	require.NoError(t, err)
	senderBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, senderAddr, nil)
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

// TestDelayedMessageFilterBlocksSubsequent verifies that messages behind filtered one are blocked.
func TestDelayedMessageFilterBlocksSubsequent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser1")
	builder.L2Info.GenerateAccount("NormalUser2")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
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

	// Grant Filterer the transaction filterer role
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Set up address filter to block FilteredUser
	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredAddr})
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
	haltedOnTx := waitForDelayedSequencerHalt(t, ctx, builder, 10*time.Second)
	require.Equal(t, txHash1, haltedOnTx, "sequencer should be halted on the first (filtered) tx")

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

	// Get sender balance before bypass to verify gas consumption later
	senderBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, senderAddr, nil)
	require.NoError(t, err)

	// Add tx hash to on-chain filter to bypass
	addTxHashToOnChainFilter(t, ctx, builder, txHash1, "Filterer")

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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(filter)

	// Prepare and send delayed tx TO normal (non-filtered) address
	delayedTx := builder.L2Info.PrepareTx("Sender", "NormalUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	sendDelayedTx(t, ctx, builder, delayedTx)

	// Advance L1 to trigger delayed message processing
	advanceAndWaitForDelayed(t, ctx, builder)

	// Give some time for processing
	<-time.After(time.Second)

	// Verify sequencer is NOT halted
	hash := builder.L2.ConsensusNode.DelayedSequencer.WaitingForFilteredTx()
	require.Nil(t, hash, "sequencer should not be halted for non-filtered address")

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
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{targetAddr})
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
	haltedOnTx := waitForDelayedSequencerHalt(t, ctx, builder, 10*time.Second)
	require.Equal(t, txHash, haltedOnTx, "sequencer should be halted on the filtered CALL tx")

	// Add tx hash to on-chain filter to bypass
	addTxHashToOnChainFilter(t, ctx, builder, txHash, "Filterer")

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
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{targetAddr})
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
	haltedOnTx := waitForDelayedSequencerHalt(t, ctx, builder, 10*time.Second)
	require.Equal(t, txHash, haltedOnTx, "sequencer should be halted on the filtered STATICCALL tx")

	// Add tx hash to on-chain filter to bypass
	addTxHashToOnChainFilter(t, ctx, builder, txHash, "Filterer")

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
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{createAddr})
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
	haltedOnTx := waitForDelayedSequencerHalt(t, ctx, builder, 10*time.Second)
	require.Equal(t, txHash, haltedOnTx, "sequencer should be halted on the filtered CREATE tx")

	// Add tx hash to on-chain filter to bypass
	addTxHashToOnChainFilter(t, ctx, builder, txHash, "Filterer")

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
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{create2Addr})
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
	haltedOnTx := waitForDelayedSequencerHalt(t, ctx, builder, 10*time.Second)
	require.Equal(t, txHash, haltedOnTx, "sequencer should be halted on the filtered CREATE2 tx")

	// Add tx hash to on-chain filter to bypass
	addTxHashToOnChainFilter(t, ctx, builder, txHash, "Filterer")

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
	cleanup := builder.Build(t)
	defer cleanup()

	// Create accounts
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")
	builder.L2Info.GenerateAccount("FilteredBeneficiary")
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
	filter := txfilter.NewStaticAsyncChecker([]common.Address{filteredBeneficiary})
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
	haltedOnTx := waitForDelayedSequencerHalt(t, ctx, builder, 10*time.Second)
	require.Equal(t, txHash, haltedOnTx, "sequencer should be halted on the filtered SELFDESTRUCT tx")

	// Add tx hash to on-chain filter to bypass
	addTxHashToOnChainFilter(t, ctx, builder, txHash, "Filterer")

	// Wait for delayed sequencer to resume
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)

	// Verify the tx was processed
	receipt, err := builder.L2.Client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "bypassed tx should have failed receipt status")
}
