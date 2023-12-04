// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/util/redisutil"
)

func TestBatchPosterParallel(t *testing.T) {
	testBatchPosterParallel(t, false)
}

func TestRedisBatchPosterParallel(t *testing.T) {
	testBatchPosterParallel(t, true)
}

func addNewBatchPoster(ctx context.Context, t *testing.T, builder *NodeBuilder, address common.Address) {
	t.Helper()
	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(builder.L2.ConsensusNode.DeployInfo.UpgradeExecutor, builder.L1.Client)
	if err != nil {
		t.Fatal("Failed to get new upgrade executor", err)
	}
	sequencerInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	if err != nil {
		t.Fatal("Failed to parse sequencer inbox abi", err)
	}
	setIsBatchPoster, err := sequencerInboxABI.Pack("setIsBatchPoster", address, true)
	if err != nil {
		t.Fatal("Failed to pack setIsBatchPoster", err)
	}
	ownerOpts := builder.L1Info.GetDefaultTransactOpts("RollupOwner", ctx)
	tx, err := upgradeExecutor.ExecuteCall(
		&ownerOpts,
		builder.L1Info.GetAddress("SequencerInbox"),
		setIsBatchPoster)
	if err != nil {
		t.Fatalf("Error creating transaction to set batch poster: %v", err)
	}
	if _, err := builder.L1.EnsureTxSucceeded(tx); err != nil {
		t.Fatalf("Error setting batch poster: %v", err)
	}
}

func testBatchPosterParallel(t *testing.T, useRedis bool) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	httpSrv, srv := newServer(ctx, t)
	t.Cleanup(func() {
		if err := httpSrv.Shutdown(ctx); err != nil {
			t.Fatalf("Error shutting down http server: %v", err)
		}
	})
	go func() {
		fmt.Println("Server is listening on port 1234...")
		if err := httpSrv.ListenAndServeTLS(signerServerCert, signerServerKey); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stdout, "ListenAndServeTLS() unexpected error:  %v", err)
			return
		}
	}()

	var redisUrl string
	if useRedis {
		redisUrl = redisutil.CreateTestRedis(ctx, t)
	}
	parallelBatchPosters := 1
	if redisUrl != "" {
		client, err := redisutil.RedisClientFromURL(redisUrl)
		Require(t, err)
		err = client.Del(ctx, "data-poster.queue").Err()
		Require(t, err)
		parallelBatchPosters = 4
	}

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.BatchPoster.Enable = false
	builder.nodeConfig.BatchPoster.RedisUrl = redisUrl
	builder.nodeConfig.BatchPoster.DataPoster.ExternalSigner = *externalSignerTestCfg(srv.address)

	cleanup := builder.Build(t)
	defer cleanup()
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()
	builder.L2Info.GenerateAccount("User2")

	addNewBatchPoster(ctx, t, builder, srv.address)

	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		builder.L1Info.PrepareTxTo("Faucet", &srv.address, 30000, big.NewInt(1e18), nil)})

	var txs []*types.Transaction

	for i := 0; i < 100; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)

		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
	}

	for _, tx := range txs {
		_, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	firstTxData, err := txs[0].MarshalBinary()
	Require(t, err)
	seqTxOpts := builder.L1Info.GetDefaultTransactOpts("Sequencer", ctx)
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.BatchPoster.MaxSize = len(firstTxData) * 2
	startL1Block, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)
	for i := 0; i < parallelBatchPosters; i++ {
		// Make a copy of the batch poster config so NewBatchPoster calling Validate() on it doesn't race
		batchPosterConfig := builder.nodeConfig.BatchPoster
		batchPoster, err := arbnode.NewBatchPoster(ctx,
			&arbnode.BatchPosterOpts{
				DataPosterDB: nil,
				L1Reader:     builder.L2.ConsensusNode.L1Reader,
				Inbox:        builder.L2.ConsensusNode.InboxTracker,
				Streamer:     builder.L2.ConsensusNode.TxStreamer,
				SyncMonitor:  builder.L2.ConsensusNode.SyncMonitor,
				Config:       func() *arbnode.BatchPosterConfig { return &batchPosterConfig },
				DeployInfo:   builder.L2.ConsensusNode.DeployInfo,
				TransactOpts: &seqTxOpts,
				DAWriter:     nil,
			},
		)
		Require(t, err)
		batchPoster.Start(ctx)
		defer batchPoster.StopAndWait()
	}

	lastTxHash := txs[len(txs)-1].Hash()
	for i := 90; i > 0; i-- {
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
		time.Sleep(500 * time.Millisecond)
		_, err := testClientB.Client.TransactionReceipt(ctx, lastTxHash)
		if err == nil {
			break
		}
		if i == 0 {
			Require(t, err)
		}
	}

	// TODO: factor this out in separate test case and skip it or delete this
	// code entirely.
	// I've locally confirmed that this passes when the clique period is set to 1.
	// However, setting the clique period to 1 slows everything else (including the L1 deployment for this test) down to a crawl.
	if false {
		// Make sure the batch poster is able to post multiple batches in one block
		endL1Block, err := builder.L1.Client.BlockNumber(ctx)
		Require(t, err)
		seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.L2.ConsensusNode.DeployInfo.SequencerInbox, 0)
		Require(t, err)
		batches, err := seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(startL1Block), new(big.Int).SetUint64(endL1Block))
		Require(t, err)
		var foundMultipleInBlock bool
		for i := range batches {
			if i == 0 {
				continue
			}
			if batches[i-1].ParentChainBlockNumber == batches[i].ParentChainBlockNumber {
				foundMultipleInBlock = true
				break
			}
		}

		if !foundMultipleInBlock {
			Fatal(t, "only found one batch per block")
		}
	}

	l2balance, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Sign() == 0 {
		Fatal(t, "Unexpected zero balance")
	}
}

func TestBatchPosterLargeTx(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig.Sequencer.MaxTxDataSize = 110000
	cleanup := builder.Build(t)
	defer cleanup()

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	data := make([]byte, 100000)
	_, err := rand.Read(data)
	Require(t, err)
	faucetAddr := builder.L2Info.GetAddress("Faucet")
	gas := builder.L2Info.TransferGas + 20000*uint64(len(data))
	tx := builder.L2Info.PrepareTxTo("Faucet", &faucetAddr, gas, common.Big0, data)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	receiptA, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	receiptB, err := testClientB.EnsureTxSucceededWithTimeout(tx, time.Second*30)
	Require(t, err)
	if receiptA.BlockHash != receiptB.BlockHash {
		Fatal(t, "receipt A block hash", receiptA.BlockHash, "does not equal receipt B block hash", receiptB.BlockHash)
	}
}

func TestBatchPosterKeepsUp(t *testing.T) {
	t.Skip("This test is for manual inspection and would be unreliable in CI even if automated")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.BatchPoster.CompressionLevel = brotli.BestCompression
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour
	builder.execConfig.RPC.RPCTxFeeCap = 1000.
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GasPrice = big.NewInt(100e9)

	go func() {
		data := make([]byte, 90000)
		_, err := rand.Read(data)
		Require(t, err)
		for {
			gas := builder.L2Info.TransferGas + 20000*uint64(len(data))
			tx := builder.L2Info.PrepareTx("Faucet", "Faucet", gas, common.Big0, data)
			err = builder.L2.Client.SendTransaction(ctx, tx)
			Require(t, err)
			_, err := builder.L2.EnsureTxSucceeded(tx)
			Require(t, err)
		}
	}()

	start := time.Now()
	for {
		time.Sleep(time.Second)
		batches, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
		Require(t, err)
		postedMessages, err := builder.L2.ConsensusNode.InboxTracker.GetBatchMessageCount(batches - 1)
		Require(t, err)
		haveMessages, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
		Require(t, err)
		duration := time.Since(start)
		fmt.Printf("batches posted: %v over %v (%.2f batches/second)\n", batches, duration, float64(batches)/(float64(duration)/float64(time.Second)))
		fmt.Printf("backlog: %v message\n", haveMessages-postedMessages)
	}
}
