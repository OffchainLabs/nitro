package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestValidationPostMEL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.Post4844Blobs = true
	builder.nodeConfig.BatchPoster.IgnoreBlobPrice = true
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	builder.nodeConfig.MELValidator.Enable = true
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.EnableMEL = true
	builder.nodeConfig.BlockValidator.ForwardBlocks = 0
	cleanup := builder.Build(t)
	defer cleanup()

	// Post a blob batch with a bunch of txs
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for range 20 {
		tx, _ := builder.L2.TransferBalance(t, "Faucet", "User2", big.NewInt(1e12), builder.L2Info)
		txs = append(txs, tx)
	}
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	_, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}
	CheckBatchCount(t, builder, initialBatchCount+1)

	// Post delayed messages
	forceDelayedBatchPosting(t, ctx, builder, testClientB, 10, 0)

	extractedMsgCountToValidate, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 40)

	timeout := getDeadlineTimeout(t, time.Minute*10)
	if !builder.L2.ConsensusNode.BlockValidator.WaitForPos(t, ctx, extractedMsgCountToValidate-1, timeout) {
		Fatal(t, "did not validate all blocks")
	}
}
