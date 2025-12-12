package arbtest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/rpc"
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

	// 3. Set 2 blocks as target block tests
	bc := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()
	blockNum35 := 35
	block35 := bc.GetBlockByNumber(uint64(blockNum35))
	blockRoot35 := block35.Root()

	blockNum150 := 150
	block150 := bc.GetBlockByNumber(uint64(blockNum150))
	blockRoot150 := block150.Root()

	// 4. Since we started L1 as a sparse archive we should only expect states to be
	// present for blocks 100, 200, 300, etc.
	_, err = bc.StateAt(blockRoot35)
	expectedErr := &trie.MissingNodeError{}
	if !errors.As(err, &expectedErr) {
		Fatal(t, "getting state failed with unexpected error, root:", block35.Root(), "blockNumber:", blockNum35, "blockHash:", block35.Hash(), "err:", err)
	}

	_, err = bc.StateAt(blockRoot150)
	expectedErr = &trie.MissingNodeError{}
	if !errors.As(err, &expectedErr) {
		Fatal(t, "getting state failed with unexpected error, root:", block150.Root(), "blockNumber:", blockNum150, "blockHash:", block150.Hash(), "err:", err)
	}

	blockTraceConfig := map[string]interface{}{"tracer": "callTracer"}
	blockTraceConfig["tracerConfig"] = map[string]interface{}{"onlyTopCall": false}

	l2rpc := builder.L2.Stack.Attach()
	var blockTrace json.RawMessage
	err = l2rpc.CallContext(ctx, &blockTrace, "debug_traceBlockByNumber", rpc.BlockNumber(blockNum150), blockTraceConfig)
	Require(t, err)

	// 5. We first run BlocksReExecutor with ValidateMultiGas set to false to make sure
	// BlocksReExecutor does not commit state to disk
	c := blocksreexecutor.TestConfig
	c.ValidateMultiGas = true
	c.Blocks = `[[0, 42], [90, 160], [180, 200]]`
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

	// 6. Now that we have run Block Re-executor CommitStateToDisk set to false
	// we should expect state to NOT be present for both of the above blocks
	_, err = bc.StateAt(blockRoot35)
	if !errors.As(err, &expectedErr) {
		Fatal(t, "getting state failed with unexpected error, root:", block35.Root(), "blockNumber:", blockNum35, "blockHash:", block35.Hash(), "err:", err)
	}

	_, err = bc.StateAt(blockRoot150)
	expectedErr = &trie.MissingNodeError{}
	if !errors.As(err, &expectedErr) {
		Fatal(t, "getting state failed with unexpected error, root:", block150.Root(), "blockNumber:", blockNum150, "blockHash:", block150.Hash(), "err:", err)
	}

	// 7. Now we run BlocksReExecutor with ValidateMultiGas set to true to make sure
	// BlocksReExecutor does indeed commit state of c.Blocks to disk.
	// We don't set c.Blocks since we want to use the same blocks range
	c.CommitStateToDisk = true
	Require(t, c.Validate())
	executorFullCommit, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ChainDB)
	Require(t, err)
	executorFullCommit.Start(ctx)
	err = executorFullCommit.WaitForReExecution(ctx)
	Require(t, err)

	// 8. Now that we have run Block Re-executor with CommitStateToDisk set to true
	// we should expect state to be present for both of the above blocks
	_, err = bc.StateAt(blockRoot35)
	Require(t, err)

	_, err = bc.StateAt(blockRoot150)
	Require(t, err)
}
