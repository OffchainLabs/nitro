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

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/testhelpers/github"
	"github.com/offchainlabs/nitro/validator/client/redis"
)

func TestFinalityData(t *testing.T) {
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

	builder.L2Info.GenerateAccount("User2")
	for i := 0; i < 30; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		_, err = WaitForTx(ctx, testClientVal.Client, tx.Hash(), time.Second*15)
		Require(t, err)
	}

	// wait for finality data to be updated in execution side
	time.Sleep(time.Second * 10)

	finalityData := builder.L2.ExecNode.SyncMonitor.GetFinalityData()
	if finalityData == nil {
		t.Fatal("Finality data is nil")
	}
	// block validator and finality data usage are disabled in first node
	expectedFinalityData := arbutil.FinalityData{
		SafeMsgCount:      0,
		FinalizedMsgCount: 0,
		BlockValidatorSet: false,
		FinalitySupported: false,
	}
	if !reflect.DeepEqual(*finalityData, expectedFinalityData) {
		t.Fatalf("Finality data is not as expected. Expected: %v, Got: %v", expectedFinalityData, *finalityData)
	}

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
}
