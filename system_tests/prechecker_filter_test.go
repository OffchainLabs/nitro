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
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

// These tests use a two-node setup: a sequencer (node A) and a forwarder
// (node B). The forwarder's TxPreChecker has address filtering enabled, but
// the sequencer has NO filtering configured. This proves filtering structurally:
// rejections can only come from the forwarder's prechecker dry-run. Clean txs
// forwarded through B reach A and are sequenced normally.

// waitForForwarderSync polls the forwarder until its latest block number
// reaches targetBlock. Unlike WaitForTx, this doesn't depend on the tx
// indexer, which can be slow on freshly-synced nodes.
func waitForForwarderSync(t *testing.T, ctx context.Context, forwarder *TestClient, targetBlock uint64) {
	t.Helper()
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		header, err := forwarder.Client.HeaderByNumber(timeoutCtx, nil)
		if err == nil && header.Number.Uint64() >= targetBlock {
			return
		}
		select {
		case <-timeoutCtx.Done():
			require.NoError(t, timeoutCtx.Err(), "forwarder did not reach block %d within timeout", targetBlock)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// buildPrecheckerFilterNodes creates a sequencer node A and a forwarder node B
// for prechecker filter testing. Node B forwards to A via IPC.
func buildPrecheckerFilterNodes(t *testing.T, ctx context.Context, withDelayedSeq bool) (builder *NodeBuilder, forwarder *TestClient, cleanup func()) {
	t.Helper()
	ipcPath := tmpPath(t, "test.ipc")

	builder = NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	builder.l2StackConfig.IPCPath = ipcPath
	if withDelayedSeq {
		builder.nodeConfig.DelayedSequencer.Enable = true
		builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1
	} else {
		builder.nodeConfig.BatchPoster.Enable = false
	}
	cleanupA := builder.Build(t)

	port := testhelpers.AddrTCPPort(builder.L2.ConsensusNode.BroadcastServer.ListenerAddr(), t)

	nodeConfigB := arbnode.ConfigDefaultL1Test()
	execConfigB := ExecConfigDefaultTest(t, env.GetTestStateScheme())
	execConfigB.Sequencer.Enable = false
	nodeConfigB.Sequencer = false
	nodeConfigB.DelayedSequencer.Enable = false
	execConfigB.Forwarder.RedisUrl = ""
	execConfigB.ForwardingTarget = ipcPath
	nodeConfigB.BatchPoster.Enable = false
	nodeConfigB.Feed.Input = *newBroadcastClientConfigTest(port)

	forwarder, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig: nodeConfigB,
		execConfig: execConfigB,
	})

	cleanup = func() {
		cleanupB()
		cleanupA()
	}
	return builder, forwarder, cleanup
}

// TestPrecheckerFilterDirectAddress verifies that the forwarder's prechecker
// dry-run filtering catches transactions sent to/from a filtered address.
func TestPrecheckerFilterDirectAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, false)
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	_, fundReceipt := builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)
	waitForForwarderSync(t, ctx, forwarder, fundReceipt.BlockNumber.Uint64())

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	filter := newHashedChecker([]common.Address{filteredAddr})
	forwarder.ExecNode.TxPreChecker.SetAddressChecker(filter)

	// tx TO filtered address via forwarder should be rejected
	tx := builder.L2Info.PrepareTx("NormalUser", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := forwarder.Client.SendTransaction(ctx, tx)
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for tx TO filtered address, got: %v", err)
	}
	builder.L2Info.GetInfoWithPrivKey("NormalUser").Nonce.Store(0)

	// tx FROM filtered address via forwarder should be rejected
	tx = builder.L2Info.PrepareTx("FilteredUser", "NormalUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = forwarder.Client.SendTransaction(ctx, tx)
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for tx FROM filtered address, got: %v", err)
	}
	builder.L2Info.GetInfoWithPrivKey("FilteredUser").Nonce.Store(0)

	// tx between non-filtered addresses via forwarder should forward and succeed
	builder.L2Info.GenerateAccount("AnotherUser")
	_, fundReceipt = builder.L2.TransferBalance(t, "Owner", "AnotherUser", big.NewInt(1e18), builder.L2Info)
	waitForForwarderSync(t, ctx, forwarder, fundReceipt.BlockNumber.Uint64())
	tx = builder.L2Info.PrepareTx("NormalUser", "AnotherUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = forwarder.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

// TestPrecheckerFilterCleanTxPasses verifies that non-filtered transactions
// pass through the forwarder's prechecker and are forwarded to the sequencer.
func TestPrecheckerFilterCleanTxPasses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, false)
	defer cleanup()

	builder.L2Info.GenerateAccount("User1")
	builder.L2Info.GenerateAccount("User2")
	builder.L2Info.GenerateAccount("FilteredUser")
	_, fundReceipt := builder.L2.TransferBalance(t, "Owner", "User1", big.NewInt(1e18), builder.L2Info)
	waitForForwarderSync(t, ctx, forwarder, fundReceipt.BlockNumber.Uint64())

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	filter := newHashedChecker([]common.Address{filteredAddr})
	forwarder.ExecNode.TxPreChecker.SetAddressChecker(filter)

	tx := builder.L2Info.PrepareTx("User1", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := forwarder.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

// TestPrecheckerFilterDisabled verifies that all transactions pass when no
// address checker is set on the forwarder's prechecker.
func TestPrecheckerFilterDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, false)
	defer cleanup()

	builder.L2Info.GenerateAccount("User1")
	builder.L2Info.GenerateAccount("User2")
	_, fundReceipt := builder.L2.TransferBalance(t, "Owner", "User1", big.NewInt(1e18), builder.L2Info)
	waitForForwarderSync(t, ctx, forwarder, fundReceipt.BlockNumber.Uint64())

	// No address checker set on forwarder -- all txs should pass
	tx := builder.L2Info.PrepareTx("User1", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := forwarder.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

// TestPrecheckerFilterEvents verifies that the forwarder's prechecker catches
// transactions whose execution emits events referencing filtered addresses.
func TestPrecheckerFilterEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	selector, _, err := eventfilter.CanonicalSelectorFromEvent("Transfer(address,address,uint256)")
	Require(t, err)

	rules := []eventfilter.EventRule{
		{
			Event:          "Transfer(address,address,uint256)",
			Selector:       selector,
			TopicAddresses: []int{1, 2},
		},
	}

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, false)
	defer cleanup()

	// Deploy contract through sequencer and wait for forwarder to sync
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	contractAddr, deployTx, _, err := localgen.DeployAddressFilterTest(&auth, builder.L2.Client)
	Require(t, err)
	deployReceipt, err := builder.L2.EnsureTxSucceeded(deployTx)
	Require(t, err)
	waitForForwarderSync(t, ctx, forwarder, deployReceipt.BlockNumber.Uint64())

	// Bind contract to forwarder client
	contractOnForwarder, err := localgen.NewAddressFilterTest(contractAddr, forwarder.Client)
	Require(t, err)

	builder.L2Info.GenerateAccount("FilteredAddr")
	builder.L2Info.GenerateAccount("CleanAddr")
	filteredAddr := builder.L2Info.GetAddress("FilteredAddr")
	cleanAddr := builder.L2Info.GetAddress("CleanAddr")

	filter := newHashedChecker([]common.Address{filteredAddr})
	ef, err := eventfilter.NewEventFilter(rules)
	Require(t, err)
	forwarder.ExecNode.TxPreChecker.SetAddressChecker(filter)
	forwarder.ExecNode.TxPreChecker.SetEventFilter(ef)

	// Transfer to filtered address via forwarder should be rejected
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, err = contractOnForwarder.EmitTransfer(&auth, auth.From, filteredAddr)
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for Transfer to filtered address, got: %v", err)
	}

	// Transfer between clean addresses via forwarder should succeed
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := contractOnForwarder.EmitTransfer(&auth, auth.From, cleanAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

// TestPrecheckerFilterManualRedeem verifies that the forwarder's prechecker
// catches a manual redeem of a retryable whose inner call touches a filtered
// address. The retryable is created via L1 and processed by the sequencer.
// The forwarder syncs the state, then the redeem is sent through the forwarder
// where the prechecker dry-run detects the filtered address.
func TestPrecheckerFilterManualRedeem(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, true)
	defer cleanup()

	// Deploy contract through sequencer as retryable destination
	contractAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	require.NoError(t, err)

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	maxSubmissionCost := big.NewInt(1e16)
	gasLimit := big.NewInt(100000)
	maxFeePerGas := big.NewInt(l2pricing.InitialBaseFeeWei * 2)

	// Invalid selector so the auto-redeem reverts (no fallback on contract)
	invalidCalldata := []byte{0xde, 0xad, 0xbe, 0xef}

	l1opts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1opts.Value = deposit
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&l1opts,
		contractAddr,
		common.Big0,
		maxSubmissionCost,
		common.Address{},
		common.Address{},
		gasLimit,
		maxFeePerGas,
		invalidCalldata,
	)
	require.NoError(t, err)
	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	require.NoError(t, err)

	ticketId := lookupSubmissionTxHash(t, ctx, builder, l1Receipt)

	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	// Wait for sequencer to process the retryable
	seqReceipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, 30*time.Second)
	require.NoError(t, err)

	// Verify ticket survived the failed auto-redeem
	arbRetryable, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketId)
	require.NoError(t, err, "retryable ticket should exist after failed auto-redeem")

	// Wait for forwarder to sync past the retryable submission block
	waitForForwarderSync(t, ctx, forwarder, seqReceipt.BlockNumber.Uint64()+2)

	// Set filter on forwarder's prechecker targeting the contract
	filter := newHashedChecker([]common.Address{contractAddr})
	forwarder.ExecNode.TxPreChecker.SetAddressChecker(filter)

	// Build redeem tx and send through forwarder -- prechecker should reject
	arbRetryableOnForwarder, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), forwarder.Client)
	require.NoError(t, err)
	auth := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	auth.GasLimit = 1_000_000
	auth.NoSend = true
	redeemTx, err := arbRetryableOnForwarder.Redeem(&auth, ticketId)
	require.NoError(t, err, "building redeem tx should not error")

	err = forwarder.Client.SendTransaction(ctx, redeemTx)
	if !isFilteredError(err) {
		t.Fatalf("expected prechecker to reject manual redeem touching filtered address, got: %v", err)
	}
}

// TestPrecheckerFilterContractTriggeredRedeem verifies that the forwarder's
// prechecker catches a redeem triggered by an intermediary contract. The user's
// outer tx targets a wrapper contract (not filtered), which internally calls
// ArbRetryableTx.redeem(). The redeem's inner execution touches the filtered
// destination contract.
func TestPrecheckerFilterContractTriggeredRedeem(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, true)
	defer cleanup()

	// Contract A: the retryable destination (will be filtered)
	destAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Contract B: the wrapper that will call redeemTicket()
	wrapperAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	builder.L2Info.GenerateAccount("Caller")
	builder.L2.TransferBalance(t, "Owner", "Caller", big.NewInt(1e18), builder.L2Info)

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	require.NoError(t, err)

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	maxSubmissionCost := big.NewInt(1e16)
	gasLimit := big.NewInt(100000)
	maxFeePerGas := big.NewInt(l2pricing.InitialBaseFeeWei * 2)

	invalidCalldata := []byte{0xde, 0xad, 0xbe, 0xef}

	l1opts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1opts.Value = deposit
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&l1opts,
		destAddr,
		common.Big0,
		maxSubmissionCost,
		common.Address{},
		common.Address{},
		gasLimit,
		maxFeePerGas,
		invalidCalldata,
	)
	require.NoError(t, err)
	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	require.NoError(t, err)

	ticketId := lookupSubmissionTxHash(t, ctx, builder, l1Receipt)

	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	// Wait for sequencer to process the retryable
	seqReceipt, err := WaitForTx(ctx, builder.L2.Client, ticketId, 30*time.Second)
	require.NoError(t, err)

	// Verify ticket survived the failed auto-redeem
	arbRetryable, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketId)
	require.NoError(t, err, "retryable ticket should exist after failed auto-redeem")

	// Wait for forwarder to sync past the retryable submission block
	waitForForwarderSync(t, ctx, forwarder, seqReceipt.BlockNumber.Uint64()+2)

	// Set filter on forwarder's prechecker targeting contract A
	filter := newHashedChecker([]common.Address{destAddr})
	forwarder.ExecNode.TxPreChecker.SetAddressChecker(filter)

	// Bind wrapper contract to forwarder client and send through forwarder
	wrapperOnForwarder, err := localgen.NewAddressFilterTest(wrapperAddr, forwarder.Client)
	require.NoError(t, err)
	auth := builder.L2Info.GetDefaultTransactOpts("Caller", ctx)
	auth.GasLimit = 1_000_000
	_, err = wrapperOnForwarder.RedeemTicket(&auth, ticketId)
	if !isFilteredError(err) {
		t.Fatalf("expected prechecker to reject contract-triggered redeem touching filtered address, got: %v", err)
	}
}

// lookupSubmissionTxHash finds the ArbitrumSubmitRetryableTx hash from an L1 receipt
// by parsing the delayed message.
func lookupSubmissionTxHash(t *testing.T, ctx context.Context, builder *NodeBuilder, l1Receipt *types.Receipt) common.Hash {
	t.Helper()

	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
	require.NoError(t, err)

	messages, err := delayedBridge.LookupMessagesInRange(ctx, l1Receipt.BlockNumber, l1Receipt.BlockNumber, nil)
	require.NoError(t, err)
	require.NotEmpty(t, messages, "no delayed messages found")

	for _, message := range messages {
		if message.Message.Header.Kind != arbostypes.L1MessageType_SubmitRetryable {
			continue
		}
		txs, err := arbos.ParseL2Transactions(message.Message, chaininfo.ArbitrumDevTestChainConfig().ChainID, params.MaxDebugArbosVersionSupported)
		require.NoError(t, err)
		for _, tx := range txs {
			if tx.Type() == types.ArbitrumSubmitRetryableTxType {
				return tx.Hash()
			}
		}
	}
	t.Fatal("no retryable submission tx found in delayed messages")
	return common.Hash{}
}
