// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/redisutil"
)

// TestSequencerUsesCurrentBaseFeeAfterPromotion guards that a promoted standby gates
// transactions against the current head base fee, not one captured at createBlock start.
func TestSequencerUsesCurrentBaseFeeAfterPromotion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const loweredBaseFee int64 = l2pricing.InitialBaseFeeWei / 5 // 0.02 gwei
	const txFeeCap int64 = l2pricing.InitialBaseFeeWei / 2       // 0.05 gwei: above lowered, below genesis

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).DontParalellise()
	builder.nodeConfig.SeqCoordinator.Enable = true
	builder.nodeConfig.SeqCoordinator.RedisUrl = redisutil.CreateTestRedis(ctx, t)
	builder.nodeConfig.BatchPoster.Enable = false

	nodeNames := []string{"stdio://A", "stdio://B"}
	initRedisForTest(t, ctx, builder.nodeConfig.SeqCoordinator.RedisUrl, nodeNames)
	builder.nodeConfig.SeqCoordinator.MyUrl = nodeNames[0]

	cleanupA := builder.Build(t)
	defer cleanupA()

	redisClient, err := redisutil.RedisClientFromURL(builder.nodeConfig.SeqCoordinator.RedisUrl)
	Require(t, err)
	defer redisClient.Close()
	waitForChosenSequencer(t, ctx, redisClient, builder.nodeConfig.SeqCoordinator.UpdateInterval)

	builder.L2Info.GenerateAccount("User2")

	nodeConfigB := *builder.nodeConfig
	nodeConfigB.Feed.Output = *newBroadcasterConfigTest()
	nodeConfigB.SeqCoordinator.MyUrl = nodeNames[1]
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: &nodeConfigB})
	defer cleanupB()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	setTx, err := arbOwner.SetMinimumL2BaseFee(&auth, big.NewInt(loweredBaseFee))
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(setTx)
	Require(t, err)

	// Drive blocks on A until B has digested the lowered base fee.
	dropped := false
	for i := 0; i < 20 && !dropped; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e15), nil)
		Require(t, builder.L2.Client.SendTransaction(ctx, tx))
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		if GetBaseFee(t, testClientB.Client, ctx).Cmp(big.NewInt(loweredBaseFee*2)) <= 0 {
			dropped = true
		}
	}
	if !dropped {
		t.Fatalf("node B base fee did not drop below %d", loweredBaseFee*2)
	}

	// Stopping A promotes B, whose createBlock has been parked since it was a standby.
	builder.L2.ConsensusNode.StopAndWait()
	pollUntil(t, ctx, 60*time.Second, 100*time.Millisecond, "node B to become chosen sequencer", func() bool {
		return testClientB.ConsensusNode.SeqCoordinator.CurrentlyChosen()
	})

	info := builder.L2Info.GetInfoWithPrivKey("User2")
	recipient := builder.L2Info.GetAddress("Owner")
	lowFeeTx := builder.L2Info.SignTxAs("User2", &types.DynamicFeeTx{
		ChainID:   builder.L2Info.Signer.ChainID(),
		Nonce:     info.Nonce.Add(1) - 1,
		GasTipCap: big.NewInt(loweredBaseFee),
		GasFeeCap: big.NewInt(txFeeCap),
		Gas:       builder.L2Info.TransferGas,
		To:        &recipient,
		Value:     big.NewInt(1e9),
	})
	Require(t, testClientB.Client.SendTransaction(ctx, lowFeeTx))
	_, err = testClientB.EnsureTxSucceeded(lowFeeTx)
	Require(t, err)
}
