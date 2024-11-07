// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/util/redisutil"
)

var transferAmount = big.NewInt(1e12) // amount of ether to use for transactions in tests

const nodesCount = 5 // number of testnodes to create in tests

func TestStaticForwarder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipcPath := tmpPath(t, "test.ipc")

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.BatchPoster.Enable = false
	builder.l2StackConfig.IPCPath = ipcPath
	cleanupA := builder.Build(t)
	defer cleanupA()

	clientA := builder.L2.Client

	nodeConfigB := arbnode.ConfigDefaultL1Test()
	execConfigB := ExecConfigDefaultTest(t)
	execConfigB.Sequencer.Enable = false
	nodeConfigB.Sequencer = false
	nodeConfigB.DelayedSequencer.Enable = false
	execConfigB.Forwarder.RedisUrl = ""
	execConfigB.ForwardingTarget = ipcPath
	nodeConfigB.BatchPoster.Enable = false

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig: nodeConfigB,
		execConfig: execConfigB,
	})
	defer cleanupB()
	clientB := testClientB.Client

	builder.L2Info.GenerateAccount("User2")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, transferAmount, nil)
	err := clientB.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	l2balance, err := clientA.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
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

func fallbackSequencer(ctx context.Context, t *testing.T, opts *fallbackSequencerOpts) *NodeBuilder {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l2StackConfig.IPCPath = opts.ipcPath
	builder.nodeConfig.SeqCoordinator.Enable = opts.enableSecCoordinator
	builder.nodeConfig.SeqCoordinator.RedisUrl = opts.redisUrl
	builder.nodeConfig.SeqCoordinator.MyUrl = opts.ipcPath
	return builder
}

func createForwardingNode(t *testing.T, builder *NodeBuilder, ipcPath string, redisUrl string, fallbackPath string) (*TestClient, func()) {
	if ipcPath != "" {
		builder.l2StackConfig.IPCPath = ipcPath
	}
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.Sequencer = false
	nodeConfig.DelayedSequencer.Enable = false
	nodeConfig.BatchPoster.Enable = false
	execConfig := ExecConfigDefaultTest(t)
	execConfig.Sequencer.Enable = false
	execConfig.Forwarder.RedisUrl = redisUrl
	execConfig.ForwardingTarget = fallbackPath
	//	nodeConfig.Feed.Output.Enable = false

	return builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig, execConfig: execConfig})
}

func createSequencer(t *testing.T, builder *NodeBuilder, ipcPath string, redisUrl string) (*TestClient, func()) {
	builder.l2StackConfig.IPCPath = ipcPath
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.BatchPoster.Enable = false
	nodeConfig.SeqCoordinator.Enable = true
	nodeConfig.SeqCoordinator.RedisUrl = redisUrl
	nodeConfig.SeqCoordinator.MyUrl = ipcPath

	return builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig})
}

// tmpPath returns file path with specified filename from temporary directory of the test.
func tmpPath(t *testing.T, filename string) string {
	t.Helper()
	// create a unique, maximum 10 characters-long temporary directory {name} with path as $TMPDIR/{name}
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err = os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to cleanup temp dir: %v", err)
		}
	})
	return filepath.Join(tmpDir, filename)
}

// testNodes creates specified number of paths for ipc from temporary directory of the test.
// e.g. /tmp/TestRedisForwarder689063006/003/0.ipc, /tmp/TestRedisForwarder689063006/007/1.ipc and so on.
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
func tryWithTimeout(f func() error, duration time.Duration) error {
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

	builder := fallbackSequencer(ctx, t,
		&fallbackSequencerOpts{
			ipcPath:              fbNodePath,
			redisUrl:             redisUrl,
			enableSecCoordinator: true,
		})
	cleanup := builder.Build(t)
	defer cleanup()
	fallbackNode, fallbackClient := builder.L2.ConsensusNode, builder.L2.Client

	TestClientForwarding, cleanupForwarding := createForwardingNode(t, builder, "", redisUrl, fbNodePath)
	defer cleanupForwarding()
	forwardingClient := TestClientForwarding.Client

	var seqNodes []*arbnode.Node
	var seqClients []*ethclient.Client
	for _, path := range nodePaths {
		testClientSeq, _ := createSequencer(t, builder, path, redisUrl)
		seqNodes = append(seqNodes, testClientSeq.ConsensusNode)
		seqClients = append(seqClients, testClientSeq.Client)
	}
	defer stopNodes(seqNodes)

	for i := range seqClients {
		userA := user("A", i)
		builder.L2Info.GenerateAccount(userA)
		// #nosec G115
		tx := builder.L2Info.PrepareTx("Owner", userA, builder.L2Info.TransferGas, big.NewInt(1e12+int64(builder.L2Info.TransferGas)*builder.L2Info.GasPrice.Int64()), nil)
		err := fallbackClient.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	for i := range seqClients {
		if err := waitForSequencerLockout(ctx, fallbackNode, 2*time.Second); err != nil {
			t.Fatalf("Error waiting for lockout: %v", err)
		}
		userA := user("A", i)
		userB := user("B", i)
		builder.L2Info.GenerateAccount(userB)
		tx := builder.L2Info.PrepareTx(userA, userB, builder.L2Info.TransferGas, transferAmount, nil)

		sendFunc := func() error { return forwardingClient.SendTransaction(ctx, tx) }
		if err := tryWithTimeout(sendFunc, DefaultTestForwarderConfig.UpdateInterval*10); err != nil {
			t.Fatalf("Client: %v, error sending transaction: %v", i, err)
		}
		_, err := EnsureTxSucceeded(ctx, seqClients[i], tx)
		Require(t, err)

		l2balance, err := seqClients[i].BalanceAt(ctx, builder.L2Info.GetAddress(userB), nil)
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

	builder := fallbackSequencer(ctx, t,
		&fallbackSequencerOpts{
			ipcPath:              fallbackIpcPath,
			redisUrl:             redisUrl,
			enableSecCoordinator: false,
		})
	cleanup := builder.Build(t)
	defer cleanup()
	fallbackClient := builder.L2.Client

	TestClientForwarding, cleanupForwarding := createForwardingNode(t, builder, "", redisUrl, fallbackIpcPath)
	defer cleanupForwarding()
	forwardingClient := TestClientForwarding.Client

	user := "User2"
	builder.L2Info.GenerateAccount(user)
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, transferAmount, nil)
	sendFunc := func() error { return forwardingClient.SendTransaction(ctx, tx) }
	err := tryWithTimeout(sendFunc, DefaultTestForwarderConfig.UpdateInterval*10)
	Require(t, err)

	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	l2balance, err := fallbackClient.BalanceAt(ctx, builder.L2Info.GetAddress(user), nil)
	Require(t, err)

	if l2balance.Cmp(transferAmount) != 0 {
		t.Errorf("Got balance: %v, want: %v", l2balance, transferAmount)
	}
}
