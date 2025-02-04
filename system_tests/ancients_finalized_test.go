// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode"
)

func generateBlocks(t *testing.T, ctx context.Context, builder *NodeBuilder, testClient2ndNode *TestClient, n int) {
	for i := 0; i < n; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		_, err = WaitForTx(ctx, testClient2ndNode.Client, tx.Hash(), time.Second*15)
		Require(t, err)
	}
}

func TestSetFinalized(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.ParentChainReader.UseFinalityData = true

	cleanup := builder.Build(t)
	defer cleanup()

	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: arbnode.ConfigDefaultL1NonSequencerTest()})
	defer cleanup2ndNode()

	bc := builder.L2.ExecNode.Backend.BlockChain()
	finalBlock := bc.CurrentFinalBlock()
	if finalBlock != nil {
		t.Fatalf("finalBlock should be nil, but got %v", finalBlock)
	}

	// Creates at least 100 L2 blocks
	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 100)

	// Waits for the procedure that periodically sets the finalized block in ExecutionNode
	time.Sleep(65 * time.Second)

	// Final block should have been set
	finalBlock = bc.CurrentFinalBlock()
	if finalBlock == nil {
		t.Fatalf("finalBlock should not be nil")
	}
}

func TestAncientsFinalized(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// The procedure that periodically sets the finalized block in ExecutionNode
	// will not be able to get the finalized block number from Consensus since UseFinalityData is false.
	// With UseFinalityData set to false, ExecutionEngine will not be able to move data to ancients,
	// at least for blocks with numbers smaller than fullImmutabilityThreshold set in ExecutionEngine.
	builder.nodeConfig.ParentChainReader.UseFinalityData = false

	cleanup := builder.Build(t)
	defer cleanup()

	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: arbnode.ConfigDefaultL1NonSequencerTest()})
	defer cleanup2ndNode()

	// Creates at least 20 L2 blocks
	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 20)

	ancients, err := builder.L2.ExecNode.ChainDB.Ancients()
	Require(t, err)
	if ancients != 0 {
		t.Fatalf("Ancients should be 0, but got %d", ancients)
	}

	// manually set finalized block
	finalizedBlock := uint64(10)
	err = builder.L2.ExecNode.ExecEngine.SetFinalized(finalizedBlock)
	Require(t, err)

	// Wait for freeze operation to be executed
	time.Sleep(65 * time.Second)

	ancients, err = builder.L2.ExecNode.ChainDB.Ancients()
	Require(t, err)
	// ancients must be finalizedBlock+1 since only blocks in [0, finalizedBlock] must be included in ancients.
	if ancients != finalizedBlock+1 {
		t.Fatalf("Ancients should be %d, but got %d", finalizedBlock+1, ancients)
	}

	hasAncient, err := builder.L2.ExecNode.ChainDB.HasAncient(rawdb.ChainFreezerHeaderTable, 8)
	Require(t, err)
	if !hasAncient {
		t.Fatalf("Ancient should exist")
	}
	hasAncient, err = builder.L2.ExecNode.ChainDB.HasAncient(rawdb.ChainFreezerHeaderTable, 15)
	Require(t, err)
	if hasAncient {
		t.Fatalf("Ancient should not exist")
	}

	// set finalized block to head of the chain
	headOfTheChain := builder.L2.ExecNode.Backend.BlockChain().CurrentBlock().Number.Uint64()
	err = builder.L2.ExecNode.ExecEngine.SetFinalized(headOfTheChain)
	Require(t, err)

	// Wait for freeze operation to be executed
	time.Sleep(65 * time.Second)

	// checks that head of the chain is not included in ancients
	ancients, err = builder.L2.ExecNode.ChainDB.Ancients()
	Require(t, err)
	// ancients must be headOfTheChain since only blocks in [0, headOfTheChain) must be included in ancients.
	if ancients != headOfTheChain {
		t.Fatalf("Ancients should be %d, but got %d", headOfTheChain, ancients)
	}
	hasAncient, err = builder.L2.ExecNode.ChainDB.HasAncient(rawdb.ChainFreezerHeaderTable, headOfTheChain)
	Require(t, err)
	if hasAncient {
		t.Fatalf("Ancient should not exist")
	}
}

func TestAncientsWithoutFinalizedSupportInParentChain(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// The procedure that periodically sets the finalized block in ExecutionNode
	// will not be able to get the finalized block number from Consensus since UseFinalityData is false.
	// With UseFinalityData set to false, ExecutionEngine will not be able to move data to ancients,
	// at least for blocks with numbers smaller than fullImmutabilityThreshold set in ExecutionEngine.
	builder.nodeConfig.ParentChainReader.UseFinalityData = false
	// Avoids creating more transactions than necessary
	builder.nodeConfig.DelayedSequencer.Enable = false

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.ExecNode.ExecEngine.SetFullImmutabilityThreshold(30)

	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: arbnode.ConfigDefaultL1NonSequencerTest()})
	defer cleanup2ndNode()

	// Creates 40 L2 blocks
	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 40)

	// Waits for the procedure that periodically sets the finalized block in ExecutionNode
	time.Sleep(65 * time.Second)

	// Waits for the procedure that moves data to ancients
	time.Sleep(65 * time.Second)

	headOfTheChain := builder.L2.ExecNode.Backend.BlockChain().CurrentBlock().Number.Uint64()
	log.Error("Head of the chain", "headOfTheChain", headOfTheChain)

	// Only blocks in [0, (head of the chain) - fullImmutabilityThreshold] == [0, 40 - 30] must be included in ancients.
	ancients, err := builder.L2.ExecNode.ChainDB.Ancients()
	Require(t, err)
	if ancients != 11 {
		t.Fatalf("Ancients should be 11, but got %d", ancients)
	}

	hasAncient, err := builder.L2.ExecNode.ChainDB.HasAncient(rawdb.ChainFreezerHeaderTable, 8)
	Require(t, err)
	if !hasAncient {
		t.Fatalf("Ancient should exist")
	}
	hasAncient, err = builder.L2.ExecNode.ChainDB.HasAncient(rawdb.ChainFreezerHeaderTable, 15)
	Require(t, err)
	if hasAncient {
		t.Fatalf("Ancient should not exist")
	}
}
