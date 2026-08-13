// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestRevalidationForSpecifiedRange(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs

	// 1st node with sequencer, stays up all the time.
	databaseEngine := rawdb.DBPebble
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).DontParalellise().WithDatabase(databaseEngine)
	builder.nodeConfig.BlockValidator.Enable = true
	builder.L2Info = NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(builder.chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	cleanup := builder.Build(t)
	defer cleanup()

	// 2nd node without sequencer, syncs up to the first node and validates it.
	// This node will be stopped in middle.
	testDir := t.TempDir()
	nodeBStack := testhelpers.CreateStackConfigForTest(testDir)
	nodeBStack.DBEngine = databaseEngine
	nodeBConfig := builder.nodeConfig
	nodeBConfig.BlockValidator.Enable = true
	nodeBConfig.BatchPoster.Enable = false
	nodeBParams := &SecondNodeParams{
		stackConfig: nodeBStack,
		nodeConfig:  nodeBConfig,
	}
	nodeB, cleanupB := builder.Build2ndNode(t, nodeBParams)

	builder.BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000000)))

	builder.L2Info.GenerateAccount("BackgroundUser")

	// Create transactions till batch count is 15
	createTransactionTillBatchCount(ctx, t, builder, 15)
	// Wait for nodeB to sync up to the first node
	waitForBlocksToCatchup(ctx, t, builder.L2.Client, nodeB.Client, 10*time.Minute)

	// nodeB has to validate the range before it can be revalidated: revalidation only
	// ever moves the last validated state backwards.
	lastBlock, err := nodeB.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	// message index is the same as the block number here
	if !nodeB.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(lastBlock.NumberU64()), 5*time.Minute) {
		Fatal(t, "nodeB did not validate up to the chain head")
	}

	// Create a config with revalidation range and same database directory as the 2nd node
	nodeConfig := createNodeConfigWithRevalidationRange(builder)

	// Cleanup the 2nd node to release the database lock
	cleanupB()
	// New node with revalidation range, and the same database directory as the 2nd node.
	nodeConfig.BlockValidator.Enable = true
	nodeC, cleanupC := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack, nodeConfig: nodeConfig})
	defer cleanupC()

	// Wait for the node to start and revalidate the blocks in the specified range
	// Once the revalidation is done, the validator will stop.
	pollUntil(t, ctx, 5*time.Minute, 100*time.Millisecond, "revalidation to complete", func() bool {
		return nodeC.ConsensusNode.BlockValidator.Stopped()
	})
}

// A revalidation start batch that doesn't exist should not stop the node from starting:
// it logs an error and keeps validating from wherever it left off.
func TestRevalidationStartBatchDoesNotExist(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.Dangerous.Revalidation.StartBatch = 1_000_000
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	_, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	lastBlock, err := builder.L2.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	// message index is the same as the block number here
	if !builder.L2.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(lastBlock.NumberU64()), time.Minute*2) {
		Fatal(t, "validation did not progress past the bad revalidation start batch")
	}
}

// Revalidation only moves backwards. A start batch that exists but is ahead of the
// last validated state would mark everything in between as valid without validating
// it, so it is refused and the node validates normally instead.
func TestRevalidationDoesNotMoveValidationForward(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	// 2nd node syncs the batches without validating any of them, so its last
	// validated state stays at genesis while its inbox tracker moves ahead.
	nodeBStack := testhelpers.CreateStackConfigForTest(t.TempDir())
	nodeB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack})

	builder.L2Info.GenerateAccount("BackgroundUser")
	createTransactionTillBatchCount(ctx, t, builder, 3)
	waitForBlocksToCatchup(ctx, t, builder.L2.Client, nodeB.Client, time.Minute)
	// Cleanup the 2nd node to release the database lock
	cleanupB()

	// Restart on the same database, asking to start from a batch that exists but is
	// ahead of the last validated state.
	nodeConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfig.MessageExtraction.Enable = builder.nodeConfig.MessageExtraction.Enable
	nodeConfig.BlockValidator.Enable = true
	nodeConfig.BlockValidator.Dangerous.Revalidation.StartBatch = 2
	nodeC, cleanupC := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack, nodeConfig: nodeConfig})
	defer cleanupC()

	lastBlock, err := nodeC.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	// message index is the same as the block number here
	if !nodeC.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(lastBlock.NumberU64()), time.Minute) {
		Fatal(t, "validation did not progress after the refused revalidation range")
	}
}

func createNodeConfigWithRevalidationRange(builder *NodeBuilder) *arbnode.Config {
	nodeConfig := *builder.nodeConfig
	nodeConfig.BlockValidator.Dangerous.Revalidation.StartBatch = 5
	nodeConfig.BlockValidator.Dangerous.Revalidation.EndBatch = 10
	return &nodeConfig
}

// waitForBlocksToCatchup polls until both clients report the same latest block number, or the limit elapses.
func waitForBlocksToCatchup(ctx context.Context, t *testing.T, clientA *ethclient.Client, clientB *ethclient.Client, limit time.Duration) {
	t.Helper()
	pollUntil(t, ctx, limit, 10*time.Millisecond, "blocks to catch up between nodes", func() bool {
		headerA, err := clientA.HeaderByNumber(ctx, nil)
		if err != nil {
			t.Logf("HeaderByNumber(A) error (will retry): %v", err)
			return false
		}
		headerB, err := clientB.HeaderByNumber(ctx, nil)
		if err != nil {
			t.Logf("HeaderByNumber(B) error (will retry): %v", err)
			return false
		}
		return headerA.Number.Cmp(headerB.Number) == 0
	})
}

func createTransactionTillBatchCount(ctx context.Context, t *testing.T, builder *NodeBuilder, finalCount uint64) {
	// We run the loop for 6000 iterations ~ maximum of 10 minutes of run time before failing. This is to avoid
	// running this function forever in weird cases such as running with race detection in nightly CI
	for i := uint64(0); i < 6000; i++ {
		Require(t, ctx.Err())
		tx := builder.L2Info.PrepareTx("Faucet", "BackgroundUser", builder.L2Info.TransferGas, big.NewInt(1), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		count, err := builder.L2.ConsensusNode.GetParentChainDataSource().GetBatchCount()
		Require(t, err)
		if count > finalCount {
			return
		}
		time.Sleep(100 * time.Millisecond) // give some time for other components (reader/tracker) to read the batches from L1
	}
	t.Fatal("createTransactionTillBatchCount didnt finish")
}
