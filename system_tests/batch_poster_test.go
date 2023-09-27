// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/util/redisutil"
)

func TestBatchPosterParallel(t *testing.T) {
	testBatchPosterParallel(t, false)
}

func TestRedisBatchPosterParallel(t *testing.T) {
	testBatchPosterParallel(t, true)
}

func testBatchPosterParallel(t *testing.T, useRedis bool) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	conf := arbnode.ConfigDefaultL1Test()
	conf.BatchPoster.Enable = false
	conf.BatchPoster.RedisUrl = redisUrl
	testNodeA := NewNodeBuilder(ctx).SetNodeConfig(conf).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNodeA.L1Stack)
	defer testNodeA.L2Node.StopAndWait()

	l2clientB, nodeB := Create2ndNode(t, ctx, testNodeA.L2Node, testNodeA.L1Stack, testNodeA.L1Info, &testNodeA.L2Info.ArbInitData, nil)
	defer nodeB.StopAndWait()

	testNodeA.L2Info.GenerateAccount("User2")

	var txs []*types.Transaction

	for i := 0; i < 100; i++ {
		tx := testNodeA.L2Info.PrepareTx("Owner", "User2", testNodeA.L2Info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)

		err := testNodeA.L2Client.SendTransaction(ctx, tx)
		Require(t, err)
	}

	for _, tx := range txs {
		_, err := EnsureTxSucceeded(ctx, testNodeA.L2Client, tx)
		Require(t, err)
	}

	firstTxData, err := txs[0].MarshalBinary()
	Require(t, err)
	seqTxOpts := testNodeA.L1Info.GetDefaultTransactOpts("Sequencer", ctx)
	conf.BatchPoster.Enable = true
	conf.BatchPoster.MaxSize = len(firstTxData) * 2
	startL1Block, err := testNodeA.L1Client.BlockNumber(ctx)
	Require(t, err)
	for i := 0; i < parallelBatchPosters; i++ {
		// Make a copy of the batch poster config so NewBatchPoster calling Validate() on it doesn't race
		batchPosterConfig := conf.BatchPoster
		batchPoster, err := arbnode.NewBatchPoster(nil, testNodeA.L2Node.L1Reader, testNodeA.L2Node.InboxTracker, testNodeA.L2Node.TxStreamer, testNodeA.L2Node.SyncMonitor, func() *arbnode.BatchPosterConfig { return &batchPosterConfig }, testNodeA.L2Node.DeployInfo, &seqTxOpts, nil)
		Require(t, err)
		batchPoster.Start(ctx)
		defer batchPoster.StopAndWait()
	}

	lastTxHash := txs[len(txs)-1].Hash()
	for i := 90; i > 0; i-- {
		SendWaitTestTransactions(t, ctx, testNodeA.L1Client, []*types.Transaction{
			testNodeA.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
		time.Sleep(500 * time.Millisecond)
		_, err := l2clientB.TransactionReceipt(ctx, lastTxHash)
		if err == nil {
			break
		}
		if i == 0 {
			Require(t, err)
		}
	}

	// I've locally confirmed that this passes when the clique period is set to 1.
	// However, setting the clique period to 1 slows everything else (including the L1 deployment for this test) down to a crawl.
	if false {
		// Make sure the batch poster is able to post multiple batches in one block
		endL1Block, err := testNodeA.L1Client.BlockNumber(ctx)
		Require(t, err)
		seqInbox, err := arbnode.NewSequencerInbox(testNodeA.L1Client, testNodeA.L2Node.DeployInfo.SequencerInbox, 0)
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

	l2balance, err := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Sign() == 0 {
		Fatal(t, "Unexpected zero balance")
	}
}

func TestBatchPosterLargeTx(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := arbnode.ConfigDefaultL1Test()
	conf.Sequencer.MaxTxDataSize = 110000
	testNodeA := NewNodeBuilder(ctx).SetNodeConfig(conf).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNodeA.L1Stack)
	defer testNodeA.L2Node.StopAndWait()

	l2clientB, nodeB := Create2ndNode(t, ctx, testNodeA.L2Node, testNodeA.L1Stack, testNodeA.L1Info, &testNodeA.L2Info.ArbInitData, nil)
	defer nodeB.StopAndWait()

	data := make([]byte, 100000)
	_, err := rand.Read(data)
	Require(t, err)
	faucetAddr := testNodeA.L2Info.GetAddress("Faucet")
	gas := testNodeA.L2Info.TransferGas + 20000*uint64(len(data))
	tx := testNodeA.L2Info.PrepareTxTo("Faucet", &faucetAddr, gas, common.Big0, data)
	err = testNodeA.L2Client.SendTransaction(ctx, tx)
	Require(t, err)
	receiptA, err := EnsureTxSucceeded(ctx, testNodeA.L2Client, tx)
	Require(t, err)
	receiptB, err := EnsureTxSucceededWithTimeout(ctx, l2clientB, tx, time.Second*30)
	Require(t, err)
	if receiptA.BlockHash != receiptB.BlockHash {
		Fatal(t, "receipt A block hash", receiptA.BlockHash, "does not equal receipt B block hash", receiptB.BlockHash)
	}
}

func TestBatchPosterKeepsUp(t *testing.T) {
	t.Skip("This test is for manual inspection and would be unreliable in CI even if automated")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := arbnode.ConfigDefaultL1Test()
	conf.BatchPoster.CompressionLevel = brotli.BestCompression
	conf.BatchPoster.MaxDelay = time.Hour
	conf.RPC.RPCTxFeeCap = 1000.
	testNodeA := NewNodeBuilder(ctx).SetNodeConfig(conf).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNodeA.L1Stack)
	defer testNodeA.L2Node.StopAndWait()
	testNodeA.L2Info.GasPrice = big.NewInt(100e9)

	go func() {
		data := make([]byte, 90000)
		_, err := rand.Read(data)
		Require(t, err)
		for {
			gas := testNodeA.L2Info.TransferGas + 20000*uint64(len(data))
			tx := testNodeA.L2Info.PrepareTx("Faucet", "Faucet", gas, common.Big0, data)
			err = testNodeA.L2Client.SendTransaction(ctx, tx)
			Require(t, err)
			_, err := EnsureTxSucceeded(ctx, testNodeA.L2Client, tx)
			Require(t, err)
		}
	}()

	start := time.Now()
	for {
		time.Sleep(time.Second)
		batches, err := testNodeA.L2Node.InboxTracker.GetBatchCount()
		Require(t, err)
		postedMessages, err := testNodeA.L2Node.InboxTracker.GetBatchMessageCount(batches - 1)
		Require(t, err)
		haveMessages, err := testNodeA.L2Node.TxStreamer.GetMessageCount()
		Require(t, err)
		duration := time.Since(start)
		fmt.Printf("batches posted: %v over %v (%.2f batches/second)\n", batches, duration, float64(batches)/(float64(duration)/float64(time.Second)))
		fmt.Printf("backlog: %v message\n", haveMessages-postedMessages)
	}
}
