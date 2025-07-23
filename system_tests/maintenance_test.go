// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func checkMaintenanceRun(t *testing.T, builder *NodeBuilder, ctx context.Context, logHandler *testhelpers.LogHandler) {
	numberOfTransfers := 10
	for i := 2; i < 3+numberOfTransfers; i++ {
		account := fmt.Sprintf("User%d", i)
		builder.L2Info.GenerateAccount(account)

		tx := builder.L2Info.PrepareTx("Owner", account, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	maybeRunMaintenanceDone := make(chan struct{})
	go func() {
		builder.L2.ConsensusNode.MaintenanceRunner.MaybeRunMaintenance(ctx)
		close(maybeRunMaintenanceDone)
	}()
	select {
	case <-maybeRunMaintenanceDone:
	case <-time.After(10 * time.Second):
		t.Fatal("Maintenance did not complete in time")
	case <-ctx.Done():
		t.Fatal("Context cancelled before maintenance completed")
	}

	if !logHandler.WasLogged("Execution is not running maintenance anymore, maintenance completed successfully") {
		t.Fatal("Maintenance did not complete successfully from Consensus perspective")
	}
	if !logHandler.WasLogged("Flushed trie db through maintenance completed successfully") {
		t.Fatal("Expected log message not found")
	}

	// checks that balances are correct after maintenance
	for i := 2; i < 3+numberOfTransfers; i++ {
		account := fmt.Sprintf("User%d", i)
		balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress(account), nil)
		Require(t, err)
		if balance.Cmp(big.NewInt(int64(1e12))) != 0 {
			t.Fatal("Unexpected balance:", balance, "for account:", account)
		}
	}
}

func TestMaintenanceWithoutSeqCoordinator(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).DontParalellise()
	builder.nodeConfig.Maintenance.Enable = true
	builder.execConfig.Caching.TrieTimeLimitBeforeFlushMaintenance = time.Duration(math.MaxInt64) // effectively execution will always suggest to run maintenance
	cleanup := builder.Build(t)
	defer cleanup()

	checkMaintenanceRun(t, builder, ctx, logHandler)
}

func TestMaintenanceWithSeqCoordinator(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).DontParalellise()
	builder.nodeConfig.Maintenance.Enable = true
	builder.execConfig.Caching.TrieTimeLimitBeforeFlushMaintenance = time.Duration(math.MaxInt64) // effectively execution will always suggest to run maintenance
	builder.nodeConfig.BatchPoster.Enable = false
	builder.nodeConfig.SeqCoordinator.Enable = true
	builder.nodeConfig.SeqCoordinator.RedisUrl = redisutil.CreateTestRedis(ctx, t)

	nodeNames := []string{"stdio://A", "stdio://B"}
	initRedisForTest(t, ctx, builder.nodeConfig.SeqCoordinator.RedisUrl, nodeNames)
	builder.nodeConfig.SeqCoordinator.MyUrl = nodeNames[0]

	cleanup := builder.Build(t)
	defer cleanup()

	redisClient, err := redisutil.RedisClientFromURL(builder.nodeConfig.SeqCoordinator.RedisUrl)
	Require(t, err)
	defer redisClient.Close()

	// wait for sequencerA to become master
	for {
		err := redisClient.Get(ctx, redisutil.CHOSENSEQ_KEY).Err()
		if errors.Is(err, redis.Nil) {
			time.Sleep(builder.nodeConfig.SeqCoordinator.UpdateInterval)
			continue
		}
		Require(t, err)
		break
	}

	nodeConfigDup := *builder.nodeConfig
	nodeConfigDup.SeqCoordinator.MyUrl = nodeNames[1]
	_, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: &nodeConfigDup})
	defer cleanupB()

	checkMaintenanceRun(t, builder, ctx, logHandler)

	if !logHandler.WasLogged("Avoided lockout and handed off chosen one") {
		t.Fatal("Expected log message not found")
	}
}
