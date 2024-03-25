package arbtest

import (
	"context"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
)

func TestA(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup Node A with sequencer
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()
	startL1Block, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Setup Node B without sequencer
	nodeBStackConfig := builder.l2StackConfig
	// Set the data dir manually, since we want to reuse the same data dir for node C
	nodeBStackConfig.DataDir = t.TempDir()
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{
		stackConfig: nodeBStackConfig,
	})
	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	builder.L2Info.GenerateAccount("BackgroundUser")
	tx := builder.L2Info.PrepareTx("Faucet", "BackgroundUser", builder.L2Info.TransferGas, balance, nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Make a bunch of L2 transactions, till we have at least 10 batches
	cleanupFirstRoundBackgroundTx := startBackgroundTxs(ctx, builder)
	var firstRoundBatches []*arbnode.SequencerInboxBatch
	for ctx.Err() == nil && len(firstRoundBatches) < 10 {
		endL1Block, err := builder.L1.Client.BlockNumber(ctx)
		Require(t, err)
		seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.L2.ConsensusNode.DeployInfo.SequencerInbox, 0)
		Require(t, err)
		firstRoundBatches, err = seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(startL1Block), new(big.Int).SetUint64(endL1Block))
		Require(t, err)
	}
	// Stop making background transactions
	cleanupFirstRoundBackgroundTx()

	// Make sure the sequencer has processed all the transactions
	time.Sleep(1 * time.Second)

	// Make sure node B has synced up to the sequencer
	for ctx.Err() == nil {
		endL1Block, err := builder.L1.Client.BlockNumber(ctx)
		Require(t, err)
		seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.L2.ConsensusNode.DeployInfo.SequencerInbox, 0)
		Require(t, err)
		firstRoundBatches, err = seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(startL1Block), new(big.Int).SetUint64(endL1Block))
		Require(t, err)
		batchCount, err := testClientB.ConsensusNode.InboxTracker.GetBatchCount()
		Require(t, err)
		if uint64(len(firstRoundBatches)) > batchCount {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	// Stop node B
	cleanupB()

	// Make a bunch of L2 transactions, till we have at least 10 more batches, since we stopped node B
	cleanupSecondRoundBackgroundTx := startBackgroundTxs(ctx, builder)
	var secondRoundBatches []*arbnode.SequencerInboxBatch
	for ctx.Err() == nil && len(secondRoundBatches) < len(firstRoundBatches)+10 {
		endL1Block, err := builder.L1.Client.BlockNumber(ctx)
		Require(t, err)
		seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.L2.ConsensusNode.DeployInfo.SequencerInbox, 0)
		Require(t, err)
		secondRoundBatches, err = seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(startL1Block), new(big.Int).SetUint64(endL1Block))
		Require(t, err)
	}
	// Stop making background transactions
	cleanupSecondRoundBackgroundTx()

	// Setup Node B without sequencer, while using the same data dir as node B
	nodeCConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	// Start node C from the first batch till which node B has synced
	nodeCConfig.InboxReader.FirstBatch = uint64(len(firstRoundBatches))
	// Remove arbitrumdata dir to simulate a fresh start
	err = os.RemoveAll(nodeBStackConfig.ResolvePath("arbitrumdata"))
	Require(t, err)
	testClientC, cleanupC := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig:  nodeCConfig,
		stackConfig: nodeBStackConfig,
	})
	defer cleanupC()
	for {
		batchCount, err := testClientC.ConsensusNode.InboxTracker.GetBatchCount()
		Require(t, err)
		if uint64(len(secondRoundBatches)) > batchCount {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
}

func startBackgroundTxs(ctx context.Context, builder *NodeBuilder) func() {
	// Continually make L2 transactions in a background thread
	backgroundTxsCtx, cancelBackgroundTxs := context.WithCancel(ctx)
	backgroundTxsShutdownChan := make(chan struct{})
	cleanup := func() {
		cancelBackgroundTxs()
		<-backgroundTxsShutdownChan
	}
	go (func() {
		defer close(backgroundTxsShutdownChan)
		err := makeBackgroundTxs(backgroundTxsCtx, builder)
		if !errors.Is(err, context.Canceled) {
			log.Warn("error making background txs", "err", err)
		}
	})()
	return cleanup
}
