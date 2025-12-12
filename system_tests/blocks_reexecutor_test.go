package arbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie"

	blocksreexecutor "github.com/offchainlabs/nitro/blocks_reexecutor"
)

func TestBlocksReExecutorModes(t *testing.T) {
	testBlocksReExecutorModes(t, false)
}

func TestBlocksReExecutorMultipleRanges(t *testing.T) {
	testBlocksReExecutorModes(t, true)
}

func testBlocksReExecutorModes(t *testing.T, onMultipleRanges bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	// For now PathDB is not supported
	builder.RequireScheme(t, rawdb.HashScheme)

	// This allows us to see reexecution of multiple ranges
	if onMultipleRanges {
		builder.execConfig.Caching.Archive = true
	}
	cleanup := builder.Build(t)
	defer cleanup()

	l2info := builder.L2Info
	client := builder.L2.Client
	blockchain := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()

	l2info.GenerateAccount("User2")
	genesis, err := client.BlockNumber(ctx)
	Require(t, err)
	for i := genesis; i < genesis+100; i++ {
		tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, common.Big1, nil)
		err := client.SendTransaction(ctx, tx)
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
		if have, want := receipt.BlockNumber.Uint64(), uint64(i)+1; have != want {
			Fatal(t, "internal test error - tx got included in unexpected block number, have:", have, "want:", want)
		}
	}

	// Set Blocks config field if running blocks reexecution on multiple ranges
	c := blocksreexecutor.TestConfig
	c.ValidateMultiGas = true
	if onMultipleRanges {
		c.Blocks = `[[0, 29], [30, 59], [60, 99]]`
	}

	// Reexecute blocks at mode full
	c.MinBlocksPerThread = 10
	Require(t, c.Validate())
	executorFull, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ChainDB)
	Require(t, err)
	executorFull.Start(ctx)
	err = executorFull.WaitForReExecution(ctx)
	Require(t, err)

	// Reexecute blocks at mode random
	c.Mode = "random"
	c.MinBlocksPerThread = 20
	Require(t, c.Validate())
	executorRandom, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ChainDB)
	Require(t, err)
	executorRandom.Start(ctx)
	err = executorFull.WaitForReExecution(ctx)
	Require(t, err)
}

func assertStateExistForBlockRange(t *testing.T, bc *core.BlockChain, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		_, err := bc.StateAt(header.Root)
		Require(t, err)
	}
}

func assertMissingStateForBlockRange(t *testing.T, bc *core.BlockChain, from, to uint64) {
	t.Helper()
	expectedErr := &trie.MissingNodeError{}
	for blockNum := from; blockNum <= to; blockNum++ {
		header := bc.GetHeaderByNumber(blockNum)
		_, err := bc.StateAt(header.Root)
		if err == nil {
			Fatal(t, "expeted StateAt to fail for blockNumber:", header.Number)
		}
		if !errors.As(err, &expectedErr) {
			Fatal(t, "getting state failed with unexpected error, root:", header.Root, "blockNumber:", header.Number, "err:", err)
		}
	}
}

func TestBlocksReExecutorCommitState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithDatabase(rawdb.DBPebble)
	// For now PathDB is not supported
	builder.RequireScheme(t, rawdb.HashScheme)

	// 1. Setup builder to be run in sparse archive mode
	builder.execConfig.Caching.Archive = true
	builder.execConfig.Caching.SnapshotCache = 0 // disable snapshots
	builder.execConfig.Caching.BlockAge = 0
	builder.execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 100
	builder.execConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0

	maxRecreateStateDepth := int64(100 * 1000 * 1000)

	builder.execConfig.RPC.MaxRecreateStateDepth = maxRecreateStateDepth
	builder.execConfig.Sequencer.MaxBlockSpeed = 0
	builder.execConfig.Sequencer.MaxTxDataSize = 150

	cleanup := builder.Build(t)
	defer cleanup()

	l2info := builder.L2Info
	client := builder.L2.Client
	blockchain := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()

	// 2. Create about 500 blocks
	l2info.GenerateAccount("User2")
	genesis, err := client.BlockNumber(ctx)
	Require(t, err)
	for i := genesis; i < genesis+500; i++ {
		tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, common.Big1, nil)
		err := client.SendTransaction(ctx, tx)
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
		if have, want := receipt.BlockNumber.Uint64(), uint64(i)+1; have != want {
			Fatal(t, "internal test error - tx got included in unexpected block number, have:", have, "want:", want)
		}
	}

	bc := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()

	// 3. Assert that some blocks are missing. We can't be too granular here since
	// statedb might commit to triedb if dirty cache size limit is exhausted
	assertMissingStateForBlockRange(t, bc, 5, 90)
	assertMissingStateForBlockRange(t, bc, 110, 190)
	assertMissingStateForBlockRange(t, bc, 210, 290)
	assertMissingStateForBlockRange(t, bc, 310, 370)

	// 4. We first run BlocksReExecutor with ValidateMultiGas set to false to make sure
	// BlocksReExecutor does not commit state to disk
	c := blocksreexecutor.TestConfig
	c.ValidateMultiGas = true
	c.Blocks = `[[0, 42], [110, 160], [180, 200]]`
	// We don't need to explicit set it to false since default is false, but we want to be explicit
	c.CommitStateToDisk = false

	// Reexecute blocks at mode full
	c.MinBlocksPerThread = 10
	Require(t, c.Validate())
	executorFull, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ChainDB)
	Require(t, err)
	executorFull.Start(ctx)
	err = executorFull.WaitForReExecution(ctx)
	Require(t, err)

	// 5. Now that we have run Block Re-executor CommitStateToDisk set to false
	// we should expect state to NOT be present for those same blocks
	assertMissingStateForBlockRange(t, bc, 5, 90)
	assertMissingStateForBlockRange(t, bc, 110, 190)
	assertMissingStateForBlockRange(t, bc, 210, 290)
	assertMissingStateForBlockRange(t, bc, 310, 370)

	// 6. Now we run BlocksReExecutor with ValidateMultiGas set to true to make sure
	// BlocksReExecutor does indeed commit state of c.Blocks to disk.
	// We don't set c.Blocks since we want to use the same blocks range
	c.CommitStateToDisk = true
	Require(t, c.Validate())
	executorFullCommit, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ChainDB)
	Require(t, err)
	executorFullCommit.Start(ctx)
	err = executorFullCommit.WaitForReExecution(ctx)
	Require(t, err)

	// 6. Now that we have run Block Re-executor with CommitStateToDisk set to true
	// we should expect state to be present for the ranges we set for c.Blocks
	assertStateExistForBlockRange(t, bc, 5, 40)
	assertStateExistForBlockRange(t, bc, 110, 200)

	// 7. Finally, make sure we haven't commited state for blocks not specified in c.Blocks range
	assertMissingStateForBlockRange(t, bc, 45, 90)
	assertMissingStateForBlockRange(t, bc, 210, 290)
	assertMissingStateForBlockRange(t, bc, 310, 370)
}
