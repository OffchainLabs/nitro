// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/util"
)

func TestSnapSync(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs

	// 1st node with sequencer, stays up all the time.
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info = NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(builder.chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	cleanup := builder.Build(t)
	defer cleanup()

	// 2nd node without sequencer, syncs up to the first node.
	// This node will be stopped in middle and arbitrumdata will be deleted.
	testDir := t.TempDir()
	nodeBStack := createStackConfigForTest(testDir)
	nodeB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack})

	builder.BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)))

	builder.L2Info.GenerateAccount("BackgroundUser")

	// Create transactions till batch count is 10
	createTransactionTillBatchCount(ctx, t, builder, 10)
	// Wait for nodeB to sync up to the first node
	waitForBlocksToCatchup(ctx, t, builder.L2.Client, nodeB.Client)

	// Create a config with snap sync enabled and same database directory as the 2nd node
	nodeConfig := createNodeConfigWithSnapSync(t, builder)
	// Cleanup the message data of 2nd node, but keep the block state data.
	// This is to simulate a snap sync environment where we’ve just gotten the block state but don’t have any messages.
	err := os.RemoveAll(nodeB.ConsensusNode.Stack.ResolvePath("arbitrumdata"))
	Require(t, err)

	// Cleanup the 2nd node to release the database lock
	cleanupB()
	// New node with snap sync enabled, and the same database directory as the 2nd node but with no message data.
	nodeC, cleanupC := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack, nodeConfig: nodeConfig})
	defer cleanupC()

	// Create transactions till batch count is 20
	createTransactionTillBatchCount(ctx, t, builder, 20)
	// Wait for nodeB to sync up to the first node
	waitForBatchCountToCatchup(ctx, t, builder.L2.ConsensusNode.InboxTracker, nodeC.ConsensusNode.InboxTracker)
	// Once the node is synced up, check if the batch metadata is the same for the last batch
	// This is to ensure that the snap sync worked correctly
	count, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
	Require(t, err)
	metadata, err := builder.L2.ConsensusNode.InboxTracker.GetBatchMetadata(count - 1)
	Require(t, err)
	metadataNodeC, err := nodeC.ConsensusNode.InboxTracker.GetBatchMetadata(count - 1)
	Require(t, err)
	if metadata != metadataNodeC {
		t.Error("Batch metadata mismatch")
	}
	finalMessageCount := uint64(metadata.MessageCount)
	waitForBlockToCatchupToMessageCount(ctx, t, builder.L2.Client, finalMessageCount)
	waitForBlockToCatchupToMessageCount(ctx, t, nodeC.Client, finalMessageCount)
	// Fetching message count - 1 instead on the latest block number as the latest block number might not be
	// present in the snap sync node since it does not have the sequencer feed.
	header, err := builder.L2.Client.HeaderByNumber(ctx, big.NewInt(int64(finalMessageCount)-1))
	Require(t, err)
	headerNodeC, err := nodeC.Client.HeaderByNumber(ctx, big.NewInt(int64(finalMessageCount)-1))
	Require(t, err)
	// Once the node is synced up, check if the block hash is the same for the last block
	// This is to ensure that the snap sync worked correctly
	if header.Hash().Cmp(headerNodeC.Hash()) != 0 {
		t.Error("Block hash mismatch")
	}
}

func waitForBlockToCatchupToMessageCount(
	ctx context.Context,
	t *testing.T,
	client *ethclient.Client,
	finalMessageCount uint64,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Millisecond):
			latestHeaderNodeC, err := client.HeaderByNumber(ctx, nil)
			Require(t, err)
			if latestHeaderNodeC.Number.Uint64() >= uint64(finalMessageCount)-1 {
				return
			}
		}
	}
}

func waitForBlocksToCatchup(ctx context.Context, t *testing.T, clientA *ethclient.Client, clientB *ethclient.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Millisecond):
			headerA, err := clientA.HeaderByNumber(ctx, nil)
			Require(t, err)
			headerB, err := clientB.HeaderByNumber(ctx, nil)
			Require(t, err)
			if headerA.Number.Cmp(headerB.Number) == 0 {
				return
			}
		}
	}
}

func waitForBatchCountToCatchup(ctx context.Context, t *testing.T, inboxTrackerA *arbnode.InboxTracker, inboxTrackerB *arbnode.InboxTracker) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Millisecond):
			countA, err := inboxTrackerA.GetBatchCount()
			Require(t, err)
			countB, err := inboxTrackerB.GetBatchCount()
			Require(t, err)
			if countA == countB {
				return
			}
		}

	}
}

func createTransactionTillBatchCount(ctx context.Context, t *testing.T, builder *NodeBuilder, finalCount uint64) {
	for {
		Require(t, ctx.Err())
		tx := builder.L2Info.PrepareTx("Faucet", "BackgroundUser", builder.L2Info.TransferGas, big.NewInt(1), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		count, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
		Require(t, err)
		if count > finalCount {
			break
		}
	}
}

func createNodeConfigWithSnapSync(t *testing.T, builder *NodeBuilder) *arbnode.Config {
	batchCount, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
	Require(t, err)
	// Last batch is batchCount - 1, so prev batch is batchCount - 2
	prevBatchMetaData, err := builder.L2.ConsensusNode.InboxTracker.GetBatchMetadata(batchCount - 2)
	Require(t, err)
	prevMessage, err := builder.L2.ConsensusNode.TxStreamer.GetMessage(prevBatchMetaData.MessageCount - 1)
	Require(t, err)
	// Create a config with snap sync enabled and same database directory as the 2nd node
	nodeConfig := builder.nodeConfig
	nodeConfig.SnapSyncTest.Enabled = true
	nodeConfig.SnapSyncTest.BatchCount = batchCount
	nodeConfig.SnapSyncTest.DelayedCount = prevBatchMetaData.DelayedMessageCount - 1
	nodeConfig.SnapSyncTest.PrevDelayedRead = prevMessage.DelayedMessagesRead
	nodeConfig.SnapSyncTest.PrevBatchMessageCount = uint64(prevBatchMetaData.MessageCount)
	return nodeConfig
}
