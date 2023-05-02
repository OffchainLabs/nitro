// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/execution"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestStaticForwarder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipcPath := filepath.Join(t.TempDir(), "test.ipc")
	ipcConfig := genericconf.IPCConfigDefault
	ipcConfig.Path = ipcPath
	stackConfig := getTestStackConfig(t)
	ipcConfig.Apply(stackConfig)
	nodeConfigA := arbnode.ConfigDefaultL1Test()
	nodeConfigA.BatchPoster.Enable = false

	l2info, nodeA, clientA, l1info, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nodeConfigA, nil, stackConfig)
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
		testhelpers.FailImpl(t, "Unexpected balance:", l2balance)
	}
}

func initMiniRedisForTest(t *testing.T, ctx context.Context, nodeNames []string) (*miniredis.Miniredis, string) {
	var priorities string
	redisServer, err := miniredis.Run()
	testhelpers.RequireImpl(t, err)

	redisUrl := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	testhelpers.RequireImpl(t, err)
	defer redisClient.Close()

	for _, name := range nodeNames {
		priorities = priorities + name + ","
	}
	priorities = priorities[:len(priorities)-1] // remove last ","
	testhelpers.RequireImpl(t, redisClient.Set(ctx, redisutil.PRIORITIES_KEY, priorities, time.Duration(0)).Err())
	return redisServer, redisUrl
}

func createFallbackSequencer(
	t *testing.T, ctx context.Context, ipcPath string, redisUrl string,
) (l2info info, currentNode *arbnode.Node, l2client *ethclient.Client,
	l1info info, l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node) {
	stackConfig := getTestStackConfig(t)
	ipcConfig := genericconf.IPCConfigDefault
	ipcConfig.Path = ipcPath
	ipcConfig.Apply(stackConfig)
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.SeqCoordinator.Enable = false
	nodeConfig.SeqCoordinator.RedisUrl = redisUrl
	nodeConfig.SeqCoordinator.MyUrlImpl = ipcPath
	return createTestNodeOnL1WithConfig(t, ctx, true, nodeConfig, nil, stackConfig)
}

func createForwardingNode(
	t *testing.T, ctx context.Context,
	first *arbnode.Node,
	l1stack *node.Node,
	l1info *BlockchainTestInfo,
	l2InitData *statetransfer.ArbosInitializationInfo,
	ipcPath string,
	redisUrl string,
	fallbackPath string,
) (*ethclient.Client, *arbnode.Node) {
	stackConfig := getTestStackConfig(t)
	if ipcPath != "" {
		ipcConfig := genericconf.IPCConfigDefault
		ipcConfig.Path = ipcPath
		ipcConfig.Apply(stackConfig)
	}
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.Sequencer.Enable = false
	nodeConfig.DelayedSequencer.Enable = false
	nodeConfig.Forwarder.RedisUrl = redisUrl
	nodeConfig.ForwardingTargetImpl = fallbackPath
	//	nodeConfig.Feed.Output.Enable = false

	return Create2ndNodeWithConfig(t, ctx, first, l1stack, l1info, l2InitData, nodeConfig, stackConfig)
}

func createSequencer(
	t *testing.T, ctx context.Context,
	first *arbnode.Node,
	l1stack *node.Node,
	l1info *BlockchainTestInfo,
	l2InitData *statetransfer.ArbosInitializationInfo,
	ipcPath string,
	redisUrl string,
) (*ethclient.Client, *arbnode.Node) {
	stackConfig := getTestStackConfig(t)
	ipcConfig := genericconf.IPCConfigDefault
	ipcConfig.Path = ipcPath
	ipcConfig.Apply(stackConfig)
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.BatchPoster.Enable = true
	nodeConfig.SeqCoordinator.Enable = true
	nodeConfig.SeqCoordinator.RedisUrl = redisUrl
	nodeConfig.SeqCoordinator.MyUrlImpl = ipcPath

	return Create2ndNodeWithConfig(t, ctx, first, l1stack, l1info, l2InitData, nodeConfig, stackConfig)
}

func TestRedisForwarder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fallbackIpcPath := filepath.Join(t.TempDir(), "fallback.ipc")
	nodePaths := []string{}
	for i := 0; i < 5; i++ {
		nodePaths = append(nodePaths, filepath.Join(t.TempDir(), fmt.Sprintf("%d.ipc", i)))
	}
	redisServer, redisUrl := initMiniRedisForTest(t, ctx, nodePaths)
	defer redisServer.Close()

	l2info, fallbackNode, fallbackClient, l1info, _, _, l1stack := createFallbackSequencer(t, ctx, fallbackIpcPath, redisUrl)
	defer requireClose(t, l1stack)
	defer fallbackNode.StopAndWait()

	forwardingClient, forwardingNode := createForwardingNode(t, ctx, fallbackNode, l1stack, l1info, &l2info.ArbInitData, "", redisUrl, fallbackIpcPath)
	defer forwardingNode.StopAndWait()

	var sequencers []*arbnode.Node
	var seqClients []*ethclient.Client
	for _, path := range nodePaths {
		client, node := createSequencer(t, ctx, fallbackNode, l1stack, l1info, &l2info.ArbInitData, path, redisUrl)
		sequencers = append(sequencers, node)
		seqClients = append(seqClients, client)
	}
	clients := seqClients
	clients = append(clients, fallbackClient)
	nodes := sequencers
	nodes = append(nodes, fallbackNode)
	defer func() {
		var wg sync.WaitGroup
		for _, node := range nodes {
			if node != nil && node != fallbackNode {
				wg.Add(1)
				n := node
				go func() {
					n.StopAndWait()
					wg.Done()
				}()
			}
		}
		wg.Wait()
	}()

	for i := range clients {
		userA := fmt.Sprintf("UserA%d", i)
		l2info.GenerateAccount(userA)
		tx := l2info.PrepareTx("Owner", userA, l2info.TransferGas, big.NewInt(1e12+int64(l2info.TransferGas)*l2info.GasPrice.Int64()), nil)
		err := fallbackClient.SendTransaction(ctx, tx)
		testhelpers.RequireImpl(t, err)
		_, err = EnsureTxSucceeded(ctx, fallbackClient, tx)
		testhelpers.RequireImpl(t, err)
	}

	for i := range clients {
		userA := fmt.Sprintf("UserA%d", i)
		userB := fmt.Sprintf("UserB%d", i)
		l2info.GenerateAccount(userB)
		tx := l2info.PrepareTx(userA, userB, l2info.TransferGas, big.NewInt(1e12), nil)
		var err error
		for j := 0; j < 20; j++ {
			err = forwardingClient.SendTransaction(ctx, tx)
			if err == nil {
				break
			}
			time.Sleep(execution.DefaultTestForwarderConfig.UpdateInterval / 2)
		}
		testhelpers.RequireImpl(t, err)
		_, err = EnsureTxSucceeded(ctx, clients[i], tx)
		testhelpers.RequireImpl(t, err)
		l2balance, err := clients[i].BalanceAt(ctx, l2info.GetAddress(userB), nil)
		testhelpers.RequireImpl(t, err)
		if l2balance.Cmp(big.NewInt(1e12)) != 0 {
			testhelpers.FailImpl(t, "Unexpected balance:", l2balance)
		}
		if i < len(nodes)-1 {
			time.Sleep(100 * time.Millisecond)
			nodes[i].StopAndWait()
			nodes[i] = nil
		}
	}
}

func TestRedisForwarderFallbackNoRedis(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fallbackIpcPath := filepath.Join(t.TempDir(), "fallback.ipc")
	nodePaths := []string{}
	for i := 0; i < 10; i++ {
		nodePaths = append(nodePaths, filepath.Join(t.TempDir(), fmt.Sprintf("%d.ipc", i)))
	}
	redisServer, redisUrl := initMiniRedisForTest(t, ctx, nodePaths)
	redisServer.Close()

	l2info, fallbackNode, fallbackClient, l1info, _, _, l1stack := createFallbackSequencer(t, ctx, fallbackIpcPath, redisUrl)
	defer requireClose(t, l1stack)
	defer fallbackNode.StopAndWait()

	forwardingClient, forwardingNode := createForwardingNode(t, ctx, fallbackNode, l1stack, l1info, &l2info.ArbInitData, "", redisUrl, fallbackIpcPath)
	defer forwardingNode.StopAndWait()

	l2info.GenerateAccount("User2")
	var err error
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	for j := 0; j < 20; j++ {
		err = forwardingClient.SendTransaction(ctx, tx)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	testhelpers.RequireImpl(t, err)

	_, err = EnsureTxSucceeded(ctx, fallbackClient, tx)
	testhelpers.RequireImpl(t, err)
	l2balance, err := fallbackClient.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	testhelpers.RequireImpl(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}
