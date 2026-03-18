// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie"

	blocksreexecutor "github.com/offchainlabs/nitro/blocks_reexecutor"
)

func TestBlocksReExecutorModes(t *testing.T) {
	testBlocksReExecutorModes(t, rawdb.HashScheme, false)
}

func TestBlocksReExecutorMultipleRanges(t *testing.T) {
	testBlocksReExecutorModes(t, rawdb.HashScheme, true)
}

func TestBlocksReExecutorPathdbModes(t *testing.T) {
	testBlocksReExecutorModes(t, rawdb.PathScheme, false)
}

func TestBlocksReExecutorPathdbMultipleRanges(t *testing.T) {
	testBlocksReExecutorModes(t, rawdb.PathScheme, true)
}

func buildReexecutorTestNode(t *testing.T, ctx context.Context, scheme string, archive bool, blocks uint64) (*NodeBuilder, func()) {
	t.Helper()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	if scheme == rawdb.PathScheme {
		builder = builder.WithDatabase(rawdb.DBPebble)
		builder.TrieNoAsyncFlush = true
	}
	builder.RequireScheme(t, scheme)

	if archive {
		builder.execConfig.Caching.Archive = true
	}

	cleanup := builder.Build(t)

	l2info := builder.L2Info
	client := builder.L2.Client

	l2info.GenerateAccount("User2")
	genesis, err := client.BlockNumber(ctx)
	Require(t, err)
	for i := genesis; i < genesis+blocks; i++ {
		tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, common.Big1, nil)
		err := client.SendTransaction(ctx, tx)
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
		if have, want := receipt.BlockNumber.Uint64(), uint64(i)+1; have != want {
			Fatal(t, "internal test error - tx got included in unexpected block number, have:", have, "want:", want)
		}
	}
	return builder, cleanup
}

func testBlocksReExecutorModes(t *testing.T, scheme string, onMultipleRanges bool) {
	testCases := []struct {
		mode               string
		minBlocksPerThread uint64
	}{
		{mode: "full", minBlocksPerThread: 10},
		{mode: "random", minBlocksPerThread: 20},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			builder, cleanup := buildReexecutorTestNode(t, ctx, scheme, onMultipleRanges, 100)
			defer cleanup()

			blockchain := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()

			c := blocksreexecutor.TestConfig
			c.ValidateMultiGas = true
			c.Mode = tc.mode
			c.MinBlocksPerThread = tc.minBlocksPerThread
			if scheme == rawdb.PathScheme {
				c.CommitStateToDisk = false
			}
			if onMultipleRanges {
				c.Blocks = `[[0, 29], [30, 59], [60, 99]]`
			}

			Require(t, c.Validate())
			executor, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ExecutionDB)
			Require(t, err)
			executor.Start(ctx)
			err = executor.WaitForReExecution(ctx)
			Require(t, err)
		})
	}
}

func assertStateExistForBlockRange(t *testing.T, bc *core.BlockChain, from, offset uint64) {
	t.Helper()
	for i := from; i <= from+offset; i++ {
		header := bc.GetHeaderByNumber(i)
		_, err := bc.StateAt(header.Root)
		Require(t, err)
	}
}

func assertMissingStateForBlockRange(t *testing.T, bc *core.BlockChain, from, offset uint64) {
	t.Helper()
	expectedErr := &trie.MissingNodeError{}
	for blockNum := from; blockNum <= from+offset; blockNum++ {
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
	builder.RequireScheme(t, rawdb.HashScheme)

	maxNumberOfBlocksToSkipStateSaving := uint32(150)

	// 1. Setup builder to be run in sparse archive mode
	builder.execConfig.Caching.Archive = true
	builder.execConfig.Caching.SnapshotCache = 0 // disable snapshots
	builder.execConfig.Caching.BlockAge = 0
	builder.execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = maxNumberOfBlocksToSkipStateSaving
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
	for i := genesis; i < genesis+900; i++ {
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

	// 3. Assert that some blocks are missing in 140 block windows
	offset := uint64(maxNumberOfBlocksToSkipStateSaving - 10)
	assertMissingStateForBlockRange(t, bc, 2, offset)
	assertMissingStateForBlockRange(t, bc, 160, offset)
	assertMissingStateForBlockRange(t, bc, 310, offset)
	assertMissingStateForBlockRange(t, bc, 460, offset)
	assertMissingStateForBlockRange(t, bc, 610, offset)

	// 4. We first run BlocksReExecutor with CommitStateToDisk set to false to make sure
	// BlocksReExecutor does not commit state to disk
	c := blocksreexecutor.TestConfig
	c.ValidateMultiGas = true
	c.Blocks = `[[0, 42], [110, 160], [180, 200], [480, 580]]`
	// We don't need to explicit set it to false since default is false, but we want to be explicit
	c.CommitStateToDisk = false

	// Reexecute blocks at mode full
	c.MinBlocksPerThread = 10
	Require(t, c.Validate())
	executorFull, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ExecutionDB)
	Require(t, err)
	executorFull.Start(ctx)
	err = executorFull.WaitForReExecution(ctx)
	Require(t, err)

	// 5. Now that we have run Block Re-executor CommitStateToDisk set to false
	// we should expect state to NOT be present for those same blocks
	assertMissingStateForBlockRange(t, bc, 2, offset)
	assertMissingStateForBlockRange(t, bc, 160, offset)
	assertMissingStateForBlockRange(t, bc, 310, offset)
	assertMissingStateForBlockRange(t, bc, 460, offset)
	assertMissingStateForBlockRange(t, bc, 610, offset)

	// 6. Now we run BlocksReExecutor with CommitStateToDisk set to true to make sure
	// BlocksReExecutor does indeed commit state of c.Blocks to disk.
	// We don't set c.Blocks since we want to use the same blocks range
	c.CommitStateToDisk = true
	Require(t, c.Validate())
	executorFullCommit, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ExecutionDB)
	Require(t, err)
	executorFullCommit.Start(ctx)
	err = executorFullCommit.WaitForReExecution(ctx)
	Require(t, err)

	// 6. Now that we have run Block Re-executor with CommitStateToDisk set to true
	// we should expect state to be present for the ranges we set for c.Blocks
	assertStateExistForBlockRange(t, bc, 2, 190)
	assertStateExistForBlockRange(t, bc, 460, 120)

	// 7. Finally, make sure we haven't committed state for blocks not specified in c.Blocks range
	assertMissingStateForBlockRange(t, bc, 310, offset)
	assertMissingStateForBlockRange(t, bc, 581, 20)
	assertMissingStateForBlockRange(t, bc, 610, offset)
}

func TestBlocksReExecutorPathdbCommitStateSmoke(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := buildReexecutorTestNode(t, ctx, rawdb.PathScheme, true, 120)
	defer cleanup()

	blockchain := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()

	c := blocksreexecutor.TestConfig
	c.Blocks = `[[15, 60]]`
	c.CommitStateToDisk = true
	c.Room = 2
	c.MinBlocksPerThread = 8
	Require(t, c.Validate())

	executor, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ExecutionDB)
	Require(t, err)
	executor.Start(ctx)
	err = executor.WaitForReExecution(ctx)
	Require(t, err)
}

func TestBlocksReExecutorPathdbConfigOnReopenedNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithDatabase(rawdb.DBPebble)
	builder.RequireScheme(t, rawdb.PathScheme)
	builder.execConfig.Caching.Archive = true
	builder.execConfig.Caching.StateHistory = 0
	builder.TrieNoAsyncFlush = true

	cleanup := builder.Build(t)
	nodeStopped := false
	defer func() {
		if !nodeStopped {
			cleanup()
		}
	}()

	l2info := builder.L2Info
	client := builder.L2.Client
	l2info.GenerateAccount("User2")
	genesis, err := client.BlockNumber(ctx)
	Require(t, err)
	for i := genesis; i < genesis+220; i++ {
		tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, common.Big1, nil)
		err := client.SendTransaction(ctx, tx)
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
		if have, want := receipt.BlockNumber.Uint64(), uint64(i)+1; have != want {
			Fatal(t, "internal test error - tx got included in unexpected block number, have:", have, "want:", want)
		}
	}

	builder.L2.cleanup()
	nodeStopped = true

	_, stack, executionDB, _, blockchain := createNonL1BlockChainWithStackConfig(
		t,
		builder.L2Info,
		builder.dataDir,
		builder.chainConfig,
		builder.arbOSInit,
		builder.initMessage,
		builder.l2StackConfig,
		builder.execConfig,
		builder.TrieNoAsyncFlush,
	)
	defer func() {
		blockchain.Stop()
		requireClose(t, stack)
		builder.ctxCancel()
	}()

	head := blockchain.CurrentBlock().Number.Uint64()
	var anchor uint64
	end := head - 1
	for candidate := uint64(1); candidate+20 < head; candidate++ {
		anchorHeader := blockchain.GetHeaderByNumber(candidate)
		if anchorHeader == nil {
			Fatal(t, "failed to get candidate anchor header")
		}
		_, err = blockchain.StateAt(anchorHeader.Root)
		if err == nil {
			anchor = candidate
			if candidate+120 < end {
				end = candidate + 120
			}
			break
		}
	}
	if anchor == 0 {
		Fatal(t, "failed to find an exact pathdb anchor state on the reopened node")
	}

	c := blocksreexecutor.TestConfig
	c.Mode = "full"
	c.Blocks = fmt.Sprintf("[[%d, %d]]", anchor+1, end)
	c.CommitStateToDisk = false
	c.Room = 16
	c.MinBlocksPerThread = 20000
	Require(t, c.Validate())

	executor, err := blocksreexecutor.New(&c, blockchain, executionDB)
	Require(t, err)

	value := reflect.ValueOf(executor).Elem()
	room := value.FieldByName("room").Int()
	if room != 1 {
		Fatal(t, "expected pathdb reexecutor room to collapse to 1, have:", room)
	}

	executor.Start(ctx)
	err = executor.WaitForReExecution(ctx)
	Require(t, err)
}

func TestBlocksReExecutorPathdbWithoutCommitStateSmoke(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := buildReexecutorTestNode(t, ctx, rawdb.PathScheme, false, 40)
	defer cleanup()

	blockchain := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()

	c := blocksreexecutor.TestConfig
	c.Blocks = `[[5, 20]]`
	c.CommitStateToDisk = false
	c.MinBlocksPerThread = 5
	Require(t, c.Validate())

	executor, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ExecutionDB)
	Require(t, err)
	executor.Start(ctx)
	err = executor.WaitForReExecution(ctx)
	Require(t, err)
}

func TestBlocksReExecutorPathdbIgnoresParallelChunking(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := buildReexecutorTestNode(t, ctx, rawdb.PathScheme, false, 160)
	defer cleanup()

	blockchain := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()

	c := blocksreexecutor.TestConfig
	c.Blocks = `[[20, 140]]`
	c.CommitStateToDisk = false
	c.Room = 8
	c.MinBlocksPerThread = 1
	Require(t, c.Validate())

	executor, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ExecutionDB)
	Require(t, err)

	value := reflect.ValueOf(executor).Elem()
	room := value.FieldByName("room").Int()
	if room != 1 {
		Fatal(t, "expected pathdb reexecutor room to collapse to 1, have:", room)
	}

	blocks := value.FieldByName("blocks")
	if blocks.Len() != 1 {
		Fatal(t, "expected exactly one pathdb work range, have:", blocks.Len())
	}

	work := blocks.Index(0)
	start := work.Index(0).Uint()
	end := work.Index(1).Uint()
	minBlocksPerThread := work.Index(2).Uint()
	if start != 19 || end != 140 {
		Fatal(t, "unexpected pathdb work range, have:", [2]uint64{start, end}, "want:", [2]uint64{19, 140})
	}
	if minBlocksPerThread != end-start {
		Fatal(t, "expected pathdb to use a single forward sweep, have:", minBlocksPerThread, "want:", end-start)
	}
}
