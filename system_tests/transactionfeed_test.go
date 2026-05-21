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
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
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
			data, err := wsutil.ReadServerText(conn)
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

func setupTransactionFeedTest(t *testing.T, ctx context.Context) (*NodeBuilder, func(), <-chan *transactionfeed.TransactionFeedMessage, <-chan error) {
	t.Helper()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).DontParalellise()
	builder.nodeConfig.Feed.TransactionFeed = newTransactionFeedConfigTest()
	cleanup := builder.Build(t)

	rfs := builder.L2.ConsensusNode.TransactionFeedServer
	if rfs == nil {
		cleanup()
		t.Fatal("TransactionFeedServer was not constructed")
	}
	port := testhelpers.AddrTCPPort(rfs.ListenerAddr(), t)

	conn := dialTransactionFeed(ctx, t, port)
	msgs, errs := startTransactionFeedReader(ctx, conn)
	waitForTransactionFeedClients(t, rfs, 1, 3*time.Second)

	tearDown := func() {
		_ = conn.Close()
		cleanup()
	}
	return builder, tearDown, msgs, errs
}

func TestTransactionFeedDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup, msgs, errs := setupTransactionFeedTest(t, ctx)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	m := awaitFeedMessageFor(t, msgs, errs, tx.Hash(), 5*time.Second)

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

	select {
	case err := <-errs:
		t.Fatalf("transaction feed reader error: %v", err)
	default:
	}
}

func TestTransactionFeedContractCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup, msgs, errs := setupTransactionFeedTest(t, ctx)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	deployAddr, deployTx, _, err := localgen.DeploySimple(&auth, builder.L2.Client)
	Require(t, err, "deploy Simple")
	_, err = builder.L2.EnsureTxSucceeded(deployTx)
	Require(t, err)

	m := awaitFeedMessageFor(t, msgs, errs, deployTx.Hash(), 5*time.Second)

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
}

func TestTransactionFeedOrdering(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup, msgs, errs := setupTransactionFeedTest(t, ctx)
	defer cleanup()

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
		feedMsgs[i] = awaitFeedMessageFor(t, msgs, errs, txs[i].Hash(), 5*time.Second)
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
}

func setupTransactionFeedTestWithL1(t *testing.T, ctx context.Context) (
	*NodeBuilder,
	*bridgegen.Inbox,
	func(*types.Receipt) *types.Transaction,
	func(),
	<-chan *transactionfeed.TransactionFeedMessage,
	<-chan error,
) {
	t.Helper()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).DontParalellise()
	builder.nodeConfig.Feed.TransactionFeed = newTransactionFeedConfigTest()
	cleanup := builder.Build(t)

	rfs := builder.L2.ConsensusNode.TransactionFeedServer
	if rfs == nil {
		cleanup()
		t.Fatal("TransactionFeedServer was not constructed")
	}
	port := testhelpers.AddrTCPPort(rfs.ListenerAddr(), t)

	conn := dialTransactionFeed(ctx, t, port)
	msgs, errs := startTransactionFeedReader(ctx, conn)
	waitForTransactionFeedClients(t, rfs, 1, 3*time.Second)

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
	Require(t, err)
	lookupL2Tx := getLookupL2Tx(t, ctx, delayedBridge)

	tearDown := func() {
		_ = conn.Close()
		cleanup()
	}
	return builder, delayedInbox, lookupL2Tx, tearDown, msgs, errs
}

func assertNoFeedMessageFor(t *testing.T, msgs <-chan *transactionfeed.TransactionFeedMessage, errs <-chan error, txHash common.Hash, window time.Duration) {
	t.Helper()
	deadline := time.After(window)
	for {
		select {
		case m, ok := <-msgs:
			if !ok {
				return
			}
			if strings.EqualFold(m.Transaction.TxHash, txHash.Hex()) {
				t.Fatalf("unexpected feed message for tx %s", txHash.Hex())
			}
		case err := <-errs:
			t.Fatalf("transaction feed reader error while draining for tx %s: %v", txHash.Hex(), err)
		case <-deadline:
			return
		}
	}
}

func parseRedeemScheduledRetryHash(t *testing.T, client *ethclient.Client, receipt *types.Receipt) common.Hash {
	t.Helper()
	arbRetryableFilterer, err := precompilesgen.NewArbRetryableTxFilterer(common.HexToAddress("6e"), client)
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

func TestTransactionFeedManualRetryableRedeem(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, delayedInbox, lookupL2Tx, cleanup, msgs, errs := setupTransactionFeedTestWithL1(t, ctx)
	defer cleanup()

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
	l1tx, err := delayedInbox.CreateRetryableTicket(
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

	submissionTx := lookupL2Tx(l1Receipt)
	_, err = builder.L2.EnsureTxSucceeded(submissionTx)
	Require(t, err)
	ticketId := submissionTx.Hash()

	// Trigger a manual redeem from L2
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
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

	m := awaitFeedMessageFor(t, msgs, errs, redeemTx.Hash(), 10*time.Second)
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

	m = awaitFeedMessageFor(t, msgs, errs, retryTxHash, 10*time.Second)
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
}

func setupTransactionFeedFilterTest(t *testing.T, ctx context.Context) (
	*NodeBuilder,
	*bridgegen.Inbox,
	func(*types.Receipt) *types.Transaction,
	func(),
	<-chan *transactionfeed.TransactionFeedMessage,
	<-chan error,
) {
	t.Helper()

	arbOSInit := &params.ArbOSInit{TransactionFilteringEnabled: true}
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithArbOSVersion(params.ArbosVersion_60).
		WithArbOSInit(arbOSInit).
		DontParalellise()
	builder.isSequencer = true
	builder.nodeConfig.DelayedSequencer.Enable = true
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1
	builder.nodeConfig.Feed.TransactionFeed = newTransactionFeedConfigTest()

	cleanup := builder.Build(t)

	rfs := builder.L2.ConsensusNode.TransactionFeedServer
	if rfs == nil {
		cleanup()
		t.Fatal("TransactionFeedServer was not constructed")
	}
	port := testhelpers.AddrTCPPort(rfs.ListenerAddr(), t)
	conn := dialTransactionFeed(ctx, t, port)
	msgs, errs := startTransactionFeedReader(ctx, conn)
	waitForTransactionFeedClients(t, rfs, 1, 3*time.Second)

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
	Require(t, err)
	lookupL2Tx := getLookupL2Tx(t, ctx, delayedBridge)

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

	tearDown := func() {
		_ = conn.Close()
		cleanup()
	}
	return builder, delayedInbox, lookupL2Tx, tearDown, msgs, errs
}

func TestTransactionFeedCascadingRedeemRollback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, delayedInbox, lookupL2Tx, cleanup, msgs, errs := setupTransactionFeedFilterTest(t, ctx)
	defer cleanup()

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
	l1tx, err := delayedInbox.CreateRetryableTicket(
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
	submissionTx := lookupL2Tx(l1Receipt)
	ticketId := submissionTx.Hash()
	_, err = builder.L2.EnsureTxSucceeded(submissionTx)
	Require(t, err)

	// Activate the address filter on the retryable's inner-call target.
	filter := newHashedChecker([]common.Address{targetAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, filter)

	// Sign the Redeem call without sending so we can capture the hash even
	// though the sequencer will reject the submission.
	arbRetryable, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
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

	// The Redeem tx must NEVER reach the feed. Drain briefly to confirm.
	assertNoFeedMessageFor(t, msgs, errs, redeemHash, 2*time.Second)

	// Sanity check the ticket still exists -- the rollback was real.
	if _, err = arbRetryable.GetTimeout(&bind.CallOpts{Context: ctx}, ticketId); err != nil {
		t.Fatalf("retryable ticket should survive the rolled-back redeem: %v", err)
	}
}
