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
	builder := NewNodeBuilder(ctx).SetNodeConfig(conf).SetIsSequencer(true)
	l1A, l2A := builder.BuildL2OnL1(t)
	// testNodeA := NewNodeBuilder(ctx).SetNodeConfig(conf).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, l1A.Stack)
	defer l2A.Node.StopAndWait()

	l2B := builder.Build2ndNodeDAS(t, &l2A.Info.ArbInitData, nil)
	defer l2B.Node.StopAndWait()

	l2A.Info.GenerateAccount("User2")

	var txs []*types.Transaction

	for i := 0; i < 100; i++ {
		tx := l2A.Info.PrepareTx("Owner", "User2", l2A.Info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)

		err := l2A.Client.SendTransaction(ctx, tx)
		Require(t, err)
	}

	for _, tx := range txs {
		_, err := EnsureTxSucceeded(ctx, l2A.Client, tx)
		Require(t, err)
	}

	firstTxData, err := txs[0].MarshalBinary()
	Require(t, err)
	seqTxOpts := l1A.Info.GetDefaultTransactOpts("Sequencer", ctx)
	conf.BatchPoster.Enable = true
	conf.BatchPoster.MaxSize = len(firstTxData) * 2
	startL1Block, err := l1A.Client.BlockNumber(ctx)
	Require(t, err)
	for i := 0; i < parallelBatchPosters; i++ {
		// Make a copy of the batch poster config so NewBatchPoster calling Validate() on it doesn't race
		batchPosterConfig := conf.BatchPoster
		batchPoster, err := arbnode.NewBatchPoster(ctx, nil, l2A.Node.L1Reader, l2A.Node.InboxTracker, l2A.Node.TxStreamer, l2A.Node.SyncMonitor, func() *arbnode.BatchPosterConfig { return &batchPosterConfig }, l2A.Node.DeployInfo, &seqTxOpts, nil)
		Require(t, err)
		batchPoster.Start(ctx)
		defer batchPoster.StopAndWait()
	}

	lastTxHash := txs[len(txs)-1].Hash()
	for i := 90; i > 0; i-- {
		SendWaitTestTransactions(t, ctx, l1A.Client, []*types.Transaction{
			l1A.Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
		time.Sleep(500 * time.Millisecond)
		_, err := l2B.Client.TransactionReceipt(ctx, lastTxHash)
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
		endL1Block, err := l1A.Client.BlockNumber(ctx)
		Require(t, err)
		seqInbox, err := arbnode.NewSequencerInbox(l1A.Client, l2A.Node.DeployInfo.SequencerInbox, 0)
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

	l2balance, err := l2B.Client.BalanceAt(ctx, l2A.Info.GetAddress("User2"), nil)
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
	builder := NewNodeBuilder(ctx).SetNodeConfig(conf).SetIsSequencer(true)
	l1A, l2A := builder.BuildL2OnL1(t)
	defer requireClose(t, l1A.Stack)
	defer l2A.Node.StopAndWait()

	l2B := builder.Build2ndNodeDAS(t, &l2A.Info.ArbInitData, nil)
	defer l2B.Node.StopAndWait()

	data := make([]byte, 100000)
	_, err := rand.Read(data)
	Require(t, err)
	faucetAddr := l2A.Info.GetAddress("Faucet")
	gas := l2A.Info.TransferGas + 20000*uint64(len(data))
	tx := l2A.Info.PrepareTxTo("Faucet", &faucetAddr, gas, common.Big0, data)
	err = l2A.Client.SendTransaction(ctx, tx)
	Require(t, err)
	receiptA, err := EnsureTxSucceeded(ctx, l2A.Client, tx)
	Require(t, err)
	receiptB, err := EnsureTxSucceededWithTimeout(ctx, l2B.Client, tx, time.Second*30)
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
	builder := NewNodeBuilder(ctx).SetNodeConfig(conf).SetIsSequencer(true)
	l1A, l2A := builder.BuildL2OnL1(t)
	defer requireClose(t, l1A.Stack)
	defer l2A.Node.StopAndWait()
	l2A.Info.GasPrice = big.NewInt(100e9)

	go func() {
		data := make([]byte, 90000)
		_, err := rand.Read(data)
		Require(t, err)
		for {
			gas := l2A.Info.TransferGas + 20000*uint64(len(data))
			tx := l2A.Info.PrepareTx("Faucet", "Faucet", gas, common.Big0, data)
			err = l2A.Client.SendTransaction(ctx, tx)
			Require(t, err)
			_, err := EnsureTxSucceeded(ctx, l2A.Client, tx)
			Require(t, err)
		}
	}()

	start := time.Now()
	for {
		time.Sleep(time.Second)
		batches, err := l2A.Node.InboxTracker.GetBatchCount()
		Require(t, err)
		postedMessages, err := l2A.Node.InboxTracker.GetBatchMessageCount(batches - 1)
		Require(t, err)
		haveMessages, err := l2A.Node.TxStreamer.GetMessageCount()
		Require(t, err)
		duration := time.Since(start)
		fmt.Printf("batches posted: %v over %v (%.2f batches/second)\n", batches, duration, float64(batches)/(float64(duration)/float64(time.Second)))
		fmt.Printf("backlog: %v message\n", haveMessages-postedMessages)
	}
}
