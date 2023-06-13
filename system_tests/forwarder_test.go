// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
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
)

var transferAmount = big.NewInt(1e12) // amount of ether to use for transactions in tests

const nodesCount = 5 // number of testnodes to create in tests

func TestStaticForwarder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipcPath := filepath.Join(t.TempDir(), "test.ipc")
	ipcConfig := genericconf.IPCConfigDefault
	ipcConfig.Path = ipcPath
	stackConfig := testStackConfig(t)
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
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, transferAmount, nil)
	err := clientB.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, clientA, tx)
	Require(t, err)

	l2balance, err := clientA.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Cmp(transferAmount) != 0 {
		Fatal(t, "Unexpected balance:", l2balance)
	}
}

func initRedis(ctx context.Context, t *testing.T, nodeNames []string) (*miniredis.Miniredis, string) {
	t.Helper()

	redisServer, err := miniredis.Run()
	Require(t, err)

	redisUrl := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	Require(t, err)
	defer redisClient.Close()

	priorities := strings.Join(nodeNames, ",")

	Require(t, redisClient.Set(ctx, redisutil.PRIORITIES_KEY, priorities, time.Duration(0)).Err())
	return redisServer, redisUrl
}

type fallbackSequencerOpts struct {
	ipcPath              string
	redisUrl             string
	enableSecCoordinator bool
}

func fallbackSequencer(
	ctx context.Context, t *testing.T, opts *fallbackSequencerOpts,
) (l2info info, currentNode *arbnode.Node, l2client *ethclient.Client,
	l1info info, l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node) {
	stackConfig := testStackConfig(t)
	ipcConfig := genericconf.IPCConfigDefault
	ipcConfig.Path = opts.ipcPath
	ipcConfig.Apply(stackConfig)
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.SeqCoordinator.Enable = opts.enableSecCoordinator
	nodeConfig.SeqCoordinator.RedisUrl = opts.redisUrl
	nodeConfig.SeqCoordinator.MyUrlImpl = opts.ipcPath
	return createTestNodeOnL1WithConfig(t, ctx, true, nodeConfig, nil, stackConfig)
}

func createForwardingNode(
	ctx context.Context, t *testing.T,
	first *arbnode.Node,
	l1stack *node.Node,
	l1info *BlockchainTestInfo,
	l2InitData *statetransfer.ArbosInitializationInfo,
	ipcPath string,
	redisUrl string,
	fallbackPath string,
) (*ethclient.Client, *arbnode.Node) {
	stackConfig := testStackConfig(t)
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
	ctx context.Context, t *testing.T,
	first *arbnode.Node,
	l1stack *node.Node,
	l1info *BlockchainTestInfo,
	l2InitData *statetransfer.ArbosInitializationInfo,
	ipcPath string,
	redisUrl string,
) (*ethclient.Client, *arbnode.Node) {
	stackConfig := testStackConfig(t)
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

// tmpPath returns file path with specified filename from temporary directory of the test.
func tmpPath(t *testing.T, filename string) string {
	return filepath.Join(t.TempDir(), filename)
}

// testNodes creates specified number of paths for ipc from temporary directory of the test.
// e.g. /tmp/TestRedisForwarder689063006/003/0.ipc, /tmp/TestRedisForwarder689063006/003/1.ipc and so on.
func testNodes(t *testing.T, n int) []string {
	var paths []string
	for i := 0; i < n; i++ {
		paths = append(paths, tmpPath(t, fmt.Sprintf("%d.ipc", i)))
	}
	return paths
}

// waitForSequencerLockout blocks and waits until there is some sequencer chosen for specified duration.
// Errors out after timeout.
func waitForSequencerLockout(ctx context.Context, node *arbnode.Node, duration time.Duration) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}
	if node.SeqCoordinator == nil {
		return fmt.Errorf("sequence coordinator in the node is nil")
	}
	// TODO: implement exponential backoff retry mechanism and use it instead.
	for {
		select {
		case <-time.After(duration):
			return fmt.Errorf("no sequencer was chosen")
		default:
			if c, err := node.SeqCoordinator.CurrentChosenSequencer(ctx); err == nil && c != "" {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// stopNodes blocks and waits until all nodes are stopped.
func stopNodes(nodes []*arbnode.Node) {
	var wg sync.WaitGroup
	for _, node := range nodes {
		if node != nil {
			wg.Add(1)
			n := node
			go func() {
				n.StopAndWait()
				wg.Done()
			}()
		}
	}
	wg.Wait()
}

func user(suffix string, idx int) string {
	return fmt.Sprintf("User%s_%d", suffix, idx)
}

// tryWithTimeout calls function f() repeatedly foruntil it succeeds.
func tryWithTimeout(ctx context.Context, f func() error, duration time.Duration) error {
	for {
		select {
		case <-time.After(duration):
			return fmt.Errorf("timeout expired")
		default:
			if err := f(); err == nil {
				return nil
			}
		}
	}
}

func TestRedisForwarder(t *testing.T) {
	ctx := context.Background()

	nodePaths := testNodes(t, nodesCount)
	fbNodePath := tmpPath(t, "fallback.ipc") // fallback node path
	redisServer, redisUrl := initRedis(ctx, t, append(nodePaths, fbNodePath))
	defer redisServer.Close()

	l2info, fallbackNode, fallbackClient, l1info, _, _, l1stack := fallbackSequencer(ctx, t,
		&fallbackSequencerOpts{
			ipcPath:              fbNodePath,
			redisUrl:             redisUrl,
			enableSecCoordinator: true,
		})
	defer requireClose(t, l1stack)
	defer fallbackNode.StopAndWait()

	forwardingClient, forwardingNode := createForwardingNode(ctx, t, fallbackNode, l1stack, l1info, &l2info.ArbInitData, "", redisUrl, fbNodePath)
	defer forwardingNode.StopAndWait()

	var seqNodes []*arbnode.Node
	var seqClients []*ethclient.Client
	for _, path := range nodePaths {
		client, node := createSequencer(ctx, t, fallbackNode, l1stack, l1info, &l2info.ArbInitData, path, redisUrl)
		seqNodes = append(seqNodes, node)
		seqClients = append(seqClients, client)
	}
	defer stopNodes(seqNodes)

	for i := range seqClients {
		userA := user("A", i)
		l2info.GenerateAccount(userA)
		tx := l2info.PrepareTx("Owner", userA, l2info.TransferGas, big.NewInt(1e12+int64(l2info.TransferGas)*l2info.GasPrice.Int64()), nil)
		err := fallbackClient.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, fallbackClient, tx)
		Require(t, err)
	}

	for i := range seqClients {
		if err := waitForSequencerLockout(ctx, fallbackNode, 2*time.Second); err != nil {
			t.Fatalf("Error waiting for lockout: %v", err)
		}
		userA := user("A", i)
		userB := user("B", i)
		l2info.GenerateAccount(userB)
		tx := l2info.PrepareTx(userA, userB, l2info.TransferGas, transferAmount, nil)

		sendFunc := func() error { return forwardingClient.SendTransaction(ctx, tx) }
		if err := tryWithTimeout(ctx, sendFunc, execution.DefaultTestForwarderConfig.UpdateInterval*10); err != nil {
			t.Fatalf("Client: %v, error sending transaction: %v", i, err)
		}
		_, err := EnsureTxSucceeded(ctx, seqClients[i], tx)
		Require(t, err)

		l2balance, err := seqClients[i].BalanceAt(ctx, l2info.GetAddress(userB), nil)
		Require(t, err)

		if l2balance.Cmp(transferAmount) != 0 {
			Fatal(t, "Unexpected balance:", l2balance)
		}
		if i < len(seqNodes) {
			seqNodes[i].StopAndWait()
			seqNodes[i] = nil
		}
	}
}

func TestRedisForwarderFallbackNoRedis(t *testing.T) {
	ctx := context.Background()

	fallbackIpcPath := tmpPath(t, "fallback.ipc")
	nodePaths := testNodes(t, nodesCount)
	redisServer, redisUrl := initRedis(ctx, t, nodePaths)
	redisServer.Close()

	l2info, fallbackNode, fallbackClient, l1info, _, _, l1stack := fallbackSequencer(ctx, t,
		&fallbackSequencerOpts{
			ipcPath:              fallbackIpcPath,
			redisUrl:             redisUrl,
			enableSecCoordinator: false,
		})
	defer requireClose(t, l1stack)
	defer fallbackNode.StopAndWait()

	forwardingClient, forwardingNode := createForwardingNode(ctx, t, fallbackNode, l1stack, l1info, &l2info.ArbInitData, "", redisUrl, fallbackIpcPath)
	defer forwardingNode.StopAndWait()

	user := "User2"
	l2info.GenerateAccount(user)
	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, transferAmount, nil)
	sendFunc := func() error { return forwardingClient.SendTransaction(ctx, tx) }
	err := tryWithTimeout(ctx, sendFunc, execution.DefaultTestForwarderConfig.UpdateInterval*10)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, fallbackClient, tx)
	Require(t, err)

	l2balance, err := fallbackClient.BalanceAt(ctx, l2info.GetAddress(user), nil)
	Require(t, err)

	if l2balance.Cmp(transferAmount) != 0 {
		t.Errorf("Got balance: %v, want: %v", l2balance, transferAmount)
	}
}
