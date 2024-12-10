// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestLargeReorg(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.nodeConfig.TransactionStreamer.MaxReorgResequenceDepth = 1_000_000_000
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	_, simple := builder.L2.DeploySimple(t, auth)

	startMsgCount, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)

	checkCounter := func(expected int64, scenario string) {
		t.Helper()
		counter, err := simple.Counter(&bind.CallOpts{Context: ctx})
		Require(t, err)
		if counter != uint64(expected) {
			Fatal(t, scenario, "counter was", counter, "expected", expected)
		}
	}

	expectedCounter := int64(500)
	var tx *types.Transaction
	for i := int64(0); i < expectedCounter; i++ {
		tx, err = simple.LogAndIncrement(&auth, big.NewInt(1))
		Require(t, err, "failed to call Increment()")
		if (i+1)%100 == 0 {
			_, err = builder.L2.EnsureTxSucceeded(tx)
			Require(t, err)
		}
	}

	checkCounter(expectedCounter, "pre reorg")

	err = builder.L2.ConsensusNode.TxStreamer.ReorgTo(startMsgCount)
	Require(t, err)
	// Old messages are replayed asynchronously so we must wait for them to catch up.
	_, err = builder.L2.ExecNode.ExecEngine.HeadMessageNumberSync(t)
	Require(t, err)
	checkCounter(expectedCounter, "post reorg")
}
