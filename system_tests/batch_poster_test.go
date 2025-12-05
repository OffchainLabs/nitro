// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/util/redisutil"
)

func TestBatchPosterParallel(t *testing.T) {
	testBatchPosterParallel(t, false, false)
}

func TestRedisBatchPosterParallel(t *testing.T) {
	testBatchPosterParallel(t, true, false)
}

func TestRedisBatchPosterParallelWithRedisLock(t *testing.T) {
	testBatchPosterParallel(t, true, true)
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

func testBatchPosterParallel(t *testing.T, useRedis bool, useRedisLock bool) {
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
	if redisutil.IsSharedTestRedisInstance() {
		builder.DontParalellise()
	}
	builder.nodeConfig.BatchPoster.Enable = false
	builder.nodeConfig.BatchPoster.RedisUrl = redisUrl
	builder.nodeConfig.BatchPoster.RedisLock.Enable = useRedisLock
	signerCfg, err := dataposter.ExternalSignerTestCfg(srv.Address, srv.URL())
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
	builder.nodeConfig.BatchPoster.MaxCalldataBatchSize = len(firstTxData) * 2
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
				DAPWriters:    nil,
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
			Require(t, err, "timed out waiting for last transaction to be included in batch and synced by node B")
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

func TestRedisBatchPosterHandoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := externalsignertest.NewServer(t)
	go func() {
		if err := srv.Start(); err != nil {
			log.Error("Failed to start external signer server:", err)
			return
		}
	}()
	miniredis, redisUrl := redisutil.CreateTestRedisAdvanced(ctx, t)
	client, err := redisutil.RedisClientFromURL(redisUrl)
	Require(t, err)
	err = client.Del(ctx, "data-poster.queue").Err()
	Require(t, err)

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	if redisutil.IsSharedTestRedisInstance() {
		builder.DontParalellise()
	}
	builder.nodeConfig.BatchPoster.Enable = false
	builder.nodeConfig.BatchPoster.RedisUrl = redisUrl
	builder.nodeConfig.BatchPoster.RedisLock.LockoutDuration = 100 * time.Millisecond
	builder.nodeConfig.BatchPoster.RedisLock.RefreshDuration = 50 * time.Millisecond
	signerCfg, err := dataposter.ExternalSignerTestCfg(srv.Address, srv.URL())
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
	builder.nodeConfig.BatchPoster.MaxCalldataBatchSize = len(firstTxData) * 2
	parentChainID, err := builder.L1.Client.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to get parent chain id: %v", err)
	}

	newBatchPoster := func() *arbnode.BatchPoster {
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
				DAPWriters:    nil,
				ParentChainID: parentChainID,
			},
		)
		Require(t, err)
		return batchPoster
	}

	nameA, batchPosterA := "BatchPoster1", newBatchPoster()
	nameB, batchPosterB := "BatchPoster2", newBatchPoster()

	for i := 0; i < 3; i++ {
		posted, err := batchPosterA.MaybePostSequencerBatch(ctx)
		if err != nil {
			t.Fatalf("Batch poster %s failed with unexpected error: %v, iter: %d", nameA, err, i)
		}
		if !posted {
			t.Fatalf("Batch poster %s should have posted, iter: %d", nameA, i)
		}
		posted, err = batchPosterB.MaybePostSequencerBatch(ctx)
		if posted {
			t.Fatalf("Batch poster %s should not have posted just after %s, iter: %d", nameB, nameA, i)
		}
		if err != nil && !strings.Contains(err.Error(), "failed to acquire lock") {
			t.Fatalf("Batch poster %s failed with unexpected error: %v, iter: %d", nameB, err, i)
		}
		if miniredis != nil {
			// fastforward to expire redis lock
			miniredis.FastForward(builder.nodeConfig.BatchPoster.RedisLock.LockoutDuration)
			// we need also to wait for our redislock.Simple lockedUntil to expire after RefreshDuration
			time.Sleep(builder.nodeConfig.BatchPoster.RedisLock.RefreshDuration)
		} else {
			// we need to wait full LockoutDuration for the redis key to expire
			time.Sleep(builder.nodeConfig.BatchPoster.RedisLock.LockoutDuration)
		}

		// swap posters
		nameA, batchPosterA, nameB, batchPosterB = nameB, batchPosterB, nameA, batchPosterA
	}
	// start one batch poster to post the rest
	batchPosterA.Start(ctx)
	defer batchPosterA.StopAndWait()

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
			Require(t, err, "timed out waiting for last transaction to be included in batch and synced by node B")
		}
	}

	l2balance, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Sign() == 0 {
		Fatal(t, "Unexpected zero balance")
	}
}

func TestBatchPosterLargeTx(t *testing.T) {
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// creates first node with batch poster disabled
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithTakeOwnership(false)
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

func GetBatchCount(t *testing.T, builder *NodeBuilder) uint64 {
	t.Helper()
	sequenceInbox, err := bridgegen.NewSequencerInbox(builder.L1Info.GetAddress("SequencerInbox"), builder.L1.Client)
	Require(t, err)
	batchCount, err := sequenceInbox.BatchCount(&bind.CallOpts{Context: builder.ctx})
	Require(t, err)
	return batchCount.Uint64()
}

func CheckBatchCount(t *testing.T, builder *NodeBuilder, want uint64) {
	t.Helper()
	if got := GetBatchCount(t, builder); got != want {
		t.Fatalf("invalid batch count, want %v, got %v", want, got)
	}
}

func testBatchPosterDelayBuffer(t *testing.T, delayBufferEnabled bool) {
	const messagesPerBatch = 3
	const numBatches = 3
	var threshold uint64
	if delayBufferEnabled {
		threshold = 200
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithDelayBuffer(threshold)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	// and prevent race conditions due to config changes during the test. We'll call MaybePostSequencerBatch manually.
	cleanup := builder.Build(t)
	defer cleanup()
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	// Advance L1 to force a batch given the delay buffer threshold
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, int(threshold)) // #nosec G115
	initialBatchCount := GetBatchCount(t, builder)
	builder.L2.ConsensusNode.BatchPoster.StopAndWait() // stop batchposter loop so we can manually call MaybePostBatch instead
	for batch := uint64(0); batch < numBatches; batch++ {
		txs := make(types.Transactions, messagesPerBatch)
		for i := range txs {
			txs[i] = builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
		}
		SendSignedTxesInBatchViaL1(t, ctx, builder.L1Info, builder.L1.Client, builder.L2.Client, txs)

		// batch poster loop, should do nothing
		_, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
		Require(t, err)

		// Check messages didn't appear in 2nd node
		_, err = WaitForTx(ctx, testClientB.Client, txs[0].Hash(), 100*time.Millisecond)
		if err == nil || !errors.Is(err, context.DeadlineExceeded) {
			Fatal(t, "expected context-deadline exceeded error, but got:", err)
		}

		// check batch was not posted
		CheckBatchCount(t, builder, initialBatchCount+batch)

		// Advance L1 to force a batch given the delay buffer threshold
		AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, int(threshold)) // #nosec G115
		if !delayBufferEnabled {
			// If the delay buffer is disabled, set max delay to zero to force it
			CheckBatchCount(t, builder, initialBatchCount+batch)
			builder.nodeConfig.BatchPoster.MaxDelay = 0
			builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
		}
		// Run batch poster loop again, this one should post a batch
		_, err = builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
		Require(t, err)
		for _, tx := range txs {
			_, err := testClientB.EnsureTxSucceeded(tx)
			Require(t, err, "tx not found on second node")
		}
		CheckBatchCount(t, builder, initialBatchCount+batch+1)
		if !delayBufferEnabled {
			builder.nodeConfig.BatchPoster.MaxDelay = time.Hour
			builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
		}
	}
}

func TestBatchPosterDelayBufferEnabled(t *testing.T) {
	testBatchPosterDelayBuffer(t, true)
}

func TestBatchPosterDelayBufferDisabled(t *testing.T) {
	testBatchPosterDelayBuffer(t, false)
}

func TestBatchPosterDelayBufferDontForceNonDelayedMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const threshold = 100
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithDelayBuffer(threshold)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour // set high max-delay so we can test the delay buffer
	cleanup := builder.Build(t)
	defer cleanup()
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	// Send non-delayed message and advance L1
	initialBatchCount := GetBatchCount(t, builder)
	const numTxs = 3
	txs := make(types.Transactions, numTxs)
	for i := range txs {
		txs[i] = builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
	}
	builder.L2.SendWaitTestTransactions(t, txs)
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, threshold)

	// Even advancing the L1, the batch won't be posted because it doesn't contain a delayed message
	CheckBatchCount(t, builder, initialBatchCount)

	builder.L2.ConsensusNode.BatchPoster.StopAndWait() // allow us to modify config and call loop at will
	// Set delay to zero to force non-delayed messages
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	_, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}
	CheckBatchCount(t, builder, initialBatchCount+1)
}

func TestParentChainNonEIP7623(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)

	// Build L1 and L2
	cleanupL1AndL2 := builder.Build(t)
	defer cleanupL1AndL2()

	// Check if L2's parent chain is using EIP-7623
	latestHeader, err := builder.L2.ConsensusNode.L1Reader.LastHeader(ctx)
	Require(t, err)
	isUsingEIP7623, err := builder.L2.ConsensusNode.BatchPoster.ParentChainIsUsingEIP7623(ctx, latestHeader)
	Require(t, err)
	if !isUsingEIP7623 {
		t.Fatal("L2's parent chain should be using EIP-7623")
	}

	// Build L3
	cleanupL3FirstNode := builder.BuildL3OnL2(t)
	defer cleanupL3FirstNode()

	// Check if L3's parent chain is not using EIP-7623
	latestHeader, err = builder.L3.ConsensusNode.L1Reader.LastHeader(ctx)
	Require(t, err)
	isUsingEIP7623, err = builder.L3.ConsensusNode.BatchPoster.ParentChainIsUsingEIP7623(ctx, latestHeader)
	Require(t, err)
	if isUsingEIP7623 {
		t.Fatal("L3's parent chain should not be using EIP-7623")
	}
}

func TestBatchPosterWithDelayProofsAndBacklog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const threshold = 10
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithDelayBuffer(threshold).
		WithL1ClientWrapper(t).
		WithTakeOwnership(false)
	cleanup := builder.Build(t)
	defer cleanup()

	initialBatchCount := GetBatchCount(t, builder)

	// Filter batch poster transactions using the L1 client wrapper
	batchPosterAddress := builder.L1Info.GetAddress("Sequencer")
	batchPosterTxsChan := make(chan *types.Transaction, 100)
	batchPosterTxs := []*types.Transaction{}
	builder.L1.ClientWrapper.EnableRawTransactionFilter(batchPosterAddress, batchPosterTxsChan)

	builder.L2Info.GenerateAccount("User2")
	delayedTx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)

	const numBatches = 3
	for i := 0; i < numBatches; i++ {
		// Send transactions using the bridge to generate delay proofs
		SendSignedTxViaL1(t, ctx, builder.L1Info, builder.L1.Client, builder.L2.Client, delayedTx)
		// Capture the batch poster transaction, ensuring the batch was closed. If it was not
		// closed, the select will time out and the test will fail.
		select {
		case tx := <-batchPosterTxsChan:
			batchPosterTxs = append(batchPosterTxs, tx)
		case <-time.After(1 * time.Second):
			Fatal(t, "Timed out waiting for batch poster tx")
		}
	}
	select {
	case <-batchPosterTxsChan:
		Fatal(t, "Unexpected batch poster transaction")
	default:
	}

	// Check that the batch poster txs didn't arrive in L1
	CheckBatchCount(t, builder, initialBatchCount)

	// Disable the filter and send the batch poster transactions
	builder.L1.ClientWrapper.DisableRawTransactionFilter()
	builder.L1.SendWaitTestTransactions(t, batchPosterTxs)
	CheckBatchCount(t, builder, initialBatchCount+numBatches)
}

func TestBatchPosterL1SurplusMatchesBatchGasFlaky(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		TakeOwnership()

	// Enable delayed sequencer to process batch posting reports
	builder.nodeConfig.DelayedSequencer.Enable = true
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1
	cleanup := builder.Build(t)
	defer cleanup()
	// Set chain params: GasFloorPerToken for fusaka, but dont charge L1 to make pricing easier
	ownerAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	tx, err := arbOwner.SetParentGasFloorPerToken(&ownerAuth, uint64(10))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)
	tx, err = arbOwner.SetL1PricePerUnit(&ownerAuth, big.NewInt(0))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)
	tx, err = arbOwner.SetL1PricingInertia(&ownerAuth, 1)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)

	// craft a large L2 tx
	data := make([]byte, 70000)
	_, err = rand.Read(data)
	Require(t, err)

	// send tx from Owner -> random address to ensure it is included
	to := builder.L2Info.GetAddress("Faucet")
	gas := builder.L2Info.TransferGas + 20000*uint64(len(data))
	tx = builder.L2Info.PrepareTxTo("Owner", &to, gas, common.Big0, data)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)

	// wait for L2 tx receipt
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	l2Block, err := builder.L2.Client.BlockByHash(ctx, receipt.BlockHash)
	Require(t, err)

	// wait for this tx to be posted in a batch, and check which batch
	var batchNum *big.Int
	for {
		batch, err := builder.L2.ConsensusNode.FindInboxBatchContainingMessage(arbutil.MessageIndex(l2Block.NumberU64())).Await(ctx)
		if err == nil && batch.Found {
			batchNum = new(big.Int).SetUint64(batch.BatchNum)
			break
		}
		t.Logf("waiting for tx to be posted in a batch")
		<-time.After(time.Millisecond * 10)
	}

	// find the transaction that posted this batch to parent chain
	seqInboxContract, err := bridgegen.NewSequencerInbox(builder.L1Info.GetAddress("SequencerInbox"), builder.L1.Client)
	Require(t, err)
	var batchTxHash common.Hash
	for {
		it, err := seqInboxContract.FilterSequencerBatchDelivered(nil, []*big.Int{batchNum}, nil, nil)
		if err == nil && it.Next() {
			batchTxHash = it.Event.Raw.TxHash
			break
		}
		t.Logf("waiting to find sequencer batch message")
		<-time.After(time.Millisecond * 10)
	}

	// get receipt of batch tx to know gas used
	batchReceipt, err := builder.L1.Client.TransactionReceipt(ctx, batchTxHash)
	Require(t, err)
	batchL1Block, err := builder.L1.Client.HeaderByHash(ctx, batchReceipt.BlockHash)
	Require(t, err)
	batchL1PostCost, _ := new(big.Int).Mul(batchL1Block.BaseFee, new(big.Int).SetUint64(batchReceipt.GasUsed)).Float64()

	t.Log("batch posting found", "l1Block", batchL1Block.Number.Uint64(), "basefee", batchL1Block.BaseFee.Uint64(), "postCost", batchL1PostCost, "gasUsed", batchReceipt.GasUsed)
	// Advance L1 to satisfy finality requirements for the batch posting report to be processed
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 2)

	// find the L2 block which processed the delayed messages (the header Nonce increases)
	latestL2, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	var foundBlock uint64
	// scan recent L2 blocks for nonce increase
	// we expect this to be within the last 50 since the batch poster should post quickly
	for b := l2Block.Number().Uint64(); b <= latestL2; b++ {
		block, err := builder.L2.Client.BlockByNumber(ctx, new(big.Int).SetUint64(b))
		Require(t, err)
		t.Logf("checking L2 block %d: nonce=%d (looking for %d)", b, block.Nonce(), batchNum)
		if block.Nonce() == batchNum.Uint64()+1 {
			foundBlock = block.Header().Number.Uint64()
			break
		}
	}
	if foundBlock == 0 {
		Fatal(t, "couldn't find L2 block that processed delayed message")
	}
	// check headers before and after the block containing our receipt
	// Find the block that processed the delayed instruction by scanning L2 headers around receipt block

	// Build l1 gas info accessor
	gasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	Require(t, err)

	// call gasInfo surplus before and after this block
	surplusBefore, err := gasInfo.GetL1PricingSurplus(&bind.CallOpts{BlockNumber: new(big.Int).SetUint64(foundBlock - 1)})
	Require(t, err)
	surplusAfter, err := gasInfo.GetL1PricingSurplus(&bind.CallOpts{BlockNumber: new(big.Int).SetUint64(foundBlock)})
	Require(t, err)

	// compute delta
	delta, _ := new(big.Int).Sub(surplusBefore, surplusAfter).Float64()

	t.Log("BATCH prices", "from-receipt", batchL1PostCost, "surplus-delta", delta)

	checkPercentDiff(t, delta, batchL1PostCost, 0.1)
}

func TestBatchPosterActuallyPostsBlobsToL1(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// Turn on unconditional blob posting
	builder.nodeConfig.BatchPoster.Post4844Blobs = true
	builder.nodeConfig.BatchPoster.IgnoreBlobPrice = true

	// Build L1 + L2
	cleanup := builder.Build(t)
	defer cleanup()
	// Create 2nd node to verify batch gets there
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	l1HeightBeforeBatch, err := builder.L1.Client.BlockNumber(ctx)
	require.NoError(t, err)

	// Do some L2 action (to become the batch content)
	tx := builder.L2Info.PrepareTx("Faucet", "Owner", builder.L2Info.TransferGas, common.Big1, nil)
	_ = builder.L2.SendWaitTestTransactions(t, []*types.Transaction{tx})[0]

	// Advance L1 enough to ensure everything is synced
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	// Wait for the batch to be posted and processed by node B
	_, err = WaitForTx(ctx, testClientB.Client, tx.Hash(), 5*time.Second)
	Require(t, err)

	// We assume that `builder.L1.Client` has the L1 block that made `testClientB.Client` notice `tx`.
	l1HeightAfterBatch, err := builder.L1.Client.BlockNumber(ctx)
	require.NoError(t, err)

	// Look up the batches posted between the two L1 heights
	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, int64(l1HeightBeforeBatch)) // nolint: gosec
	Require(t, err)
	batches, err := seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(l1HeightBeforeBatch), new(big.Int).SetUint64(l1HeightAfterBatch))
	Require(t, err)
	require.NotZero(t, len(batches), "no batches found between L1 blocks %d and %d", l1HeightBeforeBatch, l1HeightAfterBatch)

	for _, batch := range batches {
		sequenceNum := batch.SequenceNumber
		sequencerMessageBytes, _, err := builder.L2.ConsensusNode.InboxReader.GetSequencerMessageBytes(ctx, sequenceNum)
		Require(t, err)

		blobVersionedHash := common.BytesToHash(sequencerMessageBytes[41:])

		l1Block, err := builder.L1.Client.HeaderByNumber(ctx, big.NewInt(int64(batch.ParentChainBlockNumber))) // nolint: gosec
		Require(t, err)
		require.NotZero(t, l1Block.BlobGasUsed)

		restoredBlobs, err := builder.L1.L1BlobReader.GetBlobs(ctx, l1Block.Hash(), []common.Hash{blobVersionedHash})
		Require(t, err)
		require.Len(t, restoredBlobs, 1)
	}
}

func TestBatchPosterPostsReportOnlyBatchAfterMaxEmptyBatchDelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		TakeOwnership()

	// Enable delayed sequencer and set fast finalization so reports appear quickly on L2
	builder.nodeConfig.DelayedSequencer.Enable = true
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1
	// Post an empty batch if no useful messages appear within 1 second.
	builder.nodeConfig.BatchPoster.MaxEmptyBatchDelay = time.Second
	// Use a very short non-zero max delay to trigger first batch immediately.
	builder.nodeConfig.BatchPoster.MaxDelay = time.Millisecond
	// Disable automatic background posting
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour

	cleanup := builder.Build(t)
	defer cleanup()

	// Prevent background batchposter goroutine from racing our manual call.
	builder.L2.ConsensusNode.BatchPoster.StopAndWait()

	initialBatchCount := GetBatchCount(t, builder)

	// Force immediate post of first batch.
	posted, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	require.NoError(t, err)
	require.True(t, posted, "expected first batch to post immediately")

	// Wait for batch to appear on L1
	require.Eventually(t, func() bool {
		return GetBatchCount(t, builder) == initialBatchCount+1
	}, 3*time.Second, 100*time.Millisecond, "timeout waiting for first batch to appear on L1")

	// Force second batch, should not post yet as MaxEmptyBatchDelay hasn't elapsed
	posted, err = builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	require.NoError(t, err)
	require.False(t, posted, "expected second batch to not post yet")

	// Spin L1 to get batch poster report
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 1)

	// Wait for the delayed message's timestamp to become old enough to trigger MaxEmptyBatchDelay.
	// The batch posting report comes back as a delayed message with an L1 block timestamp.
	// L1 block timestamps can be up to ~12 seconds ahead of wall clock time (Ethereum PoS block interval).
	// We need to wait for:
	// 1. MaxEmptyBatchDelay (1 second) - the configured threshold that triggers batch posting
	// 2. ~12-13 seconds - to ensure the L1 block timestamp is in the past relative to time.Now()
	// 3. Extra buffer - for the delayed sequencer to process and make the report available
	time.Sleep(builder.nodeConfig.BatchPoster.MaxEmptyBatchDelay + 15*time.Second)

	// Force second batch, posting should be triggered by MaxEmptyBatchDelay
	posted, err = builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	require.NoError(t, err)
	require.True(t, posted, "expected second batch to be posted by MaxEmptyBatchDelay")
}
