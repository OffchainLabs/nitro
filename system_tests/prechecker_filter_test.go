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
	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
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
// for prechecker filter testing. Node B forwards to A via IPC. If reportURL is
// non-empty, the forwarder's TxPreChecker is wired to send filtered tx reports
// to that URL.
func buildPrecheckerFilterNodes(t *testing.T, ctx context.Context, withDelayedSeq bool, reportURL string, eventRules ...eventfilter.EventRule) (builder *NodeBuilder, forwarder *TestClient, cleanup func()) {
	t.Helper()
	ipcPath := tmpPath(t, "test.ipc")

	builder = NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig.TransactionFiltering.Enable = false
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
	execConfigB.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessAlwaysCompatible
	execConfigB.Sequencer.Enable = false
	nodeConfigB.Sequencer = false
	nodeConfigB.DelayedSequencer.Enable = false
	execConfigB.Forwarder.RedisUrl = ""
	execConfigB.ForwardingTarget = ipcPath
	nodeConfigB.BatchPoster.Enable = false
	nodeConfigB.Feed.Input = *newBroadcastClientConfigTest(port)
	if len(eventRules) > 0 {
		execConfigB.TransactionFiltering.EventFilter.Rules = eventRules
	}
	if reportURL != "" {
		execConfigB.TransactionFiltering.FilteringReportRPCClient.URL = reportURL
	}

	forwarder, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig: nodeConfigB,
		execConfig: execConfigB,
	})

	var ef *eventfilter.EventFilter
	if len(eventRules) > 0 {
		var err error
		ef, err = eventfilter.NewEventFilterFromConfig(eventfilter.EventFilterConfig{Rules: eventRules})
		Require(t, err)
	}
	forwarder.ExecNode.TxPreChecker.SetTxFiltererForTest(t, forwarder.ExecNode.ExecEngine, ef)

	cleanup = func() {
		cleanupB()
		cleanupA()
	}
	return builder, forwarder, cleanup
}

// syncForwarderToHead waits until the forwarder catches up to the sequencer's
// current head, so prechecker dry-runs see the latest filter / contract state.
func syncForwarderToHead(t *testing.T, ctx context.Context, builder *NodeBuilder, forwarder *TestClient) {
	t.Helper()
	seqLatest, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	waitForForwarderSync(t, ctx, forwarder, seqLatest)
}

// buildForwarderRedeemTx builds a signed (un-sent) Redeem tx targeting the
// forwarder's RPC, so the caller can both submit it and reference tx.Hash().
func buildForwarderRedeemTx(
	t *testing.T, ctx context.Context, builder *NodeBuilder, forwarder *TestClient,
	account string, ticketId common.Hash, gasLimit uint64,
) *types.Transaction {
	t.Helper()
	arbRetryable, err := precompilesgen.NewArbRetryableTx(types.ArbRetryableTxAddress, forwarder.Client)
	Require(t, err)
	auth := builder.L2Info.GetDefaultTransactOpts(account, ctx)
	auth.GasLimit = gasLimit
	auth.NoSend = true
	tx, err := arbRetryable.Redeem(&auth, ticketId)
	Require(t, err, "building redeem tx")
	return tx
}

// precheckerSubmitRetryable submits a retryable via L1 and returns the L2 ticket id.
func precheckerSubmitRetryable(
	t *testing.T, ctx context.Context, builder *NodeBuilder,
	destAddr common.Address, calldata []byte, gasLimit *big.Int,
) common.Hash {
	t.Helper()
	p := newPrecheckerRetryableParams(t, ctx, builder)
	_, ticketId := submitRetryableViaL1WithGasLimit(
		t, p, "Faucet", destAddr, common.Big0, common.Address{}, common.Address{}, calldata, gasLimit,
	)

	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	_, err := WaitForTx(ctx, builder.L2.Client, ticketId, 30*time.Second)
	require.NoError(t, err)

	arbRetryable, err := precompilesgen.NewArbRetryableTx(types.ArbRetryableTxAddress, builder.L2.Client)
	require.NoError(t, err)
	_, err = arbRetryable.GetTimeout(&bind.CallOpts{}, ticketId)
	require.NoError(t, err, "retryable ticket %s should exist", ticketId.Hex())

	return ticketId
}

// newPrecheckerRetryableParams returns the minimum retryableFilterTestParams for submitRetryableViaL1WithGasLimit.
func newPrecheckerRetryableParams(t *testing.T, ctx context.Context, builder *NodeBuilder) *retryableFilterTestParams {
	t.Helper()
	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	require.NoError(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
	require.NoError(t, err)
	return &retryableFilterTestParams{
		builder:       builder,
		ctx:           ctx,
		delayedInbox:  delayedInbox,
		delayedBridge: delayedBridge,
	}
}

func requireFilteredAddressWithReason(t *testing.T, report *addressfilter.FilteredTxReport, addr common.Address, reason filter.FilterReasonType) *filter.FilteredAddressRecord {
	t.Helper()
	for i := range report.FilteredAddresses {
		if report.FilteredAddresses[i].Address == addr && report.FilteredAddresses[i].Reason == reason {
			return &report.FilteredAddresses[i]
		}
	}
	t.Fatalf("report should contain filtered address %s with reason %s, got %+v", addr.Hex(), reason, report.FilteredAddresses)
	return nil
}

// checkPrecheckerReportFields asserts FilteredTxReport invariants specific to the prechecker reporter.
func checkPrecheckerReportFields(t *testing.T, ctx context.Context, builder *NodeBuilder, report *addressfilter.FilteredTxReport, tx *types.Transaction) {
	t.Helper()
	CheckCommonReportFields(t, ctx, builder, report, tx)
	require.False(t, report.IsDelayed, "prechecker must not flag tx as delayed")
	require.Nil(t, report.DelayedReportData, "prechecker must not populate delayed payload")
	require.Equal(t, uint64(0), report.PositionInBlock, "prechecker has no in-block position")
	headNum, err := builder.L2.Client.BlockNumber(ctx)
	require.NoError(t, err)
	require.Equal(t, headNum+1, report.BlockNumber, "prechecker tx is not yet sequenced; report block should be head+1")
}

// TestPrecheckerFilterDirectAddress verifies the forwarder's prechecker rejects txs sent to/from a
// filtered address (Scenario 1: preTxFilter via from/to) AND emits a matching report.
func TestPrecheckerFilterDirectAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stack, externalEndpoint := SetupFilteringReport(t)
	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, false, stack.HTTPEndpoint())
	defer cleanup()

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("NormalUser")
	builder.L2.TransferBalance(t, "Owner", "NormalUser", big.NewInt(1e18), builder.L2Info)
	_, fundReceipt := builder.L2.TransferBalance(t, "Owner", "FilteredUser", big.NewInt(1e18), builder.L2Info)
	waitForForwarderSync(t, ctx, forwarder, fundReceipt.BlockNumber.Uint64())

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	forwarder.ExecNode.ExecEngine.SetAddressChecker(t, newHashedChecker([]common.Address{filteredAddr}))

	// tx TO filtered address via forwarder should be rejected and reported with ReasonTo
	txTo := builder.L2Info.PrepareTx("NormalUser", "FilteredUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := forwarder.Client.SendTransaction(ctx, txTo)
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for tx TO filtered address, got: %v", err)
	}
	builder.L2Info.GetInfoWithPrivKey("NormalUser").Nonce.Store(0)

	report := externalEndpoint.NextReport(t)
	checkPrecheckerReportFields(t, ctx, builder, report, txTo)
	rec := requireFilteredAddressWithReason(t, report, filteredAddr, filter.ReasonTo)
	require.Nil(t, rec.EventRuleMatch, "from/to filter must not carry event-rule payload")

	// tx FROM filtered address via forwarder should be rejected and reported with ReasonFrom
	txFrom := builder.L2Info.PrepareTx("FilteredUser", "NormalUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = forwarder.Client.SendTransaction(ctx, txFrom)
	if !isFilteredError(err) {
		t.Fatalf("expected filtered error for tx FROM filtered address, got: %v", err)
	}
	builder.L2Info.GetInfoWithPrivKey("FilteredUser").Nonce.Store(0)

	report = externalEndpoint.NextReport(t)
	checkPrecheckerReportFields(t, ctx, builder, report, txFrom)
	rec = requireFilteredAddressWithReason(t, report, filteredAddr, filter.ReasonFrom)
	require.Nil(t, rec.EventRuleMatch, "from/to filter must not carry event-rule payload")

	// tx between non-filtered addresses via forwarder should forward and succeed
	builder.L2Info.GenerateAccount("AnotherUser")
	_, fundReceipt = builder.L2.TransferBalance(t, "Owner", "AnotherUser", big.NewInt(1e18), builder.L2Info)
	waitForForwarderSync(t, ctx, forwarder, fundReceipt.BlockNumber.Uint64())
	txClean := builder.L2Info.PrepareTx("NormalUser", "AnotherUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = forwarder.Client.SendTransaction(ctx, txClean)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(txClean)
	Require(t, err)
}

// TestPrecheckerFilterCleanTxPasses verifies that non-filtered transactions
// pass through the forwarder's prechecker and are forwarded to the sequencer.
func TestPrecheckerFilterCleanTxPasses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, false, "")
	defer cleanup()

	builder.L2Info.GenerateAccount("User1")
	builder.L2Info.GenerateAccount("User2")
	builder.L2Info.GenerateAccount("FilteredUser")
	_, fundReceipt := builder.L2.TransferBalance(t, "Owner", "User1", big.NewInt(1e18), builder.L2Info)
	waitForForwarderSync(t, ctx, forwarder, fundReceipt.BlockNumber.Uint64())

	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	filter := newHashedChecker([]common.Address{filteredAddr})
	forwarder.ExecNode.ExecEngine.SetAddressChecker(t, filter)

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

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, false, "")
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

// TestPrecheckerFilterEvents verifies the forwarder's prechecker catches txs whose execution emits
// events referencing filtered addresses (Scenario 2: postTxFilter via EventFilter rule) AND emits a
// matching report.
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

	stack, externalEndpoint := SetupFilteringReport(t)
	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, false, stack.HTTPEndpoint(), rules...)
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

	forwarder.ExecNode.ExecEngine.SetAddressChecker(t, newHashedChecker([]common.Address{filteredAddr}))

	// Transfer to filtered address via forwarder should be rejected and reported with ReasonEventRule
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasLimit = 500_000 // skip EstimateGas, which would surface the filter before we get a tx to assert against
	auth.NoSend = true      // build-only; explicit SendTransaction below captures the filter rejection
	txFiltered, err := contractOnForwarder.EmitTransfer(&auth, auth.From, filteredAddr)
	Require(t, err, "building EmitTransfer tx")
	err = forwarder.Client.SendTransaction(ctx, txFiltered)
	if !isFilteredError(err) {
		t.Fatalf("expected event-rule filtered error, got: %v", err)
	}

	report := externalEndpoint.NextReport(t)
	checkPrecheckerReportFields(t, ctx, builder, report, txFiltered)
	rec := requireFilteredAddressWithReason(t, report, filteredAddr, filter.ReasonEventRule)
	require.NotNil(t, rec.EventRuleMatch, "event-rule reason must carry EventRuleMatch")
	require.Equal(t, "Transfer(address,address,uint256)", rec.EventRuleMatch.MatchedEvent)
	require.Equal(t, 2, rec.EventRuleMatch.MatchedTopicIndex, "filteredAddr is the `to` arg, indexed at topic[2]")
	require.NotNil(t, rec.EventRuleMatch.RawLog, "event-rule reason must carry raw log")
	require.Equal(t, contractAddr, rec.EventRuleMatch.RawLog.Address, "event must originate from emitter contract")
	require.NotEmpty(t, rec.EventRuleMatch.RawLog.Topics, "raw log topics must be set")
	require.Equal(t, selector[:], rec.EventRuleMatch.RawLog.Topics[0].Bytes()[:len(selector)], "topic[0] must be the event selector")

	// Transfer between clean addresses via forwarder should succeed
	auth = builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	txClean, err := contractOnForwarder.EmitTransfer(&auth, auth.From, cleanAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(txClean)
	Require(t, err)
}

// TestPrecheckerFilterManualRedeem verifies the forwarder's prechecker catches a manual redeem
// whose inner retryable touches a filtered contract (Scenario 4: scheduled retryable redeem;
// filtered contract surfaces via RunScheduledTxes) AND emits a matching report.
func TestPrecheckerFilterManualRedeem(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stack, externalEndpoint := SetupFilteringReport(t)
	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, true, stack.HTTPEndpoint())
	defer cleanup()

	// Deploy contract through sequencer as retryable destination
	contractAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)

	// Submit retryable with invalid calldata so auto-redeem fails
	invalidCalldata := []byte{0xde, 0xad, 0xbe, 0xef}
	ticketId := precheckerSubmitRetryable(t, ctx, builder, contractAddr, invalidCalldata, big.NewInt(100000))

	syncForwarderToHead(t, ctx, builder, forwarder)

	forwarder.ExecNode.ExecEngine.SetAddressChecker(t, newHashedChecker([]common.Address{contractAddr}))

	// Build redeem tx and send through forwarder -- prechecker should reject
	redeemTx := buildForwarderRedeemTx(t, ctx, builder, forwarder, "Redeemer", ticketId, 1_000_000)

	err := forwarder.Client.SendTransaction(ctx, redeemTx)
	if !isFilteredError(err) {
		t.Fatalf("expected prechecker to reject manual redeem touching filtered address, got: %v", err)
	}

	report := externalEndpoint.NextReport(t)
	checkPrecheckerReportFields(t, ctx, builder, report, redeemTx)
	// Outer tx targets ArbRetryableTx; the scheduled inner redeem tx has tx.To = contractAddr,
	// which `touchAddresses` records as ReasonTo. The inner execution is a top-level state-transition
	// call (not an opcode), so ReasonCallTarget does not fire here.
	rec := requireFilteredAddressWithReason(t, report, contractAddr, filter.ReasonTo)
	require.Nil(t, rec.EventRuleMatch, "from/to filter must not carry event-rule payload")
}

// TestPrecheckerFilterContractTriggeredRedeem verifies that the forwarder's
// prechecker catches a redeem triggered by an intermediary contract. The user's
// outer tx targets a wrapper contract (not filtered), which internally calls
// ArbRetryableTx.redeem(). The redeem's inner execution touches the filtered
// destination contract.
func TestPrecheckerFilterContractTriggeredRedeem(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, true, "")
	defer cleanup()

	// Contract A: the retryable destination (will be filtered)
	destAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Contract B: the wrapper that will call redeemTicket()
	wrapperAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	builder.L2Info.GenerateAccount("Caller")
	builder.L2.TransferBalance(t, "Owner", "Caller", big.NewInt(1e18), builder.L2Info)

	// Submit retryable with invalid calldata so auto-redeem fails
	invalidCalldata := []byte{0xde, 0xad, 0xbe, 0xef}
	ticketId := precheckerSubmitRetryable(t, ctx, builder, destAddr, invalidCalldata, big.NewInt(100000))

	// Wait for forwarder to sync to the sequencer's latest block
	syncForwarderToHead(t, ctx, builder, forwarder)

	// Set filter on forwarder's prechecker targeting contract A
	filter := newHashedChecker([]common.Address{destAddr})
	forwarder.ExecNode.ExecEngine.SetAddressChecker(t, filter)

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

// testPrecheckerFilterCascadingRedeem tests that the prechecker's FIFO redeem
// loop catches filtered addresses at arbitrary cascade depth. The deepest ticket
// targets a neutral wrapper contract that internally CALLs filteredTarget, so the
// filtered address is only discovered during actual execution (not via the To field).
func testPrecheckerFilterCascadingRedeem(t *testing.T, depth int) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, true, "")
	defer cleanup()

	// Deploy wrapper (neutral) and filteredTarget contracts
	wrapperAddr, _ := deployAddressFilterTestContract(t, ctx, builder)
	filteredTarget, _ := deployAddressFilterTestContract(t, ctx, builder)

	// Fund redeemer early so balance syncs when we sync the forwarder later
	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)

	wrapperABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	require.NoError(t, err)
	arbRetryableABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	require.NoError(t, err)

	// Build retryable chain bottom-up.
	// ticket[0] is the deepest: targets wrapper with callTarget(filteredTarget).
	// ticket[i>0]: dest=ArbRetryableTx, data=redeem(ticket[i-1]).
	ticketIds := make([]common.Hash, depth)

	// Deepest ticket targets wrapper.callTarget(filteredTarget) — the filtered
	// address is only touched during execution, not in the To field.
	callTargetData, err := wrapperABI.Pack("callTarget", filteredTarget)
	require.NoError(t, err)
	ticketIds[0] = precheckerSubmitRetryable(
		t, ctx, builder, wrapperAddr, callTargetData, common.Big0,
	)

	// Each subsequent ticket redeems the previous one.
	for i := 1; i < depth; i++ {
		redeemData, err := arbRetryableABI.Pack("redeem", ticketIds[i-1])
		require.NoError(t, err)
		ticketIds[i] = precheckerSubmitRetryable(
			t, ctx, builder, types.ArbRetryableTxAddress, redeemData, common.Big0,
		)
	}

	topTicketId := ticketIds[depth-1]

	// Sync forwarder to sequencer's latest block
	syncForwarderToHead(t, ctx, builder, forwarder)

	// Set filter on forwarder's prechecker targeting filteredTarget
	filter := newHashedChecker([]common.Address{filteredTarget})
	forwarder.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// Manual redeem of the top ticket through forwarder — prechecker must
	// execute the full cascade including the deepest redeem to discover the
	// filtered address touched via wrapper.callTarget().
	redeemTx := buildForwarderRedeemTx(t, ctx, builder, forwarder, "Redeemer", topTicketId, 2_000_000)

	err = forwarder.Client.SendTransaction(ctx, redeemTx)
	if !isFilteredError(err) {
		t.Fatalf("expected prechecker to reject cascading redeem at depth %d, got: %v", depth, err)
	}
}

// TestPrecheckerFilterCascadingRedeemDepth2 tests A -> B -> wrapper.callTarget(filtered).
func TestPrecheckerFilterCascadingRedeemDepth2(t *testing.T) {
	testPrecheckerFilterCascadingRedeem(t, 2)
}

// TestPrecheckerFilterCascadingRedeemDepth3 tests A -> B -> C -> wrapper.callTarget(filtered).
func TestPrecheckerFilterCascadingRedeemDepth3(t *testing.T) {
	testPrecheckerFilterCascadingRedeem(t, 3)
}

// TestPrecheckerFilterCascadingRedeemDepth4 tests A -> B -> C -> D -> wrapper.callTarget(filtered).
func TestPrecheckerFilterCascadingRedeemDepth4(t *testing.T) {
	testPrecheckerFilterCascadingRedeem(t, 4)
}

func TestPrecheckerFilterContractCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stack, externalEndpoint := SetupFilteringReport(t)
	builder, forwarder, cleanup := buildPrecheckerFilterNodes(t, ctx, false, stack.HTTPEndpoint())
	defer cleanup()

	wrapperAddr, _ := deployAddressFilterTestContract(t, ctx, builder)
	filteredTargetAddr, _ := deployAddressFilterTestContract(t, ctx, builder)

	syncForwarderToHead(t, ctx, builder, forwarder)

	forwarder.ExecNode.ExecEngine.SetAddressChecker(t, newHashedChecker([]common.Address{filteredTargetAddr}))

	wrapperOnForwarder, err := localgen.NewAddressFilterTest(wrapperAddr, forwarder.Client)
	Require(t, err)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasLimit = 500_000 // skip EstimateGas, which would surface the filter via the forwarder's prechecker before we get a tx to assert against
	auth.NoSend = true      // build-only; explicit SendTransaction below captures the filter rejection
	tx, err := wrapperOnForwarder.CallTarget(&auth, filteredTargetAddr)
	Require(t, err, "building CallTarget tx")
	err = forwarder.Client.SendTransaction(ctx, tx)
	if !isFilteredError(err) {
		t.Fatalf("expected post-execution filtered error, got: %v", err)
	}

	report := externalEndpoint.NextReport(t)
	checkPrecheckerReportFields(t, ctx, builder, report, tx)
	rec := requireFilteredAddressWithReason(t, report, filteredTargetAddr, filter.ReasonCallTarget)
	require.Nil(t, rec.EventRuleMatch, "call-target reason must not carry EventRuleMatch")
}
