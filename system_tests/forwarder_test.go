// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"path/filepath"
	"testing"

	// "github.com/alicebob/miniredis"
	// "github.com/go-redis/redis/v8"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// func initRedisForTest(t *testing.T, ctx context.Context, nodeNames []string) *Miniredis {
//	var priorities string
//	redisServer, err := miniredis.Run()
//	testhelpers.RequireImpl(t, err)
//
//	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
//	testhelpers.RequireImpl(t, err)
//	defer redisClient.Close()
//
//	for _, name := range nodeNames {
//		priorities = priorities + name + ","
//		redisClient.Del(ctx, redisutil.LIVELINESS_KEY_PREFIX+name)
//	}
//	priorities = priorities[:len(priorities)-1] // remove last ","
//	testhelpers.RequireImpl(t, redisClient.Set(ctx, redisutil.PRIORITIES_KEY, priorities, time.Duration(0)).Err())
//	for msg := 0; msg < 1000; msg++ {
//		redisClient.Del(ctx, fmt.Sprintf("%s%d", redisutil.MESSAGE_KEY_PREFIX, msg))
//	}
//	redisClient.Del(ctx, redisutil.CHOSENSEQ_KEY, redisutil.MSG_COUNT_KEY)
//
//	return redisServer
//}

func TestStaticForwarder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipcPath := filepath.Join(t.TempDir(), "test.ipc")
	ipcConfig := genericconf.IPCConfigDefault
	ipcConfig.Path = ipcPath
	stackConf := getTestStackConfig(t)
	ipcConfig.Apply(stackConf)
	nodeConfigA := arbnode.ConfigDefaultL1Test()
	nodeConfigA.BatchPoster.Enable = false

	l2info, nodeA, clientA, l1info, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nodeConfigA, nil, stackConf)
	defer requireClose(t, l1stack)
	defer nodeA.StopAndWait()

	nodeConfigB := arbnode.ConfigDefaultL1Test()
	nodeConfigB.Sequencer.Enable = false
	nodeConfigB.DelayedSequencer.Enable = false
	nodeConfigB.Forwarder.RedisUrl = ""
	nodeConfigB.ForwardingTargetImpl = ipcPath
	nodeConfigB.BatchPoster.Enable = false

	clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, l1info, &l2info.ArbInitData, nodeConfigB, nil)
	defer nodeB.StopAndWait()

	l2info.GenerateAccount("User2")
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := clientB.SendTransaction(ctx, tx)
	testhelpers.RequireImpl(t, err)

	_, err = EnsureTxSucceeded(ctx, clientA, tx)
	testhelpers.RequireImpl(t, err)
	l2balance, err := clientA.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	testhelpers.RequireImpl(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}
