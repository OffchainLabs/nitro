package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbutil"
)

func TestEthSyncing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info = nil
	cleanup := builder.Build(t)
	defer cleanup()

	builder.execConfig.SyncMonitor.MsgLag = builder.nodeConfig.SyncMonitor.MsgLag
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	// stop txstreamer so it won't feed execution messages
	testClientB.ConsensusNode.TxStreamer.StopAndWait()

	countBefore, err := testClientB.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)

	builder.L2Info.GenerateAccount("User2")

	numTxs := uint64(5)
	for range numTxs {
		builder.L2.TransferBalance(t, "Owner", "User2", big.NewInt(1e12), builder.L2Info)
	}

	// Give the inbox reader of testClientB a bit of time to pick up batches from L1 and add it to the consensus db
	time.Sleep(time.Millisecond * 500)

	// Advance parent chain enough to get the batches in
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	attempt := 0
	for {
		if attempt > 30 {
			Fatal(t, "2nd node didn't get all txs on time")
		}
		Require(t, ctx.Err())
		countAfter, err := testClientB.ConsensusNode.TxStreamer.GetMessageCount()
		Require(t, err)
		if countAfter >= countBefore+arbutil.MessageIndex(numTxs) {
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
	for testClientB.ConsensusNode.TxStreamer.ExecuteNextMsg(ctx) {
	}
	for range 10 {
		progress, err = testClientB.Client.SyncProgress(ctx)
		Require(t, err)
		if progress == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if progress != nil {
		Fatal(t, "eth_syncing did not return nil but should have")
	}
}
