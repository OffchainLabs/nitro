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
	l2Info, arbNode, l2Client, l1Info, l1Backend, l1Client, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, conf, nil, nil, nil)
	defer requireClose(t, l1stack)
	defer arbNode.StopAndWait()

	seqInbox, err := bridgegen.NewSequencerInbox(l1Info.GetAddress("SequencerInbox"), l1Client)
	Require(t, err)
	seqOpts := l1Info.GetDefaultTransactOpts("Sequencer", ctx)

	tx, err := seqInbox.AddSequencerL2BatchFromOrigin(&seqOpts, big.NewInt(1), nil, big.NewInt(1), common.Address{})
	Require(t, err)
	batchReceipt, err := EnsureTxSucceeded(ctx, l1Client, tx)
	Require(t, err)

	execNode := getExecNode(t, arbNode)
	for i := 0; ; i++ {
		if i >= 500 {
			Fatal(t, "Failed to read batch from L1")
		}
		msgNum, err := execNode.ExecEngine.HeadMessageNumber()
		Require(t, err)
		if msgNum == 1 {
			break
		} else if msgNum > 1 {
			Fatal(t, "More than two batches in test?")
		}
		time.Sleep(10 * time.Millisecond)
	}
	metadata, err := arbNode.InboxTracker.GetBatchMetadata(1)
	Require(t, err)
	originalBatchBlock := batchReceipt.BlockNumber.Uint64()
	if metadata.ParentChainBlock != originalBatchBlock {
		Fatal(t, "Posted batch in block", originalBatchBlock, "but metadata says L1 block was", metadata.ParentChainBlock)
	}

	_, l2Receipt := TransferBalance(t, "Owner", "Owner", common.Big1, l2Info, l2Client, ctx)

	// Make the reorg larger to force the miner to discard transactions.
	// The miner usually collects transactions from deleted blocks and puts them in the mempool.
	// However, this code doesn't run on reorgs larger than 64 blocks for performance reasons.
	// Therefore, we make a bunch of small blocks to prevent the code from running.
	for j := uint64(0); j < 70; j++ {
		TransferBalance(t, "Faucet", "Faucet", common.Big1, l1Info, l1Client, ctx)
	}

	parentBlock := l1Backend.BlockChain().GetBlockByNumber(batchReceipt.BlockNumber.Uint64() - 1)
	err = l1Backend.BlockChain().ReorgToOldBlock(parentBlock)
	Require(t, err)

	// Produce a new l1Block so that the batch ends up in a different l1Block than before
	TransferBalance(t, "User", "User", common.Big1, l1Info, l1Client, ctx)

	tx, err = seqInbox.AddSequencerL2BatchFromOrigin(&seqOpts, big.NewInt(1), nil, big.NewInt(1), common.Address{})
	Require(t, err)
	newBatchReceipt, err := EnsureTxSucceeded(ctx, l1Client, tx)
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
		metadata, err = arbNode.InboxTracker.GetBatchMetadata(1)
		Require(t, err)
		if metadata.ParentChainBlock == newBatchBlock {
			break
		} else if metadata.ParentChainBlock != originalBatchBlock {
			Fatal(t, "Batch L1 block changed from", originalBatchBlock, "to", metadata.ParentChainBlock, "instead of expected", metadata.ParentChainBlock)
		}
		time.Sleep(10 * time.Millisecond)
	}

	_, err = arbNode.InboxReader.GetSequencerMessageBytes(ctx, 1)
	Require(t, err)

	l2Header, err := l2Client.HeaderByNumber(ctx, l2Receipt.BlockNumber)
	Require(t, err)

	if l2Header.Hash() != l2Receipt.BlockHash {
		Fatal(t, "L2 block hash changed")
	}
}
