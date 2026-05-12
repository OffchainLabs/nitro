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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/transactionfeed"
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
