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

	// 2nd node without sequencer, syncs up to the first node.
	// This node will be stopped in middle.
	testDir := t.TempDir()
	nodeBStack := testhelpers.CreateStackConfigForTest(testDir)
	nodeBStack.DBEngine = databaseEngine
	nodeBConfig := builder.nodeConfig
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

	// Create a config with revalidation range and same database directory as the 2nd node
	nodeConfig := createNodeConfigWithRevalidationRange(builder)

	// Cleanup the 2nd node to release the database lock
	cleanupB()
	// New node with revalidation range, and the same database directory as the 2nd node.
	nodeC, cleanupC := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack, nodeConfig: nodeConfig})
	defer cleanupC()

	// Wait for the node to start and revalidate the blocks in the specified range
	// Once the revalidation is done, the validator will stop.
	startTime := time.Now()
	for {
		if nodeC.ConsensusNode.BlockValidator.Stopped() {
			break
		} else if time.Since(startTime) > 5*time.Minute {
			t.Fatalf("Revalidation took too long")
		}
	}
}

func createNodeConfigWithRevalidationRange(builder *NodeBuilder) *arbnode.Config {
	nodeConfig := *builder.nodeConfig
	nodeConfig.BlockValidator.Dangerous.Revalidation.StartBlock = 5
	nodeConfig.BlockValidator.Dangerous.Revalidation.EndBlock = 10
	return &nodeConfig
}

// waitForBlocksToCatchup has a time "limit" factor to limit running this function forever in weird cases such as running with race detection in nightly CI
func waitForBlocksToCatchup(ctx context.Context, t *testing.T, clientA *ethclient.Client, clientB *ethclient.Client, limit time.Duration) {
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
		case <-time.After(limit):
			t.Fatal("waitForBlocksToCatchup didnt finish")
		}
	}
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
		count, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
		Require(t, err)
		if count > finalCount {
			return
		}
		time.Sleep(100 * time.Millisecond) // give some time for other components (reader/tracker) to read the batches from L1
	}
	t.Fatal("createTransactionTillBatchCount didnt finish")
}
