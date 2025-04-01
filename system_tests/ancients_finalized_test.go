// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
)

func generateBlocks(t *testing.T, ctx context.Context, builder *NodeBuilder, testClient2ndNode *TestClient, transactions int) {
	for i := 0; i < transactions; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		_, err = WaitForTx(ctx, testClient2ndNode.Client, tx.Hash(), time.Second*15)
		Require(t, err)
	}
}

func TestFinalizedBlocksMovedToAncients(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// The procedure that periodically pushes finality data, from consensus to execution,
	// will not be able to get finalized/safe block numbers since UseFinalityData is false.
	// Therefore, with UseFinalityData set to false, ExecutionEngine will not be able to move data to ancients by itself,
	// at least while HEAD is smaller than the params.FullImmutabilityThreshold const defined in go-ethereum.
	// In that way we can control in this test which blocks are moved to ancients by calling ExecEngine.SetFinalized.
	builder.nodeConfig.ParentChainReader.UseFinalityData = false

	cleanup := builder.Build(t)
	defer cleanup()

	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: arbnode.ConfigDefaultL1NonSequencerTest()})
	defer cleanup2ndNode()

	// Creates at least 20 L2 blocks
	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 20)

	headOfTheChain := builder.L2.ExecNode.Backend.BlockChain().CurrentBlock().Number.Uint64()
	if headOfTheChain >= params.FullImmutabilityThreshold {
		t.Fatalf("Test should be adjusted to generate less blocks. Current head: %d", headOfTheChain)
	}

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
}

func TestFinalityDataWaitForBlockValidator(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// The procedure that periodically pushes finality data, from consensus to execution,
	// will not be able to get finalized/safe block numbers since UseFinalityData is false.
	// Therefore, with UseFinalityData set to false, Consensus will not be able to push finalized/safe block numbers to Execution by itself.
	// In that way we can control in this test which finality data is pushed to from Consentus to Execution by calling SyncMonitor.SetFinalityData.
	builder.nodeConfig.ParentChainReader.UseFinalityData = false
	builder.execConfig.SyncMonitor.SafeBlockWaitForBlockValidator = true
	builder.execConfig.SyncMonitor.FinalizedBlockWaitForBlockValidator = true

	cleanup := builder.Build(t)
	defer cleanup()

	nodeConfig2ndNode := arbnode.ConfigDefaultL1NonSequencerTest()
	execConfig2ndNode := ExecConfigDefaultTest(t)
	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig2ndNode, execConfig: execConfig2ndNode})
	defer cleanup2ndNode()

	// Creates at least 20 L2 blocks
	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 20)

	validatedMsgCount := arbutil.MessageIndex(7)
	finalityData := arbutil.FinalityData{
		FinalizedMsgCount: 10,
		SafeMsgCount:      15,
		ValidatedMsgCount: &validatedMsgCount,
	}

	// wait for block validator is set to true in first node
	err := builder.L2.ExecNode.SyncMonitor.SetFinalityData(ctx, &finalityData)
	Require(t, err)
	finalBlock, err := builder.L2.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.FinalizedBlockNumber)
	Require(t, err)
	if finalBlock == nil {
		t.Fatalf("finalBlock should not be nil")
	}
	validatedBlockNumber, err := builder.L2.ExecNode.MessageIndexToBlockNumber(validatedMsgCount - 1).Await(ctx)
	Require(t, err)
	if finalBlock.NumberU64() != validatedBlockNumber {
		t.Fatalf("finalBlock is not correct")
	}
	safeBlock, err := builder.L2.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.SafeBlockNumber)
	Require(t, err)
	if safeBlock == nil {
		t.Fatalf("safeBlock should not be nil")
	}
	if safeBlock.NumberU64() != validatedBlockNumber {
		t.Fatalf("safeBlock is not correct")
	}

	// wait for block validator is no set to true in second node
	err = testClient2ndNode.ExecNode.SyncMonitor.SetFinalityData(ctx, &finalityData)
	Require(t, err)
	finalBlock, err = testClient2ndNode.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.FinalizedBlockNumber)
	Require(t, err)
	if finalBlock == nil {
		t.Fatalf("finalBlock should not be nil")
	}
	if finalBlock.NumberU64() != uint64(finalityData.FinalizedMsgCount-1) {
		t.Fatalf("finalBlock is not correct")
	}
	safeBlock, err = testClient2ndNode.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.SafeBlockNumber)
	Require(t, err)
	if safeBlock == nil {
		t.Fatalf("safeBlock should not be nil")
	}
	if safeBlock.NumberU64() != uint64(finalityData.SafeMsgCount-1) {
		t.Fatalf("safeBlock is not correct")
	}

	// if validatedMsgCount is nil, error should be returned if waitForBlockValidator is set to true
	finalityData.ValidatedMsgCount = nil
	err = builder.L2.ExecNode.SyncMonitor.SetFinalityData(ctx, &finalityData)
	if err == nil {
		t.Fatalf("err should not be nil")
	}
	if err.Error() != "block validator not set" {
		t.Fatalf("err is not correct")
	}
}

func TestFinalityDataPushedFromConsensusToExecution(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.ParentChainReader.UseFinalityData = false
	cleanup := builder.Build(t)
	defer cleanup()

	nodeConfig2ndNode := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfig2ndNode.ParentChainReader.UseFinalityData = true
	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig2ndNode})
	defer cleanup2ndNode()

	ensureFinalizedBlockDoesNotExist := func(testClient *TestClient, scenario string) {
		_, err := testClient.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.FinalizedBlockNumber)
		if err == nil {
			t.Fatalf("err should not be nil, scenario: %v", scenario)
		}
		if err.Error() != "finalized block not found" {
			t.Fatalf("err is not correct, scenario: %v", scenario)
		}
	}
	ensureSafeBlockDoesNotExist := func(testClient *TestClient, scenario string) {
		_, err := testClient.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.SafeBlockNumber)
		if err == nil {
			t.Fatalf("err should not be nil, scenario: %v", scenario)
		}
		if err.Error() != "safe block not found" {
			t.Fatalf("err is not correct, scenario: %v", scenario)
		}
	}

	// final and safe blocks shouldn't exist before generating blocks
	ensureFinalizedBlockDoesNotExist(builder.L2, "first node before generating blocks")
	ensureSafeBlockDoesNotExist(builder.L2, "first node before generating blocks")
	ensureFinalizedBlockDoesNotExist(testClient2ndNode, "2nd node before generating blocks")
	ensureSafeBlockDoesNotExist(testClient2ndNode, "2nd node before generating blocks")

	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 100)

	// wait for finality data to be updated in execution side
	time.Sleep(time.Second * 20)

	// finality data usage is disabled in first node, so finality data should not be set in first node
	ensureFinalizedBlockDoesNotExist(builder.L2, "first node after generating blocks")
	ensureSafeBlockDoesNotExist(builder.L2, "first node after generating blocks")

	// finality data usage is enabled in second node, so finality data should be set in second node
	finalBlock, err := testClient2ndNode.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.FinalizedBlockNumber)
	Require(t, err)
	if finalBlock == nil {
		t.Fatalf("finalBlock should not be nil")
	}
	if finalBlock.NumberU64() == 0 {
		t.Fatalf("finalBlock is not correct")
	}
	safeBlock, err := testClient2ndNode.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.SafeBlockNumber)
	Require(t, err)
	if safeBlock == nil {
		t.Fatalf("safeBlock should not be nil")
	}
	if safeBlock.NumberU64() == 0 {
		t.Fatalf("safeBlock is not correct")
	}
}
