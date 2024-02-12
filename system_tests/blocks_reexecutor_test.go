package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	blocksreexecutor "github.com/offchainlabs/nitro/blocks_reexecutor"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

func TestBlocksReExecutorModes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	execConfig := gethexec.ConfigDefaultTest()
	Require(t, execConfig.Validate())
	l2info, stack, chainDb, arbDb, blockchain := createL2BlockChain(t, nil, t.TempDir(), params.ArbitrumDevTestChainConfig(), &execConfig.Caching)

	execConfigFetcher := func() *gethexec.Config { return execConfig }
	execNode, err := gethexec.CreateExecutionNode(ctx, stack, chainDb, blockchain, nil, execConfigFetcher)
	Require(t, err)

	parentChainID := big.NewInt(1234)
	feedErrChan := make(chan error, 10)
	node, err := arbnode.CreateNode(ctx, stack, execNode, arbDb, NewFetcherFromConfig(arbnode.ConfigDefaultL2Test()), blockchain.Config(), nil, nil, nil, nil, nil, feedErrChan, parentChainID)
	Require(t, err)
	err = node.TxStreamer.AddFakeInitMessage()
	Require(t, err)
	Require(t, node.Start(ctx))
	client := ClientForStack(t, stack)

	l2info.GenerateAccount("User2")
	for i := 0; i < 100; i++ {
		tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, common.Big1, nil)
		err := client.SendTransaction(ctx, tx)
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
		if have, want := receipt.BlockNumber.Uint64(), uint64(i)+1; have != want {
			Fatal(t, "internal test error - tx got included in unexpected block number, have:", have, "want:", want)
		}
	}

	success := make(chan struct{})

	// Reexecute blocks at mode full
	go func() {
		executorFull := blocksreexecutor.New(&blocksreexecutor.TestConfig, blockchain, feedErrChan)
		executorFull.StopWaiter.Start(ctx, executorFull)
		executorFull.Impl(ctx)
		executorFull.StopAndWait()
		success <- struct{}{}
	}()
	select {
	case err := <-feedErrChan:
		t.Errorf("error occurred: %v", err)
		if node != nil {
			node.StopAndWait()
		}
		t.FailNow()
	case <-success:
	}

	// Reexecute blocks at mode random
	go func() {
		c := &blocksreexecutor.TestConfig
		c.Mode = "random"
		executorRandom := blocksreexecutor.New(c, blockchain, feedErrChan)
		executorRandom.StopWaiter.Start(ctx, executorRandom)
		executorRandom.Impl(ctx)
		executorRandom.StopAndWait()
		success <- struct{}{}
	}()
	select {
	case err := <-feedErrChan:
		t.Errorf("error occurred: %v", err)
		if node != nil {
			node.StopAndWait()
		}
		t.FailNow()
	case <-success:
	}
}
