package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestEthSyncing(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info = nil
	cleanup := builder.Build(t)
	defer cleanup()

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	// stop txstreamer so it won't feed execution messages
	testClientB.ConsensusNode.TxStreamer.StopAndWait()

	countBefore, err := testClientB.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)

	builder.L2Info.GenerateAccount("User2")

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	attempt := 0
	for {
		if attempt > 30 {
			Fatal(t, "2nd node didn't get tx on time")
		}
		Require(t, ctx.Err())
		countAfter, err := testClientB.ConsensusNode.TxStreamer.GetMessageCount()
		Require(t, err)
		if countAfter > countBefore {
			break
		}
		select {
		case <-time.After(time.Millisecond * 100):
		case <-ctx.Done():
		}
		attempt++
	}

	progress, err := testClientB.Client.SyncProgress(ctx)
	Require(t, err)
	if progress == nil {
		Fatal(t, "eth_syncing returned nil but shouldn't have")
	}
	for testClientB.ConsensusNode.TxStreamer.ExecuteNextMsg(ctx, testClientB.ExecNode) {
	}
	progress, err = testClientB.Client.SyncProgress(ctx)
	Require(t, err)
	if progress != nil {
		Fatal(t, "eth_syncing did not return nil but should have")
	}
}
