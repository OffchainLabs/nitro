// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func TestMeaninglessBatchReorg(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conf := arbnode.ConfigDefaultL1Test()
	conf.BatchPoster.Enable = false
	testNode := NewNodeBuilder(ctx).SetNodeConfig(conf).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNode.L1Stack)
	defer testNode.L2Node.StopAndWait()

	seqInbox, err := bridgegen.NewSequencerInbox(testNode.L1Info.GetAddress("SequencerInbox"), testNode.L1Client)
	Require(t, err)
	seqOpts := testNode.L1Info.GetDefaultTransactOpts("Sequencer", ctx)

	tx, err := seqInbox.AddSequencerL2BatchFromOrigin(&seqOpts, big.NewInt(1), nil, big.NewInt(1), common.Address{})
	Require(t, err)
	batchReceipt, err := EnsureTxSucceeded(ctx, testNode.L1Client, tx)
	Require(t, err)

	for i := 0; ; i++ {
		if i >= 500 {
			Fatal(t, "Failed to read batch from L1")
		}
		msgNum, err := testNode.L2Node.Execution.ExecEngine.HeadMessageNumber()
		Require(t, err)
		if msgNum == 1 {
			break
		} else if msgNum > 1 {
			Fatal(t, "More than two batches in test?")
		}
		time.Sleep(10 * time.Millisecond)
	}
	metadata, err := testNode.L2Node.InboxTracker.GetBatchMetadata(1)
	Require(t, err)
	originalBatchBlock := batchReceipt.BlockNumber.Uint64()
	if metadata.ParentChainBlock != originalBatchBlock {
		Fatal(t, "Posted batch in block", originalBatchBlock, "but metadata says L1 block was", metadata.ParentChainBlock)
	}

	_, l2Receipt := testNode.TransferBalanceViaL2(t, "Owner", "Owner", common.Big1)

	// Make the reorg larger to force the miner to discard transactions.
	// The miner usually collects transactions from deleted blocks and puts them in the mempool.
	// However, this code doesn't run on reorgs larger than 64 blocks for performance reasons.
	// Therefore, we make a bunch of small blocks to prevent the code from running.
	for j := uint64(0); j < 70; j++ {
		testNode.TransferBalanceViaL1(t, "Faucet", "Faucet", common.Big1)
	}

	parentBlock := testNode.L1Backend.BlockChain().GetBlockByNumber(batchReceipt.BlockNumber.Uint64() - 1)
	err = testNode.L1Backend.BlockChain().ReorgToOldBlock(parentBlock)
	Require(t, err)

	// Produce a new l1Block so that the batch ends up in a different l1Block than before
	testNode.TransferBalanceViaL1(t, "User", "User", common.Big1)

	tx, err = seqInbox.AddSequencerL2BatchFromOrigin(&seqOpts, big.NewInt(1), nil, big.NewInt(1), common.Address{})
	Require(t, err)
	newBatchReceipt, err := EnsureTxSucceeded(ctx, testNode.L1Client, tx)
	Require(t, err)

	newBatchBlock := newBatchReceipt.BlockNumber.Uint64()
	if newBatchBlock == originalBatchBlock {
		Fatal(t, "Attempted to change L1 block number in batch reorg, but it ended up in the same block", newBatchBlock)
	} else {
		t.Log("Batch successfully moved in reorg from L1 block", originalBatchBlock, "to L1 block", newBatchBlock)
	}

	for i := 0; ; i++ {
		if i >= 500 {
			Fatal(t, "Failed to read batch reorg from L1")
		}
		metadata, err = testNode.L2Node.InboxTracker.GetBatchMetadata(1)
		Require(t, err)
		if metadata.ParentChainBlock == newBatchBlock {
			break
		} else if metadata.ParentChainBlock != originalBatchBlock {
			Fatal(t, "Batch L1 block changed from", originalBatchBlock, "to", metadata.ParentChainBlock, "instead of expected", metadata.ParentChainBlock)
		}
		time.Sleep(10 * time.Millisecond)
	}

	_, err = testNode.L2Node.InboxReader.GetSequencerMessageBytes(ctx, 1)
	Require(t, err)

	l2Header, err := testNode.L2Client.HeaderByNumber(ctx, l2Receipt.BlockNumber)
	Require(t, err)

	if l2Header.Hash() != l2Receipt.BlockHash {
		Fatal(t, "L2 block hash changed")
	}
}
