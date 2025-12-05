package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

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
	feedErrChan := make(chan error, 10)

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
	executorFull, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ChainDB, feedErrChan)
	Require(t, err)
	success := make(chan struct{})
	executorFull.Start(ctx, success)
	select {
	case err := <-feedErrChan:
		t.Fatalf("error occurred: %v", err)
	case <-success:
	}

	// Reexecute blocks at mode random
	c.Mode = "random"
	c.MinBlocksPerThread = 20
	Require(t, c.Validate())
	executorRandom, err := blocksreexecutor.New(&c, blockchain, builder.L2.ExecNode.ChainDB, feedErrChan)
	Require(t, err)
	success = make(chan struct{})
	executorRandom.Start(ctx, success)
	select {
	case err := <-feedErrChan:
		t.Fatalf("error occurred: %v", err)
	case <-success:
	}
}
