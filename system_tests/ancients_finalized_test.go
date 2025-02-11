// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/testhelpers/github"
	"github.com/offchainlabs/nitro/validator/client/redis"
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
	// will not be able to get the finalized block number since UseFinalityData is false.
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

func TestFinalityDataPushedFromConsensusToExecution(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.ParentChainReader.UseFinalityData = false
	builder.nodeConfig.BlockValidator.Enable = false
	// For now PathDB is not supported when using block validation
	builder.execConfig.Caching.StateScheme = rawdb.HashScheme
	cleanup := builder.Build(t)
	defer cleanup()

	validatorConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	validatorConfig.ParentChainReader.UseFinalityData = true
	validatorConfig.BlockValidator.Enable = true
	validatorConfig.BlockValidator.RedisValidationClientConfig = redis.ValidationClientConfig{}

	cr, err := github.LatestConsensusRelease(context.Background())
	Require(t, err)
	machPath := populateMachineDir(t, cr)
	AddValNode(t, ctx, validatorConfig, true, "", machPath)

	testClientVal, cleanupVal := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: validatorConfig})
	defer cleanupVal()

	// final block should be nil in before generating blocks
	finalBlock := builder.L2.ExecNode.Backend.BlockChain().CurrentFinalBlock()
	if finalBlock != nil {
		t.Fatalf("finalBlock should be nil, but got %v", finalBlock)
	}
	finalBlock = testClientVal.ExecNode.Backend.BlockChain().CurrentFinalBlock()
	if finalBlock != nil {
		t.Fatalf("finalBlock should be nil, but got %v", finalBlock)
	}

	builder.L2Info.GenerateAccount("User2")
	generateBlocks(t, ctx, builder, testClientVal, 100)

	// wait for finality data to be updated in execution side
	time.Sleep(time.Second * 20)

	// block validator and finality data usage are disabled in first node,
	// so finality data should not be set in first node
	finalityData := builder.L2.ExecNode.SyncMonitor.GetFinalityData()
	if finalityData == nil {
		t.Fatal("Finality data is nil")
	}
	expectedFinalityData := arbutil.FinalityData{
		SafeMsgCount:      0,
		FinalizedMsgCount: 0,
		BlockValidatorSet: false,
		FinalitySupported: false,
	}
	if !reflect.DeepEqual(*finalityData, expectedFinalityData) {
		t.Fatalf("Finality data is not as expected. Expected: %v, Got: %v", expectedFinalityData, *finalityData)
	}
	finalBlock = builder.L2.ExecNode.Backend.BlockChain().CurrentFinalBlock()
	if finalBlock != nil {
		t.Fatalf("finalBlock should be nil, but got %v", finalBlock)
	}

	// block validator and finality data usage are enabled in second node,
	// so finality data should be set in second node
	finalityDataVal := testClientVal.ExecNode.SyncMonitor.GetFinalityData()
	if finalityDataVal == nil {
		t.Fatal("Finality data is nil")
	}
	if finalityDataVal.SafeMsgCount == 0 {
		t.Fatal("SafeMsgCount is 0")
	}
	if finalityDataVal.FinalizedMsgCount == 0 {
		t.Fatal("FinalizedMsgCount is 0")
	}
	if !finalityDataVal.BlockValidatorSet {
		t.Fatal("BlockValidatorSet is false")
	}
	if !finalityDataVal.FinalitySupported {
		t.Fatal("FinalitySupported is false")
	}
	finalBlock = testClientVal.ExecNode.Backend.BlockChain().CurrentFinalBlock()
	if finalBlock == nil {
		t.Fatalf("finalBlock should not be nil")
	}
}
