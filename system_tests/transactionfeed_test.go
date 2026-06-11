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

func startTransactionFeedReader(ctx context.Context, conn net.Conn) (<-chan *transactionfeed.TransactionFeedMessage, <-chan error) {
	msgs := make(chan *transactionfeed.TransactionFeedMessage, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(msgs)
		for {
			data, err := wsutil.ReadServerBinary(conn)
			if err != nil {
				select {
				case errs <- err:
				default:
				}
				return
			}
			var msg transactionfeed.TransactionFeedMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				select {
				case errs <- err:
				default:
				}
				return
			}
			select {
			case msgs <- &msg:
			case <-ctx.Done():
				return
			}
		}
	}()
	return msgs, errs
}

func awaitFeedMessageFor(t *testing.T, msgs <-chan *transactionfeed.TransactionFeedMessage, errs <-chan error, txHash common.Hash, timeout time.Duration) *transactionfeed.TransactionFeedMessage {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case m, ok := <-msgs:
			if !ok {
				t.Fatalf("transaction feed closed before tx %s arrived", txHash.Hex())
			}
			if strings.EqualFold(m.Transaction.TxHash, txHash.Hex()) {
				return m
			}
		case err := <-errs:
			t.Fatalf("transaction feed reader error while waiting for tx %s: %v", txHash.Hex(), err)
		case <-deadline:
			t.Fatalf("timeout waiting for tx %s on transaction feed", txHash.Hex())
		}
	}
}

// assertTxNotBroadcastBefore drains the feed until the sentinel tx's broadcast
// arrives, failing if disallowed appears in any message en route. Because the
// feed is FIFO and the sentinel was submitted after disallowed, the sentinel's
// arrival is a deterministic "we have caught up past that point" signal --
// faster and less flaky than a wall-clock window.
func assertTxNotBroadcastBefore(t *testing.T, msgs <-chan *transactionfeed.TransactionFeedMessage, errs <-chan error, disallowed common.Hash, sentinel common.Hash, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case m, ok := <-msgs:
			if !ok {
				t.Fatalf("transaction feed closed before sentinel tx %s arrived", sentinel.Hex())
			}
			if strings.EqualFold(m.Transaction.TxHash, disallowed.Hex()) {
				t.Fatalf("unexpected feed message for disallowed tx %s", disallowed.Hex())
			}
			if strings.EqualFold(m.Transaction.TxHash, sentinel.Hex()) {
				return
			}
		case err := <-errs:
			t.Fatalf("transaction feed reader error while waiting for sentinel %s: %v", sentinel.Hex(), err)
		case <-deadline:
			t.Fatalf("timeout waiting for sentinel tx %s on transaction feed", sentinel.Hex())
		}
	}
}

func assertNoReaderError(t *testing.T, errs <-chan error) {
	t.Helper()
	select {
	case err := <-errs:
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
	builder      *NodeBuilder
	msgs         <-chan *transactionfeed.TransactionFeedMessage
	errs         <-chan error
	server       *transactionfeed.Server                 // the running feed server
	conn         net.Conn                                // the first dialed client conn
	delayedInbox *bridgegen.Inbox                        // non-nil when withL1
	lookupL2Tx   func(*types.Receipt) *types.Transaction // non-nil when withL1
	cleanup      func()
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
	msgs, errs := startTransactionFeedReader(ctx, conn)
	waitForTransactionFeedClients(t, rfs, 1, 3*time.Second)

	env := &transactionFeedTestEnv{
		builder: builder,
		msgs:    msgs,
		errs:    errs,
		server:  rfs,
		conn:    conn,
		cleanup: func() {
			_ = conn.Close()
			cleanup()
		},
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
		builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

		ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
		arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
		Require(t, err)
		tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		tx, err = arbOwner.SetFilteredFundsRecipient(&ownerTxOpts, builder.L2Info.GetAddress("FundsRecipient"))
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
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

	m := awaitFeedMessageFor(t, env.msgs, env.errs, tx.Hash(), 5*time.Second)

	if m.Version != uint32(transactionfeed.TransactionFeedV1) {
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

	assertNoReaderError(t, env.errs)
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

	m := awaitFeedMessageFor(t, env.msgs, env.errs, deployTx.Hash(), 5*time.Second)

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

	assertNoReaderError(t, env.errs)
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
		feedMsgs[i] = awaitFeedMessageFor(t, env.msgs, env.errs, txs[i].Hash(), 5*time.Second)
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
		// PGARound is currently a placeholder hardcoded to 1; revisit when PGA wiring lands.
		if m.PGARound != 1 {
			t.Fatalf("tx %d: pga_round got %d, want 1", i, m.PGARound)
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

	assertNoReaderError(t, env.errs)
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

	m := awaitFeedMessageFor(t, env.msgs, env.errs, redeemTx.Hash(), 10*time.Second)
	if m.Transaction.BlockNumber != redeemReceipt.BlockNumber.Uint64() {
		t.Fatalf("redeem block mismatch: feed=%d receipt=%d",
			m.Transaction.BlockNumber, redeemReceipt.BlockNumber.Uint64())
	}
	if uint64(m.Transaction.TxIndex) != uint64(redeemReceipt.TransactionIndex) {
		t.Fatalf("redeem tx_index mismatch: feed=%d receipt=%d",
			m.Transaction.TxIndex, redeemReceipt.TransactionIndex)
	}
	if m.Transaction.Receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("redeem feed status: got %d, want %d",
			m.Transaction.Receipt.Status, types.ReceiptStatusSuccessful)
	}

	m = awaitFeedMessageFor(t, env.msgs, env.errs, retryTxHash, 10*time.Second)
	if m.Transaction.BlockNumber != retryReceipt.BlockNumber.Uint64() {
		t.Fatalf("redeem block mismatch: feed=%d receipt=%d",
			m.Transaction.BlockNumber, retryReceipt.BlockNumber.Uint64())
	}
	if uint64(m.Transaction.TxIndex) != uint64(retryReceipt.TransactionIndex) {
		t.Fatalf("redeem tx_index mismatch: feed=%d receipt=%d",
			m.Transaction.TxIndex, retryReceipt.TransactionIndex)
	}
	if m.Transaction.Receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("redeem feed status: got %d, want %d",
			m.Transaction.Receipt.Status, types.ReceiptStatusSuccessful)
	}

	assertNoReaderError(t, env.errs)
}

func TestTransactionFeedCascadingRedeemRollback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{enableFiltering: true})
	defer env.cleanup()
	builder := env.builder

	builder.L2Info.GenerateAccount("Redeemer")
	builder.L2.TransferBalance(t, "Owner", "Redeemer", big.NewInt(1e18), builder.L2Info)
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

	// Sign the Redeem call without sending so we can capture the hash even
	// though the sequencer will reject the submission.
	arbRetryable, err := precompilesgen.NewArbRetryableTx(types.ArbRetryableTxAddress, builder.L2.Client)
	Require(t, err)
	redeemOpts := builder.L2Info.GetDefaultTransactOpts("Redeemer", ctx)
	redeemOpts.NoSend = true
	signedRedeemTx, err := arbRetryable.Redeem(&redeemOpts, ticketId)
	Require(t, err)
	redeemHash := signedRedeemTx.Hash()

	// Submit; the sequencer rejects with the cascading-filter error and the
	// tx never enters any block.
	sendErr := builder.L2.Client.SendTransaction(ctx, signedRedeemTx)
	if sendErr == nil {
		t.Fatal("expected SendTransaction to fail with cascading-filter error")
	}
	if !strings.Contains(sendErr.Error(), "cascading redeem filtered") {
		t.Fatalf("unexpected SendTransaction error: %v", sendErr)
	}

	// The Redeem tx must NEVER reach the feed. Submit a sentinel value
	// transfer and use its broadcast as a deterministic "we've drained past
	// the rejected tx" signal -- avoids a flaky wall-clock window.
	builder.L2Info.GenerateAccount("Sentinel")
	sentinelTx := builder.L2Info.PrepareTx("Owner", "Sentinel", builder.L2Info.TransferGas, big.NewInt(1e6), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, sentinelTx))
	_, err = builder.L2.EnsureTxSucceeded(sentinelTx)
	Require(t, err)
	assertTxNotBroadcastBefore(t, env.msgs, env.errs, redeemHash, sentinelTx.Hash(), 10*time.Second)

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

	// Dial a second client and start a parallel reader.
	port := testhelpers.AddrTCPPort(env.server.ListenerAddr(), t)
	conn2 := dialTransactionFeed(ctx, t, port)
	defer conn2.Close()
	msgs2, errs2 := startTransactionFeedReader(ctx, conn2)
	waitForTransactionFeedClients(t, env.server, 2, 3*time.Second)

	builder.L2Info.GenerateAccount("User2")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	_, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	m1 := awaitFeedMessageFor(t, env.msgs, env.errs, tx.Hash(), 5*time.Second)
	m2 := awaitFeedMessageFor(t, msgs2, errs2, tx.Hash(), 5*time.Second)

	if m1.Transaction.BlockNumber != m2.Transaction.BlockNumber {
		t.Fatalf("subscribers disagree on block number: c1=%d c2=%d",
			m1.Transaction.BlockNumber, m2.Transaction.BlockNumber)
	}
	assertNoReaderError(t, env.errs)
	assertNoReaderError(t, errs2)
}

func TestTransactionFeedDynamicFeeTx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{})
	defer env.cleanup()
	builder := env.builder

	builder.L2Info.GenerateAccount("DynamicRecipient")
	recipient := builder.L2Info.GetAddress("DynamicRecipient")

	header, err := builder.L2.Client.HeaderByNumber(ctx, nil)
	Require(t, err, "fetch L2 head for baseFee")
	tip := big.NewInt(1e9)
	gasFeeCap := new(big.Int).Add(header.BaseFee, tip)

	faucet := builder.L2Info.GetInfoWithPrivKey("Faucet")
	tx := builder.L2Info.SignTxAs("Faucet", &types.DynamicFeeTx{
		To:        &recipient,
		Gas:       builder.L2Info.TransferGas,
		GasTipCap: tip,
		GasFeeCap: gasFeeCap,
		Value:     big.NewInt(1e12),
		Nonce:     faucet.Nonce.Add(1) - 1,
	})
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	m := awaitFeedMessageFor(t, env.msgs, env.errs, tx.Hash(), 5*time.Second)
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
	assertNoReaderError(t, env.errs)
}

func TestTransactionFeedSlowConsumerEviction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ClientBuf=1 so cc.out can only hold a single pending broadcast. A tight
	// PingInterval keeps the writer goroutine stuck in conn.Ping (waiting up
	// to WriteTimeout for a pong that a stalled client never sends) -- while
	// the writer is parked there, the next broadcast finds cc.out full and
	// triggers slow-consumer eviction.
	cfg := newTransactionFeedConfigTest()
	cfg.ClientBuf = 1
	cfg.PingInterval = 100 * time.Millisecond
	cfg.WriteTimeout = 5 * time.Second
	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{feedConfig: &cfg})
	defer env.cleanup()
	builder := env.builder

	// env's first client is being read by startTransactionFeedReader (healthy).
	// Add a stalled second client that never reads from its conn -- so the
	// server's pings to it go unanswered.
	port := testhelpers.AddrTCPPort(env.server.ListenerAddr(), t)
	stalledConn := dialTransactionFeed(ctx, t, port)
	defer stalledConn.Close()
	waitForTransactionFeedClients(t, env.server, 2, 3*time.Second)

	slowCounter := metrics.GetOrRegisterCounter("arb/transactionfeed/clients/disconnected/slow", nil)
	startSlow := slowCounter.Snapshot().Count()

	// Let the first ping fire and park the stalled client's writer.
	time.Sleep(200 * time.Millisecond)

	// Send a small burst -- once cc.out holds its single message, the next
	// broadcast evicts.
	const burst = 3
	recipients := make([]string, burst)
	txs := make([]*types.Transaction, burst)
	for i := 0; i < burst; i++ {
		recipients[i] = fmt.Sprintf("SlowRecv%d", i)
		builder.L2Info.GenerateAccount(recipients[i])
		txs[i] = builder.L2Info.PrepareTx("Owner", recipients[i], builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		Require(t, builder.L2.Client.SendTransaction(ctx, txs[i]))
		_, err := builder.L2.EnsureTxSucceeded(txs[i])
		Require(t, err)
	}

	// Healthy reader still receives every message in order.
	for i := 0; i < burst; i++ {
		awaitFeedMessageFor(t, env.msgs, env.errs, txs[i].Hash(), 10*time.Second)
	}

	// Stalled client should have been evicted: ClientCount drops to 1 and
	// the slow-disconnect metric advanced by at least 1.
	deadline := time.Now().Add(10 * time.Second)
	for env.server.ClientCount() > 1 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if env.server.ClientCount() != 1 {
		t.Fatalf("expected stalled client to be evicted (ClientCount=1), got %d", env.server.ClientCount())
	}
	if delta := slowCounter.Snapshot().Count() - startSlow; delta < 1 {
		t.Fatalf("clientsDisconnectedSlow did not advance: delta=%d", delta)
	}
	assertNoReaderError(t, env.errs)
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
	awaitFeedMessageFor(t, env.msgs, env.errs, tx.Hash(), 5*time.Second)

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
	awaitFeedMessageFor(t, env.msgs, env.errs, tx.Hash(), 5*time.Second)

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
	// in the same delayed-sequencing block as the submission. Both txs go
	// through DelayedFilteringSequencingHooks (FilteredTxCount == 0 because
	// no address filter is active) and so should hit the broadcast.
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

	// The submission tx must appear on the feed -- proving the
	// DelayedFilteringSequencingHooks.TxAccepted path is live.
	mSub := awaitFeedMessageFor(t, env.msgs, env.errs, submissionTx.Hash(), 10*time.Second)
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
	mRetry := awaitFeedMessageFor(t, env.msgs, env.errs, retryTxHash, 10*time.Second)
	if mRetry.Transaction.BlockNumber != retryReceipt.BlockNumber.Uint64() {
		t.Fatalf("auto-redeem block mismatch: feed=%d receipt=%d",
			mRetry.Transaction.BlockNumber, retryReceipt.BlockNumber.Uint64())
	}

	assertNoReaderError(t, env.errs)
}

func TestTransactionFeedMaxClientsCap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// MaxClients=2: env's auto-dial is client 1, we'll dial a 2nd, and the 3rd
	// must be rejected before WS upgrade.
	cfg := newTransactionFeedConfigTest()
	cfg.MaxClients = 2
	env := setupTransactionFeedTest(t, ctx, transactionFeedTestOpts{feedConfig: &cfg})
	defer env.cleanup()

	port := testhelpers.AddrTCPPort(env.server.ListenerAddr(), t)
	secondConn := dialTransactionFeed(ctx, t, port)
	defer secondConn.Close()
	waitForTransactionFeedClients(t, env.server, 2, 3*time.Second)

	rejectedCounter := metrics.GetOrRegisterCounter("arb/transactionfeed/clients/rejected/at_cap", nil)
	startRejected := rejectedCounter.Snapshot().Count()

	// Third dial: dialer should error because the server returns HTTP 503
	// before completing the WebSocket upgrade.
	dctx, dcancel := context.WithTimeout(ctx, 3*time.Second)
	defer dcancel()
	thirdConn, _, _, err := ws.Dialer{}.Dial(dctx, fmt.Sprintf("ws://127.0.0.1:%d/", port))
	if err == nil {
		_ = thirdConn.Close()
		t.Fatalf("expected third dial to be rejected; got connected conn")
	}

	if got := env.server.ClientCount(); got != 2 {
		t.Fatalf("ClientCount should stay at 2 after rejection, got %d", got)
	}
	if delta := rejectedCounter.Snapshot().Count() - startRejected; delta < 1 {
		t.Fatalf("clientsRejectedAtCap did not advance: delta=%d", delta)
	}
	assertNoReaderError(t, env.errs)
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

	first := awaitFeedMessageFor(t, env.msgs, env.errs, tx.Hash(), 5*time.Second)

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
	second := awaitFeedMessageFor(t, env.msgs, env.errs, tx.Hash(), 10*time.Second)
	if second.TimestampMs < first.TimestampMs {
		t.Fatalf("second broadcast timestamp %d earlier than first %d",
			second.TimestampMs, first.TimestampMs)
	}
	assertNoReaderError(t, env.errs)
}
