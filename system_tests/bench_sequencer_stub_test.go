//go:build !benchsequencer

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestBenchSequencerStub(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.execConfig.Dangerous.BenchSequencer.Enable = true

	err := builder.execConfig.Validate()
	if err == nil {
		t.Fatal("validation of execution config with BenchSequencer enabled should have failed in production build")
	}

	// bypass the execution config validation to validate that stubs don't change node behaviour
	builder = builder.IgnoreExecConfigValidationError()
	cleanup := builder.Build(t)
	defer cleanup()

	// check benchseq rpc is not available
	rpcClient := builder.L2.Client.Client()
	var txQueueLen int
	err = rpcClient.CallContext(ctx, &txQueueLen, "benchseq_txQueueLength")
	if err == nil {
		Fatal(t, "benchseq_txQueueLength should not have succeeded")
	} else if !strings.Contains(err.Error(), "the method benchseq_txQueueLength does not exist") {
		Fatal(t, "benchseq_txQueueLength failed with unexpected error:", err)
	}
	var blockCreated bool
	// create block with all of the transactions (they should fit)
	err = rpcClient.CallContext(ctx, &blockCreated, "benchseq_createBlock")
	if err == nil {
		Fatal(t, "benchseq_createBlock should not have succeeded")
	} else if !strings.Contains(err.Error(), "the method benchseq_createBlock does not exist") {
		Fatal(t, "benchseq_createBlock failed with unexpected error:", err)
	}

	// check that blocks are created automatically
	startBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	tx := builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, big.NewInt(1), nil)
	builder.L2.SendWaitTestTransactions(t, types.Transactions{tx})
	block, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	timeout := time.After(5 * time.Second)
	for block <= startBlock {
		select {
		case <-timeout:
			Fatal(t, "timeout exceeded while waiting for new block")
		case <-time.After(20 * time.Millisecond):
		}
		block, err = builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
	}
}
