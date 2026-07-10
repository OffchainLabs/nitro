// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/transactionfeed"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func newTransactionFeedConfigTest() transactionfeed.ServerConfig {
	cfg := transactionfeed.DefaultServerConfig
	cfg.Enable = true
	cfg.Addr = "127.0.0.1"
	cfg.Port = "0"
	cfg.HandshakeTimeout = 2 * time.Second
	cfg.WriteTimeout = 2 * time.Second
	return cfg
}

func dialTransactionFeed(ctx context.Context, t *testing.T, port int) net.Conn {
	t.Helper()
	dctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	url := fmt.Sprintf("ws://127.0.0.1:%d/", port)
	conn, _, _, err := ws.Dialer{}.Dial(dctx, url)
	Require(t, err, "failed to dial transaction feed")
	return conn
}

func waitForTransactionFeedClients(t *testing.T, s *transactionfeed.Server, want int32, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s.ClientCount() >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("transaction feed never reached %d clients (have %d)", want, s.ClientCount())
}

// feedRecorder drains a transaction feed connection, recording every received message
type feedRecorder struct {
	mu       sync.Mutex
	received []*transactionfeed.TransactionFeedMessage
	errs     chan error
}

func startTransactionFeedRecorder(conn net.Conn) *feedRecorder {
	r := &feedRecorder{errs: make(chan error, 1)}
	go func() {
		for {
			data, err := wsutil.ReadServerBinary(conn)
			if err != nil {
				select {
				case r.errs <- err:
				default:
				}
				return
			}
			var msg transactionfeed.TransactionFeedMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				select {
				case r.errs <- err:
				default:
				}
				return
			}
			r.mu.Lock()
			r.received = append(r.received, &msg)
			r.mu.Unlock()
		}
	}()
	return r
}

// messagesFor returns the recorded messages for txHash in arrival order.
func (r *feedRecorder) messagesFor(txHash common.Hash) []*transactionfeed.TransactionFeedMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*transactionfeed.TransactionFeedMessage
	for _, m := range r.received {
		if strings.EqualFold(m.Transaction.TxHash, txHash.Hex()) {
			out = append(out, m)
		}
	}
	return out
}

// counts returns how many times each tx hash has been recorded.
func (r *feedRecorder) counts() map[common.Hash]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	got := make(map[common.Hash]int, len(r.received))
	for _, m := range r.received {
		got[common.HexToHash(m.Transaction.TxHash)]++
	}
	return got
}

// awaitFeedMessages waits until at least count messages for txHash have been
// recorded and returns them in arrival order.
func (r *feedRecorder) awaitFeedMessages(t *testing.T, txHash common.Hash, count int, timeout time.Duration) []*transactionfeed.TransactionFeedMessage {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if msgs := r.messagesFor(txHash); len(msgs) >= count {
			return msgs
		}
		select {
		case err := <-r.errs:
			t.Fatalf("transaction feed reader error while waiting for tx %s: %v", txHash.Hex(), err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %d message(s) for tx %s on transaction feed", count, txHash.Hex())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (r *feedRecorder) awaitFeedMessageFor(t *testing.T, txHash common.Hash, timeout time.Duration) *transactionfeed.TransactionFeedMessage {
	t.Helper()
	return r.awaitFeedMessages(t, txHash, 1, timeout)[0]
}

func (r *feedRecorder) assertNoError(t *testing.T) {
	t.Helper()
	select {
	case err := <-r.errs:
		t.Fatalf("transaction feed reader error: %v", err)
	default:
	}
}

func parseRedeemScheduledRetryHash(t *testing.T, client *ethclient.Client, receipt *types.Receipt) common.Hash {
	t.Helper()
	arbRetryableFilterer, err := precompilesgen.NewArbRetryableTxFilterer(types.ArbRetryableTxAddress, client)
	Require(t, err)
	for _, l := range receipt.Logs {
		event, err := arbRetryableFilterer.ParseRedeemScheduled(*l)
		if err != nil {
			continue
		}
		return common.Hash(event.RetryTxHash)
	}
	t.Fatalf("RedeemScheduled event not found on receipt for tx %s", receipt.TxHash.Hex())
	return common.Hash{}
}

type transactionFeedTestOpts struct {
	// withL1 builds the node with an L1 chain and exposes the delayed bridge
	// helpers (delayedInbox, lookupL2Tx) on the returned env.
	withL1 bool
	// withDelayedSequencer implies withL1; enables the delayed sequencer so
	// that L1->L2 messages drive the DelayedFilteringSequencingHooks emit
	// path in arbos/block_processor.go without ArbOS-level filtering.
	withDelayedSequencer bool
	// enableFiltering implies withDelayedSequencer plus the ArbOS-level
	// transaction-filtering machinery: ArbOS v60 and a Filterer /
	// FundsRecipient registered through ArbOwner.
	enableFiltering bool
	// feedConfig overrides the default test feed config when non-nil. Used by
	// tests that need a tighter ClientBuf (slow-consumer eviction) etc.
	feedConfig *transactionfeed.ServerConfig
}

type transactionFeedTestEnv struct {
	ctx          context.Context
	builder      *NodeBuilder
	recorder     *feedRecorder
	server       *transactionfeed.Server
	conn         net.Conn
	delayedInbox *bridgegen.Inbox
	lookupL2Tx   func(*types.Receipt) *types.Transaction
	// setupTxs are txs broadcast on the feed outside the test's own traffic
	// (filtering registration, earlier sentinels) stored so they are accounted
	// for by helpers like assertFeedExactly.
	setupTxs      []common.Hash
	sentinelCount int
	cleanup       func()
}

func (env *transactionFeedTestEnv) awaitFeedMessageFor(t *testing.T, txHash common.Hash, timeout time.Duration) *transactionfeed.TransactionFeedMessage {
	t.Helper()
	return env.recorder.awaitFeedMessageFor(t, txHash, timeout)
}

func (env *transactionFeedTestEnv) assertFeedExactly(t *testing.T, want map[common.Hash]int) {
	t.Helper()
	builder := env.builder
	env.sentinelCount++
	// We use a sentinel tx to ensure that the feed has delivered all prior txs before we check the counts.F
	sentinelAccount := fmt.Sprintf("FeedSentinel%d", env.sentinelCount)
	builder.L2Info.GenerateAccount(sentinelAccount)
	sentinelTx := builder.L2Info.PrepareTx("Owner", sentinelAccount, builder.L2Info.TransferGas, big.NewInt(1e6), nil)
	Require(t, builder.L2.Client.SendTransaction(env.ctx, sentinelTx))
	_, err := builder.L2.EnsureTxSucceeded(sentinelTx)
	Require(t, err)
	env.recorder.awaitFeedMessageFor(t, sentinelTx.Hash(), 10*time.Second)
	env.setupTxs = append(env.setupTxs, sentinelTx.Hash())

	expected := make(map[common.Hash]int, len(want)+len(env.setupTxs))
	for h, c := range want {
		expected[h] = c
	}
	for _, h := range env.setupTxs {
		expected[h]++
	}

	got := env.recorder.counts()
	for h, wantCount := range expected {
		if got[h] != wantCount {
			t.Errorf("feed delivered tx %s %d time(s), want %d", h.Hex(), got[h], wantCount)
		}
	}
	for h, gotCount := range got {
		if _, ok := expected[h]; !ok {
			t.Errorf("unexpected tx %s on feed (delivered %d time(s))", h.Hex(), gotCount)
		}
	}
	if t.Failed() {
		t.FailNow()
	}
}

func setupTransactionFeedTest(t *testing.T, ctx context.Context, opts transactionFeedTestOpts) *transactionFeedTestEnv {
	t.Helper()
	withDelayedSequencer := opts.withDelayedSequencer || opts.enableFiltering
	withL1 := opts.withL1 || withDelayedSequencer

	builderChain := NewNodeBuilder(ctx).DefaultConfig(t, withL1)
	if opts.enableFiltering {
		builderChain = builderChain.
			WithArbOSVersion(params.ArbosVersion_60).
			WithArbOSInit(&params.ArbOSInit{TransactionFilteringEnabled: true})
	}
	builder := builderChain.DontParalellise()

	if withDelayedSequencer {
		builder.isSequencer = true
		builder.nodeConfig.DelayedSequencer.Enable = true
		builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1
	}
	if opts.feedConfig != nil {
		builder.execConfig.TransactionFeed = *opts.feedConfig
	} else {
		builder.execConfig.TransactionFeed = newTransactionFeedConfigTest()
	}

	cleanup := builder.Build(t)

	rfs := builder.L2.ExecNode.TransactionFeedServer
	if rfs == nil {
		cleanup()
		t.Fatal("TransactionFeedServer was not constructed")
	}
	port := testhelpers.AddrTCPPort(rfs.ListenerAddr(), t)

	conn := dialTransactionFeed(ctx, t, port)
	recorder := startTransactionFeedRecorder(conn)
	waitForTransactionFeedClients(t, rfs, 1, 3*time.Second)

	env := &transactionFeedTestEnv{
		ctx:      ctx,
		builder:  builder,
		recorder: recorder,
		server:   rfs,
		conn:     conn,
	}
	var cleanupOnce sync.Once
	env.cleanup = func() {
		cleanupOnce.Do(func() {
			_ = conn.Close()
			cleanup()
		})
	}

	if withL1 {
		delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
		Require(t, err)
		delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
		Require(t, err)
		env.delayedInbox = delayedInbox
		env.lookupL2Tx = getLookupL2Tx(t, ctx, delayedBridge)
	}

	if opts.enableFiltering {
		// Register a Filterer and a FundsRecipient on L2 so the precompile
		// machinery is fully wired.
		builder.L2Info.GenerateAccount("Filterer")
		builder.L2Info.GenerateAccount("FundsRecipient")
		transferTx, _ := builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

		ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
		arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
		Require(t, err)
		addFiltererTx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(addFiltererTx)
		Require(t, err)
		setRecipientTx, err := arbOwner.SetFilteredFundsRecipient(&ownerTxOpts, builder.L2Info.GetAddress("FundsRecipient"))
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(setRecipientTx)
		Require(t, err)

		env.setupTxs = []common.Hash{transferTx.Hash(), addFiltererTx.Hash(), setRecipientTx.Hash()}
	}

	return env
}

func TestTransactionFeedDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{})
	defer env.cleanup()
	builder := env.builder

	builder.L2Info.GenerateAccount("User2")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	m := env.awaitFeedMessageFor(t, tx.Hash(), 5*time.Second)

	if m.Version != transactionfeed.TransactionFeedV1 {
		t.Fatalf("unexpected version: got %d, want %d", m.Version, transactionfeed.TransactionFeedV1)
	}
	if m.Transaction.BlockNumber != receipt.BlockNumber.Uint64() {
		t.Fatalf("block mismatch: feed=%d receipt=%d", m.Transaction.BlockNumber, receipt.BlockNumber.Uint64())
	}
	if m.Transaction.Receipt.Status != 1 {
		t.Fatalf("status: got %d, want 1", m.Transaction.Receipt.Status)
	}
	if m.Transaction.Receipt.ContractAddress != "" {
		t.Fatalf("expected empty contract_address for value transfer, got %q", m.Transaction.Receipt.ContractAddress)
	}
	if !strings.HasPrefix(m.Transaction.Receipt.EffectiveGasPrice, "0x") {
		t.Fatalf("effective_gas_price not hex: %q", m.Transaction.Receipt.EffectiveGasPrice)
	}
	if !strings.HasPrefix(m.Transaction.Receipt.BaseFee, "0x") {
		t.Fatalf("base_fee not hex: %q", m.Transaction.Receipt.BaseFee)
	}

	if !strings.HasPrefix(m.Transaction.RawTx, "0x") {
		t.Fatalf("raw_tx missing 0x prefix: %q", m.Transaction.RawTx)
	}
	rawBytes, err := hexutil.Decode(m.Transaction.RawTx)
	Require(t, err, "raw_tx hex decode")
	var roundTrip types.Transaction
	Require(t, roundTrip.UnmarshalBinary(rawBytes), "raw_tx unmarshal")
	if roundTrip.Hash() != tx.Hash() {
		t.Fatalf("raw_tx round-trip hash mismatch: got %s want %s", roundTrip.Hash().Hex(), tx.Hash().Hex())
	}

	env.assertFeedExactly(t, map[common.Hash]int{tx.Hash(): 1})
}

func TestTransactionFeedContractCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{})
	defer env.cleanup()
	builder := env.builder

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	deployAddr, deployTx, _, err := localgen.DeploySimple(&auth, builder.L2.Client)
	Require(t, err, "deploy Simple")
	_, err = builder.L2.EnsureTxSucceeded(deployTx)
	Require(t, err)

	m := env.awaitFeedMessageFor(t, deployTx.Hash(), 5*time.Second)

	if m.Transaction.Receipt.ContractAddress == "" {
		t.Fatal("expected non-empty contract_address for deploy tx")
	}
	if !strings.EqualFold(m.Transaction.Receipt.ContractAddress, deployAddr.Hex()) {
		t.Fatalf("contract_address mismatch: feed=%s deployed=%s",
			m.Transaction.Receipt.ContractAddress, deployAddr.Hex())
	}
	if m.Transaction.Receipt.Status != 1 {
		t.Fatalf("deploy status: got %d, want 1", m.Transaction.Receipt.Status)
	}

	env.assertFeedExactly(t, map[common.Hash]int{deployTx.Hash(): 1})
}

func TestTransactionFeedOrdering(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{})
	defer env.cleanup()
	builder := env.builder

	const n = 3
	recipients := []string{"User2", "User3", "User4"}
	txs := make([]*types.Transaction, n)
	receipts := make([]*types.Receipt, n)
	for i, name := range recipients {
		builder.L2Info.GenerateAccount(name)
		txs[i] = builder.L2Info.PrepareTx("Owner", name, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		Require(t, builder.L2.Client.SendTransaction(ctx, txs[i]))
		r, err := builder.L2.EnsureTxSucceeded(txs[i])
		Require(t, err)
		receipts[i] = r
	}

	feedMsgs := make([]*transactionfeed.TransactionFeedMessage, n)
	for i := range txs {
		feedMsgs[i] = env.awaitFeedMessageFor(t, txs[i].Hash(), 5*time.Second)
	}

	for i, m := range feedMsgs {
		wantBlock := receipts[i].BlockNumber.Uint64()
		wantIdx := uint64(receipts[i].TransactionIndex)
		if m.Transaction.BlockNumber != wantBlock {
			t.Fatalf("tx %d: block mismatch feed=%d receipt=%d", i, m.Transaction.BlockNumber, wantBlock)
		}
		if uint64(m.Transaction.TxIndex) != wantIdx {
			t.Fatalf("tx %d: tx_index mismatch feed=%d receipt=%d", i, m.Transaction.TxIndex, wantIdx)
		}
		// TODO: PGARound is currently a placeholder hardcoded to 0; revisit when PGA wiring lands.
		if m.PGARound != 0 {
			t.Fatalf("tx %d: pga_round got %d, want 0", i, m.PGARound)
		}
	}

	for i := 1; i < n; i++ {
		prev, curr := feedMsgs[i-1], feedMsgs[i]
		if curr.Transaction.BlockNumber < prev.Transaction.BlockNumber {
			t.Fatalf("block number went backwards between tx %d and %d", i-1, i)
		}
		if curr.Transaction.BlockNumber == prev.Transaction.BlockNumber &&
			curr.Transaction.TxIndex <= prev.Transaction.TxIndex {
			t.Fatalf("tx_index not increasing within block: tx %d idx=%d, tx %d idx=%d",
				i-1, prev.Transaction.TxIndex, i, curr.Transaction.TxIndex)
		}
	}

	env.assertFeedExactly(t, map[common.Hash]int{
		txs[0].Hash(): 1,
		txs[1].Hash(): 1,
		txs[2].Hash(): 1,
	})
}

func TestTransactionFeedManualRetryableRedeem(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{withL1: true})
	defer env.cleanup()
	builder := env.builder

	builder.L2Info.GenerateAccount("RetryDest")
	builder.L2Info.GenerateAccount("Beneficiary")
	destAddr := builder.L2Info.GetAddress("RetryDest")
	beneficiaryAddr := builder.L2Info.GetAddress("Beneficiary")

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)
	maxSubmissionCost := big.NewInt(1e16)
	maxFeePerGas := big.NewInt(l2pricing.InitialBaseFeeWei * 2)

	l1opts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1opts.Value = deposit
	l1tx, err := env.delayedInbox.CreateRetryableTicket(
		&l1opts,
		destAddr,
		callValue,
		maxSubmissionCost,
		beneficiaryAddr,
		beneficiaryAddr,
		big.NewInt(0),
		maxFeePerGas,
		nil,
	)
	Require(t, err)

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatal("l1Receipt indicated failure")
	}
	waitForL1DelayBlocks(t, builder)

	submissionTx := env.lookupL2Tx(l1Receipt)
	_, err = builder.L2.EnsureTxSucceeded(submissionTx)
	Require(t, err)
	ticketId := submissionTx.Hash()

	// Trigger a manual redeem from L2
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(types.ArbRetryableTxAddress, builder.L2.Client)
	Require(t, err)
	redeemerOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	redeemTx, err := arbRetryableTx.Redeem(&redeemerOpts, ticketId)
	Require(t, err)
	redeemReceipt, err := builder.L2.EnsureTxSucceeded(redeemTx)
	Require(t, err)
	if redeemReceipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("redeem status: got %d, want %d",
			redeemReceipt.Status, types.ReceiptStatusSuccessful)
	}

	retryTxHash := parseRedeemScheduledRetryHash(t, builder.L2.Client, redeemReceipt)
	retryReceipt, err := WaitForTx(ctx, builder.L2.Client, retryTxHash, 10*time.Second)
	Require(t, err)
	if retryReceipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("inner retry status: got %d, want %d",
			retryReceipt.Status, types.ReceiptStatusSuccessful)
	}

	m := env.awaitFeedMessageFor(t, redeemTx.Hash(), 10*time.Second)
	if m.Transaction.BlockNumber != redeemReceipt.BlockNumber.Uint64() {
		t.Fatalf("redeem block mismatch: feed=%d receipt=%d",
			m.Transaction.BlockNumber, redeemReceipt.BlockNumber.Uint64())
	}
	if uint64(m.Transaction.TxIndex) != uint64(redeemReceipt.TransactionIndex) {
		t.Fatalf("redeem tx_index mismatch: feed=%d receipt=%d",
			m.Transaction.TxIndex, redeemReceipt.TransactionIndex)
	}
	if uint64(m.Transaction.Receipt.Status) != types.ReceiptStatusSuccessful {
		t.Fatalf("redeem feed status: got %d, want %d",
			m.Transaction.Receipt.Status, types.ReceiptStatusSuccessful)
	}

	m = env.awaitFeedMessageFor(t, retryTxHash, 10*time.Second)
	if m.Transaction.BlockNumber != retryReceipt.BlockNumber.Uint64() {
		t.Fatalf("redeem block mismatch: feed=%d receipt=%d",
			m.Transaction.BlockNumber, retryReceipt.BlockNumber.Uint64())
	}
	if uint64(m.Transaction.TxIndex) != uint64(retryReceipt.TransactionIndex) {
		t.Fatalf("redeem tx_index mismatch: feed=%d receipt=%d",
			m.Transaction.TxIndex, retryReceipt.TransactionIndex)
	}
	if uint64(m.Transaction.Receipt.Status) != types.ReceiptStatusSuccessful {
		t.Fatalf("redeem feed status: got %d, want %d",
			m.Transaction.Receipt.Status, types.ReceiptStatusSuccessful)
	}

	env.assertFeedExactly(t, map[common.Hash]int{
		submissionTx.Hash(): 1,
		redeemTx.Hash():     1,
		retryTxHash:         1,
	})
}

func TestTransactionFeedCascadingRedeemRollback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{enableFiltering: true})
	defer env.cleanup()
	builder := env.builder

	builder.L2Info.GenerateAccount("Redeemer")
	redeemerFundingTx, _ := builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
	builder.L2Info.GenerateAccount("CleanBeneficiary")
	cleanBeneficiary := builder.L2Info.GetAddress("CleanBeneficiary")

	// Caller contract that forwards a CALL to the target; the cascading filter
	// trips when the retry executes the caller and the caller hits the target.
	callerAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callerAddr, callerDeployTx, _, err := localgen.DeployAddressFilterTest(&callerAuth, builder.L2.Client)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(callerDeployTx)
	Require(t, err)

	targetAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	targetAddr, targetDeployTx, _, err := localgen.DeployAddressFilterTest(&targetAuth, builder.L2.Client)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(targetDeployTx)
	Require(t, err)

	callerABI, err := localgen.AddressFilterTestMetaData.GetAbi()
	Require(t, err)
	retryData, err := callerABI.Pack("callTarget", targetAddr)
	Require(t, err)

	// Submit the retryable with gasLimit=0 so no auto-redeem is scheduled and
	// the ticket survives for the manual redeem below.
	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	maxSubmissionCost := big.NewInt(1e16)
	maxFeePerGas := big.NewInt(l2pricing.InitialBaseFeeWei * 2)
	l1opts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1opts.Value = deposit
	l1tx, err := env.delayedInbox.CreateRetryableTicket(
		&l1opts,
		callerAddr,
		common.Big0,
		maxSubmissionCost,
		cleanBeneficiary,
		cleanBeneficiary,
		common.Big0,
		maxFeePerGas,
		retryData,
	)
	Require(t, err)
	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	waitForL1DelayBlocks(t, builder)
	submissionTx := env.lookupL2Tx(l1Receipt)
	ticketId := submissionTx.Hash()
	_, err = builder.L2.EnsureTxSucceeded(submissionTx)
	Require(t, err)

	// Activate the address filter on the retryable's inner-call target.
	filter := newHashedChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// Sign the Redeem call without sending; the sequencer will reject the
	// submission.
	arbRetryable, err := precompilesgen.NewArbRetryableTx(types.ArbRetryableTxAddress, builder.L2.Client)
	Require(t, err)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	redeemOpts.NoSend = true
	signedRedeemTx, err := arbRetryable.Redeem(&redeemOpts, ticketId)
	Require(t, err)

	// Submit; the sequencer rejects with the cascading-filter error and the
	// tx never enters any block.
	sendErr := builder.L2.Client.SendTransaction(ctx, signedRedeemTx)
	if sendErr == nil {
		t.Fatal("expected SendTransaction to fail with cascading-filter error")
	}
	if !strings.Contains(sendErr.Error(), "cascading redeem filtered") {
		t.Fatalf("unexpected SendTransaction error: %v", sendErr)
	}

	// The rejected redeem tx is deliberately absent from want: the exactness
	// check fails if it -- or anything else unexpected -- was broadcast.
	env.assertFeedExactly(t, map[common.Hash]int{
		redeemerFundingTx.Hash(): 1,
		callerDeployTx.Hash():    1,
		targetDeployTx.Hash():    1,
		submissionTx.Hash():      1,
	})

	// Sanity check the ticket still exists -- the rollback was real.
	if _, err = arbRetryable.GetTimeout(&bind.CallOpts{Context: ctx}, ticketId); err != nil {
		t.Fatalf("retryable ticket should survive the rolled-back redeem: %v", err)
	}
}

func TestTransactionFeedDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).DontParalellise()
	// Leave execConfig.TransactionFeed at the zero value -- Enable defaults to false.
	cleanup := builder.Build(t)
	defer cleanup()

	if builder.L2.ExecNode.TransactionFeedServer != nil {
		t.Fatal("TransactionFeedServer should be nil when TransactionFeed.Enable is false")
	}
}

func TestTransactionFeedMultipleSubscribers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{})
	defer env.cleanup()
	builder := env.builder

	// Dial a second client and start a parallel recorder.
	port := testhelpers.AddrTCPPort(env.server.ListenerAddr(), t)
	conn2 := dialTransactionFeed(ctx, t, port)
	defer conn2.Close()
	recorder2 := startTransactionFeedRecorder(conn2)
	waitForTransactionFeedClients(t, env.server, 2, 3*time.Second)

	builder.L2Info.GenerateAccount("User2")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	_, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	m1 := env.awaitFeedMessageFor(t, tx.Hash(), 5*time.Second)
	m2 := recorder2.awaitFeedMessageFor(t, tx.Hash(), 5*time.Second)

	if m1.Transaction.BlockNumber != m2.Transaction.BlockNumber {
		t.Fatalf("subscribers disagree on block number: c1=%d c2=%d",
			m1.Transaction.BlockNumber, m2.Transaction.BlockNumber)
	}

	env.assertFeedExactly(t, map[common.Hash]int{tx.Hash(): 1})
	recorder2.assertNoError(t)
}

func TestTransactionFeedDynamicFeeTx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{})
	defer env.cleanup()
	builder := env.builder

	// PrepareTx builds a types.DynamicFeeTx under the hood.
	builder.L2Info.GenerateAccount("DynamicRecipient")
	tx := builder.L2Info.PrepareTx("Owner", "DynamicRecipient", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	m := env.awaitFeedMessageFor(t, tx.Hash(), 5*time.Second)
	if m.Transaction.BlockNumber != receipt.BlockNumber.Uint64() {
		t.Fatalf("block mismatch: feed=%d receipt=%d", m.Transaction.BlockNumber, receipt.BlockNumber.Uint64())
	}

	rawBytes, err := hexutil.Decode(m.Transaction.RawTx)
	Require(t, err, "raw_tx hex decode")
	var roundTrip types.Transaction
	Require(t, roundTrip.UnmarshalBinary(rawBytes), "raw_tx unmarshal")
	if roundTrip.Type() != types.DynamicFeeTxType {
		t.Fatalf("expected DynamicFeeTx (type %d), got type %d", types.DynamicFeeTxType, roundTrip.Type())
	}
	if roundTrip.Hash() != tx.Hash() {
		t.Fatalf("raw_tx round-trip hash mismatch: got %s want %s", roundTrip.Hash().Hex(), tx.Hash().Hex())
	}

	env.assertFeedExactly(t, map[common.Hash]int{tx.Hash(): 1})
}

func TestTransactionFeedSlowConsumerEviction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := newTransactionFeedConfigTest()
	cfg.ClientBuf = 4
	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{feedConfig: &cfg})
	defer env.cleanup()

	_ = env.conn.Close()
	deadline := time.Now().Add(3 * time.Second)
	for env.server.ClientCount() > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if env.server.ClientCount() != 0 {
		t.Fatalf("env client did not disconnect: ClientCount=%d", env.server.ClientCount())
	}

	port := testhelpers.AddrTCPPort(env.server.ListenerAddr(), t)
	stalledConn := dialTransactionFeed(ctx, t, port)
	defer stalledConn.Close()
	waitForTransactionFeedClients(t, env.server, 1, 3*time.Second)

	slowCounter := metrics.GetOrRegisterCounter("arb/transactionfeed/clients/disconnected/slow", nil)
	startSlow := slowCounter.Snapshot().Count()

	largeHex := "0x" + strings.Repeat("ab", 4096) // ~8 KB raw_tx field
	msg := &transactionfeed.TransactionFeedMessage{
		Version: transactionfeed.TransactionFeedV1,
		Transaction: transactionfeed.IncludedTransaction{
			RawTx:  largeHex,
			TxHash: "0x" + strings.Repeat("00", 32),
		},
	}
	for i := 0; i < 100; i++ {
		env.server.BroadcastTransaction(msg)
	}

	deadline = time.Now().Add(10 * time.Second)
	for env.server.ClientCount() > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if env.server.ClientCount() != 0 {
		t.Fatalf("expected stalled client to be evicted (ClientCount=0), got %d", env.server.ClientCount())
	}
	if delta := slowCounter.Snapshot().Count() - startSlow; delta < 1 {
		t.Fatalf("clientsDisconnectedSlow did not advance: delta=%d", delta)
	}
}

func TestTransactionFeedAbruptDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{})
	defer env.cleanup()
	builder := env.builder

	// One delivery to confirm the client is fully wired.
	builder.L2Info.GenerateAccount("AbruptRecv")
	tx := builder.L2Info.PrepareTx("Owner", "AbruptRecv", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	_, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	env.awaitFeedMessageFor(t, tx.Hash(), 5*time.Second)

	env.assertFeedExactly(t, map[common.Hash]int{tx.Hash(): 1})

	// Slam the conn shut and confirm the server unregisters the client.
	_ = env.conn.Close()
	deadline := time.Now().Add(5 * time.Second)
	for env.server.ClientCount() > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if env.server.ClientCount() != 0 {
		t.Fatalf("server did not unregister client after abrupt close: ClientCount=%d", env.server.ClientCount())
	}
}

func TestTransactionFeedStopAndWaitWithClients(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{})
	defer env.cleanup()
	builder := env.builder

	// Dial a second client so cleanup has two active sessions to tear down.
	port := testhelpers.AddrTCPPort(env.server.ListenerAddr(), t)
	conn2 := dialTransactionFeed(ctx, t, port)
	defer conn2.Close()
	waitForTransactionFeedClients(t, env.server, 2, 3*time.Second)

	builder.L2Info.GenerateAccount("PreStopRecv")
	tx := builder.L2Info.PrepareTx("Owner", "PreStopRecv", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	_, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	env.awaitFeedMessageFor(t, tx.Hash(), 5*time.Second)

	env.assertFeedExactly(t, map[common.Hash]int{tx.Hash(): 1})

	server := env.server
	env.cleanup() // triggers ExecutionNode.StopAndWait -> server.StopAndWait
	if c := server.ClientCount(); c != 0 {
		t.Fatalf("ClientCount should be 0 after StopAndWait, got %d", c)
	}
}

func TestTransactionFeedDelayedSequencerBroadcast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{withDelayedSequencer: true})
	defer env.cleanup()
	builder := env.builder

	builder.L2Info.GenerateAccount("DelayedDest")
	builder.L2Info.GenerateAccount("Beneficiary")
	destAddr := builder.L2Info.GetAddress("DelayedDest")
	beneficiaryAddr := builder.L2Info.GetAddress("Beneficiary")

	// Submit a retryable with a non-zero gas limit so the auto-redeem fires
	// in the same delayed-sequencing block as the submission. Both txs are
	// broadcast on the delayed-sequencing path (no address filter is active).
	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)
	maxSubmissionCost := big.NewInt(1e16)
	maxFeePerGas := big.NewInt(l2pricing.InitialBaseFeeWei * 2)
	gasLimit := big.NewInt(1_000_000)

	l1opts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1opts.Value = deposit
	l1tx, err := env.delayedInbox.CreateRetryableTicket(
		&l1opts,
		destAddr,
		callValue,
		maxSubmissionCost,
		beneficiaryAddr,
		beneficiaryAddr,
		gasLimit,
		maxFeePerGas,
		nil,
	)
	Require(t, err)
	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	waitForL1DelayBlocks(t, builder)

	submissionTx := env.lookupL2Tx(l1Receipt)
	submissionReceipt, err := builder.L2.EnsureTxSucceeded(submissionTx)
	Require(t, err)

	// The submission tx must appear on the feed -- proving the delayed
	// sequencing emit path is live.
	mSub := env.awaitFeedMessageFor(t, submissionTx.Hash(), 10*time.Second)
	if mSub.Transaction.BlockNumber != submissionReceipt.BlockNumber.Uint64() {
		t.Fatalf("submission block mismatch: feed=%d receipt=%d",
			mSub.Transaction.BlockNumber, submissionReceipt.BlockNumber.Uint64())
	}

	// The auto-redeem fires from arbos as a follow-up tx. Locate it via the
	// RedeemScheduled event on the submission receipt and confirm it lands
	// on the feed too.
	retryTxHash := parseRedeemScheduledRetryHash(t, builder.L2.Client, submissionReceipt)
	retryReceipt, err := WaitForTx(ctx, builder.L2.Client, retryTxHash, 10*time.Second)
	Require(t, err)
	if retryReceipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("auto-redeem status: got %d, want %d",
			retryReceipt.Status, types.ReceiptStatusSuccessful)
	}
	mRetry := env.awaitFeedMessageFor(t, retryTxHash, 10*time.Second)
	if mRetry.Transaction.BlockNumber != retryReceipt.BlockNumber.Uint64() {
		t.Fatalf("auto-redeem block mismatch: feed=%d receipt=%d",
			mRetry.Transaction.BlockNumber, retryReceipt.BlockNumber.Uint64())
	}

	env.assertFeedExactly(t, map[common.Hash]int{
		submissionTx.Hash(): 1,
		retryTxHash:         1,
	})
}

func TestTransactionFeedReorgRebroadcast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{})
	defer env.cleanup()
	builder := env.builder

	// Send a tx; capture the head message index immediately after it lands.
	builder.L2Info.GenerateAccount("ReorgRecv")
	tx := builder.L2Info.PrepareTx("Owner", "ReorgRecv", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	_, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	first := env.awaitFeedMessageFor(t, tx.Hash(), 5*time.Second)

	headIdx, err := builder.L2.ExecNode.ExecEngine.HeadMessageIndex()
	Require(t, err)
	if headIdx == 0 {
		t.Fatal("head message index is 0; cannot reorg")
	}

	// Reorg out the message containing tx (and any after). The execution
	// engine sends popped messages to its resequence channel, which now
	// re-broadcasts via the wired transactionFeedServer.
	reorgFrom := arbutil.MessageIndex(headIdx)
	Require(t, builder.L2.ConsensusNode.TxStreamer.ReorgAt(reorgFrom))

	// Expect a second broadcast of the same tx hash.
	msgs := env.recorder.awaitFeedMessages(t, tx.Hash(), 2, 10*time.Second)
	second := msgs[1]
	if second.TimestampMs < first.TimestampMs {
		t.Fatalf("second broadcast timestamp %d earlier than first %d",
			second.TimestampMs, first.TimestampMs)
	}

	// The multiset count pins down exactly two deliveries of the same hash.
	env.assertFeedExactly(t, map[common.Hash]int{tx.Hash(): 2})
}
