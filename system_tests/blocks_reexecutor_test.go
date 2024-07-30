package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	blocksreexecutor "github.com/offchainlabs/nitro/blocks_reexecutor"
)

func TestBlocksReExecutorModes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
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

	// Reexecute blocks at mode full
	success := make(chan struct{})
	executorFull := blocksreexecutor.New(&blocksreexecutor.TestConfig, blockchain, feedErrChan)
	executorFull.Start(ctx, success)
	select {
	case err := <-feedErrChan:
		t.Fatalf("error occurred: %v", err)
	case <-success:
	}

	// Reexecute blocks at mode random
	success = make(chan struct{})
	c := &blocksreexecutor.TestConfig
	c.Mode = "random"
	executorRandom := blocksreexecutor.New(c, blockchain, feedErrChan)
	executorRandom.Start(ctx, success)
	select {
	case err := <-feedErrChan:
		t.Fatalf("error occurred: %v", err)
	case <-success:
	}
}
