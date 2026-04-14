// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func TestMeaninglessBatchReorg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithTakeOwnership(false)
	builder.nodeConfig.BatchPoster.Enable = false
	cleanup := builder.Build(t)
	defer cleanup()

	seqInbox, err := bridgegen.NewSequencerInbox(builder.L1Info.GetAddress("SequencerInbox"), builder.L1.Client)
	Require(t, err)
	seqOpts := builder.L1Info.GetDefaultTransactOpts("Sequencer", ctx)

	tx, err := seqInbox.AddSequencerL2BatchFromOrigin8f111f3c(&seqOpts, big.NewInt(1), nil, big.NewInt(1), common.Address{}, common.Big0, common.Big0)
	Require(t, err)
	batchReceipt, err := builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	for i := 0; ; i++ {
		if i >= 500 {
			Fatal(t, "Failed to read batch from L1")
		}
		msgIdx, err := builder.L2.ExecNode.ExecEngine.HeadMessageIndex()
		Require(t, err)
		if msgIdx == 1 {
			break
		} else if msgIdx > 1 {
			Fatal(t, "More than two batches in test?")
		}
		time.Sleep(10 * time.Millisecond)
	}
	metadata, err := builder.L2.ConsensusNode.GetParentChainDataSource().GetBatchMetadata(1)
	Require(t, err)
	originalBatchBlock := batchReceipt.BlockNumber.Uint64()
	if metadata.ParentChainBlock != originalBatchBlock {
		Fatal(t, "Posted batch in block", originalBatchBlock, "but metadata says L1 block was", metadata.ParentChainBlock)
	}

	_, l2Receipt := builder.L2.TransferBalance(t, "Owner", "Owner", common.Big1, builder.L2Info)

	// Make the reorg larger to force the miner to discard transactions.
	// The miner usually collects transactions from deleted blocks and puts them in the mempool.
	// However, this code doesn't run on reorgs larger than 64 blocks for performance reasons.
	// Therefore, we make a bunch of small blocks to prevent the code from running.
	builder.L1.AdvanceBlocks(t, 70, builder.L1Info)

	compareAllMsgResultsFromConsensusAndExecution(t, ctx, builder.L2, "before reorg")

	currHead, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)
	parentBlock := builder.L1.L1Backend.BlockChain().GetBlockByNumber(batchReceipt.BlockNumber.Uint64() - 1)
	err = builder.L1.L1Backend.BlockChain().ReorgToOldBlock(parentBlock)
	Require(t, err)

	// Check that reorg was processed so that correct pending nonce is set
	for i := 0; ; i++ {
		if i >= 500 {
			Fatal(t, "Failed to see reorg in L1")
		}
		newHead, err := builder.L1.Client.BlockNumber(ctx)
		Require(t, err)
		if newHead < currHead {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	builder.L1.RecalibrateNonce(t, builder.L1Info)
	// Produce new l1Blocks so that the batch ends up in a different l1Block than before
	// #nosec G115
	builder.L1.AdvanceBlocks(t, int(currHead-parentBlock.NumberU64()+5), builder.L1Info)

	tx, err = seqInbox.AddSequencerL2BatchFromOrigin8f111f3c(&seqOpts, big.NewInt(1), nil, big.NewInt(1), common.Address{}, common.Big0, common.Big0)
	Require(t, err)
	newBatchReceipt, err := builder.L1.EnsureTxSucceeded(tx)
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
		if builder.L2.ConsensusNode.MessageExtractor != nil {
			batchCount, err := builder.L2.ConsensusNode.MessageExtractor.GetBatchCount()
			Require(t, err)
			if batchCount <= 1 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
		}
		metadata, err = builder.L2.ConsensusNode.GetParentChainDataSource().GetBatchMetadata(1)
		Require(t, err)
		if metadata.ParentChainBlock == newBatchBlock {
			break
		} else if metadata.ParentChainBlock != originalBatchBlock {
			Fatal(t, "Batch L1 block changed from", originalBatchBlock, "to", metadata.ParentChainBlock, "instead of expected", newBatchBlock)
		}
		time.Sleep(10 * time.Millisecond)
	}

	_, _, err = builder.L2.ConsensusNode.GetParentChainDataSource().GetSequencerMessageBytes(ctx, 1)
	Require(t, err)

	l2Header, err := builder.L2.Client.HeaderByNumber(ctx, l2Receipt.BlockNumber)
	Require(t, err)

	if l2Header.Hash() != l2Receipt.BlockHash {
		Fatal(t, "L2 block hash changed")
	}

	compareAllMsgResultsFromConsensusAndExecution(t, ctx, builder.L2, "after reorg")
}
