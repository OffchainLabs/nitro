// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithDatabase(rawdb.DBPebble)
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
	finalizedBlockNumber := uint64(10)
	finalizedBlock := builder.L2.ExecNode.Backend.BlockChain().GetBlockByNumber(finalizedBlockNumber)
	if finalizedBlock == nil {
		t.Fatalf("unable to get block by number")
	}
	builder.L2.ExecNode.Backend.BlockChain().SetFinalized(finalizedBlock.Header())
	Require(t, err)

	// Wait for freeze operation to be executed
	time.Sleep(65 * time.Second)

	ancients, err = builder.L2.ExecNode.ChainDB.Ancients()
	Require(t, err)
	// ancients must be finalizedBlock+1 since only blocks in [0, finalizedBlock] must be included in ancients.
	if ancients != finalizedBlockNumber+1 {
		t.Fatalf("Ancients should be %d, but got %d", finalizedBlockNumber+1, ancients)
	}

	ancient, err := builder.L2.ExecNode.ChainDB.Ancient(rawdb.ChainFreezerHeaderTable, 8)
	if err != nil || ancient == nil {
		t.Fatalf("Ancient should exist")
	}
	_, err = builder.L2.ExecNode.ChainDB.Ancient(rawdb.ChainFreezerHeaderTable, 15)
	if err == nil {
		t.Fatalf("Ancient should not exist")
	}
}

func checksFinalityData(
	t *testing.T,
	scenario string,
	ctx context.Context,
	testClient *TestClient,
	expectedFinalizedMsgIdx arbutil.MessageIndex,
	expectedSafeMsgIdx arbutil.MessageIndex,
) {
	expectedFinalizedBlockNumber, err := testClient.ExecNode.MessageIndexToBlockNumber(expectedFinalizedMsgIdx).Await(ctx)
	Require(t, err)
	expectedSafeBlockNumber, err := testClient.ExecNode.MessageIndexToBlockNumber(expectedSafeMsgIdx).Await(ctx)
	Require(t, err)

	finalizedBlock, err := testClient.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.FinalizedBlockNumber)
	Require(t, err)
	if finalizedBlock == nil {
		t.Fatalf("finalizedBlock should not be nil, scenario: %v", scenario)
	}
	if finalizedBlock.NumberU64() != expectedFinalizedBlockNumber {
		t.Fatalf("finalizedBlock is %d, but expected %d, scenario: %v", finalizedBlock.NumberU64(), expectedFinalizedBlockNumber, scenario)
	}
	safeBlock, err := testClient.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.SafeBlockNumber)
	Require(t, err)
	if safeBlock == nil {
		t.Fatalf("safeBlock should not be nil, scenario: %v", scenario)
	}
	if safeBlock.NumberU64() != expectedSafeBlockNumber {
		t.Fatalf("safeBlock is %d, but expected %d, scenario: %v", safeBlock.NumberU64(), expectedSafeBlockNumber, scenario)
	}
}

func TestFinalityDataWaitForBlockValidator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// The procedure that periodically pushes finality data, from consensus to execution,
	// will not be able to get finalized/safe block numbers since UseFinalityData is false.
	// Therefore, with UseFinalityData set to false, Consensus will not be able to push finalized/safe block numbers to Execution by itself.
	// In that way we can control in this test which finality data is pushed to Execution by calling SyncMonitor.SetFinalityData.
	builder.nodeConfig.ParentChainReader.UseFinalityData = false
	builder.execConfig.SyncMonitor.SafeBlockWaitForBlockValidator = true
	builder.execConfig.SyncMonitor.FinalizedBlockWaitForBlockValidator = true

	cleanup := builder.Build(t)
	defer cleanup()

	nodeConfig2ndNode := arbnode.ConfigDefaultL1NonSequencerTest()
	execConfig2ndNode := ExecConfigDefaultTest(t, env.GetTestStateScheme())
	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig2ndNode, execConfig: execConfig2ndNode})
	defer cleanup2ndNode()

	// Creates at least 20 L2 blocks
	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 20)

	safeMsgIdx := arbutil.MessageIndex(14)
	safeMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(safeMsgIdx).Await(ctx)
	Require(t, err)
	safeFinalityData := arbutil.FinalityData{
		MsgIdx:    safeMsgIdx,
		BlockHash: safeMsgResult.BlockHash,
	}

	finalizedMsgIdx := arbutil.MessageIndex(9)
	finalizedMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(finalizedMsgIdx).Await(ctx)
	Require(t, err)
	finalizedFinalityData := arbutil.FinalityData{
		MsgIdx:    finalizedMsgIdx,
		BlockHash: finalizedMsgResult.BlockHash,
	}

	validatedMsgIdx := arbutil.MessageIndex(6)
	validatedMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(validatedMsgIdx).Await(ctx)
	Require(t, err)
	validatedFinalityData := arbutil.FinalityData{
		MsgIdx:    validatedMsgIdx,
		BlockHash: validatedMsgResult.BlockHash,
	}

	err = builder.L2.ExecNode.SyncMonitor.SetFinalityData(&safeFinalityData, &finalizedFinalityData, &validatedFinalityData)
	Require(t, err)

	// wait for block validator is set to true in second node
	checksFinalityData(t, "first node", ctx, builder.L2, validatedMsgIdx, validatedMsgIdx)

	err = testClient2ndNode.ExecNode.SyncMonitor.SetFinalityData(&safeFinalityData, &finalizedFinalityData, &validatedFinalityData)
	Require(t, err)

	// wait for block validator is no set to true in second node
	checksFinalityData(t, "2nd node", ctx, testClient2ndNode, finalizedMsgIdx, safeMsgIdx)

	// if validatedFinalityData is nil, error should be returned if waitForBlockValidator is set to true
	err = builder.L2.ExecNode.SyncMonitor.SetFinalityData(&safeFinalityData, &finalizedFinalityData, nil)
	if err == nil {
		t.Fatalf("err should not be nil")
	}
	if err.Error() != "block validator not set" {
		t.Fatalf("err is not correct")
	}
}

func ensureFinalizedBlockDoesNotExist(t *testing.T, ctx context.Context, testClient *TestClient, scenario string) {
	_, err := testClient.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.FinalizedBlockNumber)
	if err == nil {
		t.Fatalf("err should not be nil, scenario: %v", scenario)
	}
	if err.Error() != "finalized block not found" {
		t.Fatalf("err is not correct, scenario: %v", scenario)
	}
}

func ensureSafeBlockDoesNotExist(t *testing.T, ctx context.Context, testClient *TestClient, scenario string) {
	_, err := testClient.ExecNode.Backend.APIBackend().BlockByNumber(ctx, rpc.SafeBlockNumber)
	if err == nil {
		t.Fatalf("err should not be nil, scenario: %v", scenario)
	}
	if err.Error() != "safe block not found" {
		t.Fatalf("err is not correct, scenario: %v", scenario)
	}
}

func TestFinalityDataPushedFromConsensusToExecution(t *testing.T) {
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

	// final and safe blocks shouldn't exist before generating blocks
	ensureFinalizedBlockDoesNotExist(t, ctx, builder.L2, "first node before generating blocks")
	ensureSafeBlockDoesNotExist(t, ctx, builder.L2, "first node before generating blocks")
	ensureFinalizedBlockDoesNotExist(t, ctx, testClient2ndNode, "2nd node before generating blocks")
	ensureSafeBlockDoesNotExist(t, ctx, testClient2ndNode, "2nd node before generating blocks")

	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 100)

	// wait for finality data to be updated in execution side
	time.Sleep(time.Second * 20)

	// finality data usage is disabled in first node, so finality data should not be set in first node
	ensureFinalizedBlockDoesNotExist(t, ctx, builder.L2, "first node after generating blocks")
	ensureSafeBlockDoesNotExist(t, ctx, builder.L2, "first node after generating blocks")

	// if nil is passed finality data should not be set
	err := builder.L2.ExecNode.SyncMonitor.SetFinalityData(nil, nil, nil)
	Require(t, err)
	ensureFinalizedBlockDoesNotExist(t, ctx, builder.L2, "first node after generating blocks and setting finality data to nil")
	ensureSafeBlockDoesNotExist(t, ctx, builder.L2, "first node after generating blocks and setting finality data to nil")

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

func TestFinalityAfterReorg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// The procedure that periodically pushes finality data, from consensus to execution,
	// will not be able to get finalized/safe block numbers since UseFinalityData is false.
	// Therefore, with UseFinalityData set to false, Consensus will not be able to push finalized/safe block numbers to Execution by itself.
	// In that way we can control in this test which finality data is pushed to Execution by calling SyncMonitor.SetFinalityData.
	builder.nodeConfig.ParentChainReader.UseFinalityData = false

	cleanup := builder.Build(t)
	defer cleanup()

	nodeConfig2ndNode := arbnode.ConfigDefaultL1NonSequencerTest()
	execConfig2ndNode := ExecConfigDefaultTest(t, env.GetTestStateScheme())
	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig2ndNode, execConfig: execConfig2ndNode})
	defer cleanup2ndNode()

	// Creates at least 20 L2 blocks
	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 20)

	safeMsgIdx := arbutil.MessageIndex(14)
	safeMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(safeMsgIdx).Await(ctx)
	Require(t, err)
	safeFinalityData := arbutil.FinalityData{
		MsgIdx:    safeMsgIdx,
		BlockHash: safeMsgResult.BlockHash,
	}

	finalizedMsgIdx := arbutil.MessageIndex(9)
	finalizedMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(finalizedMsgIdx).Await(ctx)
	Require(t, err)
	finalizedFinalityData := arbutil.FinalityData{
		MsgIdx:    finalizedMsgIdx,
		BlockHash: finalizedMsgResult.BlockHash,
	}

	err = builder.L2.ExecNode.SyncMonitor.SetFinalityData(&safeFinalityData, &finalizedFinalityData, nil)
	Require(t, err)

	checksFinalityData(t, "before reorg", ctx, builder.L2, finalizedFinalityData.MsgIdx, safeFinalityData.MsgIdx)

	reorgAt := arbutil.MessageIndex(6)
	err = builder.L2.ConsensusNode.TxStreamer.ReorgAt(reorgAt)
	Require(t, err)
	_, err = builder.L2.ExecNode.ExecEngine.HeadMessageIndexSync(t)
	Require(t, err)

	ensureFinalizedBlockDoesNotExist(t, ctx, builder.L2, "after reorg")
	ensureSafeBlockDoesNotExist(t, ctx, builder.L2, "after reorg")
}

func TestSetFinalityBlockHashMismatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// The procedure that periodically pushes finality data, from consensus to execution,
	// will not be able to get finalized/safe block numbers since UseFinalityData is false.
	// Therefore, with UseFinalityData set to false, Consensus will not be able to push finalized/safe block numbers to Execution by itself.
	// In that way we can control in this test which finality data is pushed to Execution by calling SyncMonitor.SetFinalityData.
	builder.nodeConfig.ParentChainReader.UseFinalityData = false

	cleanup := builder.Build(t)
	defer cleanup()

	nodeConfig2ndNode := arbnode.ConfigDefaultL1NonSequencerTest()
	execConfig2ndNode := ExecConfigDefaultTest(t, env.GetTestStateScheme())
	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig2ndNode, execConfig: execConfig2ndNode})
	defer cleanup2ndNode()

	// Creates at least 20 L2 blocks
	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 20)

	safeMsgIdx := arbutil.MessageIndex(14)
	safeFinalityData := arbutil.FinalityData{
		MsgIdx:    safeMsgIdx,
		BlockHash: common.Hash{},
	}

	finalizedMsgIdx := arbutil.MessageIndex(9)
	finalizedFinalityData := arbutil.FinalityData{
		MsgIdx:    finalizedMsgIdx,
		BlockHash: common.Hash{},
	}

	err := builder.L2.ExecNode.SyncMonitor.SetFinalityData(&safeFinalityData, &finalizedFinalityData, nil)
	if err == nil {
		t.Fatalf("err should not be nil")
	}
	if !strings.HasPrefix(err.Error(), "finality block hash mismatch") {
		t.Fatalf("err is not correct")
	}
}

func TestFinalityDataNodeOutOfSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// The procedure that periodically pushes finality data, from consensus to execution,
	// will not be able to get finalized/safe block numbers since UseFinalityData is false.
	// Therefore, with UseFinalityData set to false, Consensus will not be able to push finalized/safe block numbers to Execution by itself.
	// In that way we can control in this test which finality data is pushed to Execution by calling SyncMonitor.SetFinalityData.
	builder.nodeConfig.ParentChainReader.UseFinalityData = false

	cleanup := builder.Build(t)
	defer cleanup()

	nodeConfig2ndNode := arbnode.ConfigDefaultL1NonSequencerTest()
	execConfig2ndNode := builder.ExecConfigDefaultTest(t, true)
	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig2ndNode, execConfig: execConfig2ndNode})
	defer cleanup2ndNode()

	// Creates at least 20 L2 blocks
	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClient2ndNode, 20)

	safeMsgIdx := arbutil.MessageIndex(14)
	safeMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(safeMsgIdx).Await(ctx)
	Require(t, err)
	safeFinalityData := arbutil.FinalityData{
		MsgIdx:    safeMsgIdx,
		BlockHash: safeMsgResult.BlockHash,
	}

	finalizedMsgIdx := arbutil.MessageIndex(9)
	finalizedMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(finalizedMsgIdx).Await(ctx)
	Require(t, err)
	finalizedFinalityData := arbutil.FinalityData{
		MsgIdx:    finalizedMsgIdx,
		BlockHash: finalizedMsgResult.BlockHash,
	}

	err = builder.L2.ExecNode.SyncMonitor.SetFinalityData(&safeFinalityData, &finalizedFinalityData, nil)
	Require(t, err)

	checksFinalityData(t, "before out of sync", ctx, builder.L2, finalizedFinalityData.MsgIdx, safeFinalityData.MsgIdx)

	// out of sync node
	err = builder.L2.ExecNode.SyncMonitor.SetFinalityData(nil, nil, nil)
	Require(t, err)

	ensureFinalizedBlockDoesNotExist(t, ctx, builder.L2, "out of sync")
	ensureSafeBlockDoesNotExist(t, ctx, builder.L2, "out of sync")
}
