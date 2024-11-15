// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
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

func externalSignerTestCfg(addr common.Address, url string) (*dataposter.ExternalSignerCfg, error) {
	cp, err := externalsignertest.CertPaths()
	if err != nil {
		return nil, fmt.Errorf("getting certificates path: %w", err)
	}
	return &dataposter.ExternalSignerCfg{
		Address:          common.Bytes2Hex(addr.Bytes()),
		URL:              url,
		Method:           externalsignertest.SignerMethod,
		RootCA:           cp.ServerCert,
		ClientCert:       cp.ClientCert,
		ClientPrivateKey: cp.ClientKey,
	}, nil
}

func testBatchPosterParallel(t *testing.T, useRedis bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := externalsignertest.NewServer(t)
	go func() {
		if err := srv.Start(); err != nil {
			log.Error("Failed to start external signer server:", err)
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
	signerCfg, err := externalSignerTestCfg(srv.Address, srv.URL())
	if err != nil {
		t.Fatalf("Error getting external signer config: %v", err)
	}
	builder.nodeConfig.BatchPoster.DataPoster.ExternalSigner = *signerCfg

	cleanup := builder.Build(t)
	defer cleanup()
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()
	builder.L2Info.GenerateAccount("User2")

	addNewBatchPoster(ctx, t, builder, srv.Address)

	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		builder.L1Info.PrepareTxTo("Faucet", &srv.Address, 30000, big.NewInt(1e18), nil)})

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
	parentChainID, err := builder.L1.Client.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to get parent chain id: %v", err)
	}
	for i := 0; i < parallelBatchPosters; i++ {
		// Make a copy of the batch poster config so NewBatchPoster calling Validate() on it doesn't race
		batchPosterConfig := builder.nodeConfig.BatchPoster
		batchPoster, err := arbnode.NewBatchPoster(ctx,
			&arbnode.BatchPosterOpts{
				DataPosterDB:  nil,
				L1Reader:      builder.L2.ConsensusNode.L1Reader,
				Inbox:         builder.L2.ConsensusNode.InboxTracker,
				Streamer:      builder.L2.ConsensusNode.TxStreamer,
				VersionGetter: builder.L2.ExecNode,
				SyncMonitor:   builder.L2.ConsensusNode.SyncMonitor,
				Config:        func() *arbnode.BatchPosterConfig { return &batchPosterConfig },
				DeployInfo:    builder.L2.ConsensusNode.DeployInfo,
				TransactOpts:  &seqTxOpts,
				DAPWriter:     nil,
				ParentChainID: parentChainID,
			},
		)
		Require(t, err)
		batchPoster.Start(ctx)
		defer batchPoster.StopAndWait()
	}

	lastTxHash := txs[len(txs)-1].Hash()
	for i := 90; i >= 0; i-- {
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

func testAllowPostingFirstBatchWhenSequencerMessageCountMismatch(t *testing.T, enabled bool) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// creates first node with batch poster disabled
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.BatchPoster.Enable = false
	cleanup := builder.Build(t)
	defer cleanup()
	testClientNonBatchPoster := builder.L2

	// adds a batch to the sequencer inbox with a wrong next message count,
	// should be 2 but it is set to 10
	seqInbox, err := bridgegen.NewSequencerInbox(builder.L1Info.GetAddress("SequencerInbox"), builder.L1.Client)
	Require(t, err)
	seqOpts := builder.L1Info.GetDefaultTransactOpts("Sequencer", ctx)
	tx, err := seqInbox.AddSequencerL2Batch(&seqOpts, big.NewInt(1), nil, big.NewInt(1), common.Address{}, big.NewInt(1), big.NewInt(10))
	Require(t, err)
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	// creates a batch poster
	nodeConfigBatchPoster := arbnode.ConfigDefaultL1Test()
	nodeConfigBatchPoster.BatchPoster.Dangerous.AllowPostingFirstBatchWhenSequencerMessageCountMismatch = enabled
	testClientBatchPoster, cleanupBatchPoster := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfigBatchPoster})
	defer cleanupBatchPoster()

	// sends a transaction through the batch poster
	accountName := "User2"
	builder.L2Info.GenerateAccount(accountName)
	tx = builder.L2Info.PrepareTx("Owner", accountName, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err = testClientBatchPoster.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = testClientBatchPoster.EnsureTxSucceeded(tx)
	Require(t, err)

	if enabled {
		// if AllowPostingFirstBatchWhenSequencerMessageCountMismatch is enabled
		// then the L2 transaction should be posted to L1, and the non batch
		// poster node should be able to see it
		_, err = WaitForTx(ctx, testClientNonBatchPoster.Client, tx.Hash(), time.Second*3)
		Require(t, err)
		l2balance, err := testClientNonBatchPoster.Client.BalanceAt(ctx, builder.L2Info.GetAddress(accountName), nil)
		Require(t, err)
		if l2balance.Cmp(big.NewInt(1e12)) != 0 {
			t.Fatal("Unexpected balance:", l2balance)
		}
	} else {
		// if AllowPostingFirstBatchWhenSequencerMessageCountMismatch is disabled
		// then the L2 transaction should not be posted to L1, so the non
		// batch poster will not be able to see it
		_, err = WaitForTx(ctx, testClientNonBatchPoster.Client, tx.Hash(), time.Second*3)
		if err == nil {
			Fatal(t, "tx received by non batch poster node with AllowPostingFirstBatchWhenSequencerMessageCountMismatch disabled")
		}
		l2balance, err := testClientNonBatchPoster.Client.BalanceAt(ctx, builder.L2Info.GetAddress(accountName), nil)
		Require(t, err)
		if l2balance.Cmp(big.NewInt(0)) != 0 {
			t.Fatal("Unexpected balance:", l2balance)
		}
	}
}

func TestAllowPostingFirstBatchWhenSequencerMessageCountMismatchEnabled(t *testing.T) {
	testAllowPostingFirstBatchWhenSequencerMessageCountMismatch(t, true)
}

func TestAllowPostingFirstBatchWhenSequencerMessageCountMismatchDisabled(t *testing.T) {
	testAllowPostingFirstBatchWhenSequencerMessageCountMismatch(t, false)
}
