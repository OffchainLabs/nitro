// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/relay"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func newBroadcasterConfigTest() *wsbroadcastserver.BroadcasterConfig {
	config := wsbroadcastserver.DefaultTestBroadcasterConfig
	config.Enable = true
	config.Port = "0"
	return &config
}

func newBroadcastClientConfigTest(port int) *broadcastclient.Config {
	return &broadcastclient.Config{
		URL:     []string{fmt.Sprintf("ws://localhost:%d/feed", port)},
		Timeout: 200 * time.Millisecond,
		Verify: signature.VerifierConfig{
			Dangerous: signature.DangerousVerifierConfig{
				AcceptMissing: true,
			},
		},
	}
}

func TestSequencerFeed(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seqNodeConfig := arbnode.ConfigDefaultL2Test()
	seqNodeConfig.Feed.Output = *newBroadcasterConfigTest()
	testNode1 := NewNodeBuilder(ctx).SetNodeConfig(seqNodeConfig).CreateTestNodeOnL2Only(t, true)
	defer testNode1.L2Node.StopAndWait()
	clientNodeConfig := arbnode.ConfigDefaultL2Test()
	port := testNode1.L2Node.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)

	testNode2 := NewNodeBuilder(ctx).SetNodeConfig(clientNodeConfig).CreateTestNodeOnL2Only(t, false)
	defer testNode2.L2Node.StopAndWait()

	testNode1.L2Info.GenerateAccount("User2")

	tx := testNode1.L2Info.PrepareTx("Owner", "User2", testNode1.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := testNode1.L2Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, testNode1.L2Client, tx)
	Require(t, err)

	_, err = WaitForTx(ctx, testNode2.L2Client, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := testNode2.L2Client.BalanceAt(ctx, testNode1.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}

func TestRelayedSequencerFeed(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seqNodeConfig := arbnode.ConfigDefaultL2Test()
	seqNodeConfig.Feed.Output = *newBroadcasterConfigTest()
	testNode1 := NewNodeBuilder(ctx).SetNodeConfig(seqNodeConfig).CreateTestNodeOnL2Only(t, true)
	defer testNode1.L2Node.StopAndWait()

	bigChainId, err := testNode1.L2Client.ChainID(ctx)
	Require(t, err)

	config := relay.ConfigDefault
	port := testNode1.L2Node.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	config.Node.Feed.Input = *newBroadcastClientConfigTest(port)
	config.Node.Feed.Output = *newBroadcasterConfigTest()
	config.Chain.ID = bigChainId.Uint64()

	feedErrChan := make(chan error, 10)
	currentRelay, err := relay.NewRelay(&config, feedErrChan)
	Require(t, err)
	err = currentRelay.Start(ctx)
	Require(t, err)
	defer currentRelay.StopAndWait()

	clientNodeConfig := arbnode.ConfigDefaultL2Test()
	port = currentRelay.GetListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	testNode3 := NewNodeBuilder(ctx).SetNodeConfig(clientNodeConfig).CreateTestNodeOnL2Only(t, false)
	defer testNode3.L2Node.StopAndWait()
	StartWatchChanErr(t, ctx, feedErrChan, testNode3.L2Node)

	testNode1.L2Info.GenerateAccount("User2")

	tx := testNode1.L2Info.PrepareTx("Owner", "User2", testNode1.L2Info.TransferGas, big.NewInt(1e12), nil)

	err = testNode1.L2Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, testNode1.L2Client, tx)
	Require(t, err)

	_, err = WaitForTx(ctx, testNode3.L2Client, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := testNode3.L2Client.BalanceAt(ctx, testNode1.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}

func testLyingSequencer(t *testing.T, dasModeStr string) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The truthful sequencer
	chainConfig, nodeConfigA, lifecycleManager, _, dasSignerKey := setupConfigWithDAS(t, ctx, dasModeStr)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	nodeConfigA.BatchPoster.Enable = true
	nodeConfigA.Feed.Output.Enable = false
	testNodeA := NewNodeBuilder(ctx).SetNodeConfig(nodeConfigA).SetChainConfig(chainConfig).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNodeA.L1Stack, "unable to close l1Stack")
	defer testNodeA.L2Node.StopAndWait()

	authorizeDASKeyset(t, ctx, dasSignerKey, testNodeA.L1Info, testNodeA.L1Client)

	// The lying sequencer
	nodeConfigC := arbnode.ConfigDefaultL1Test()
	nodeConfigC.BatchPoster.Enable = false
	nodeConfigC.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigC.DataAvailability.RPCAggregator.Enable = false
	nodeConfigC.Feed.Output = *newBroadcasterConfigTest()
	l2clientC, nodeC := Create2ndNodeWithConfig(t, ctx, testNodeA.L2Node, testNodeA.L1Stack, testNodeA.L1Info, &testNodeA.L2Info.ArbInitData, nodeConfigC, nil)
	defer nodeC.StopAndWait()

	port := nodeC.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port

	// The client node, connects to lying sequencer's feed
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.Feed.Output.Enable = false
	nodeConfigB.Feed.Input = *newBroadcastClientConfigTest(port)
	nodeConfigB.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigB.DataAvailability.RPCAggregator.Enable = false
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, testNodeA.L2Node, testNodeA.L1Stack, testNodeA.L1Info, &testNodeA.L2Info.ArbInitData, nodeConfigB, nil)
	defer nodeB.StopAndWait()

	testNodeA.L2Info.GenerateAccount("FraudUser")
	testNodeA.L2Info.GenerateAccount("RealUser")

	fraudTx := testNodeA.L2Info.PrepareTx("Owner", "FraudUser", testNodeA.L2Info.TransferGas, big.NewInt(1e12), nil)
	testNodeA.L2Info.GetInfoWithPrivKey("Owner").Nonce -= 1 // Use same l2info object for different l2s
	realTx := testNodeA.L2Info.PrepareTx("Owner", "RealUser", testNodeA.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := l2clientC.SendTransaction(ctx, fraudTx)
	if err != nil {
		t.Fatal("error sending fraud transaction:", err)
	}

	_, err = EnsureTxSucceeded(ctx, l2clientC, fraudTx)
	if err != nil {
		t.Fatal("error ensuring fraud transaction succeeded:", err)
	}

	// Node B should get the transaction immediately from the sequencer feed
	_, err = WaitForTx(ctx, l2clientB, fraudTx.Hash(), time.Second*15)
	if err != nil {
		t.Fatal("error waiting for tx:", err)
	}
	l2balance, err := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal("error getting balance:", err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	// Send the real transaction to client A
	err = testNodeA.L2Client.SendTransaction(ctx, realTx)
	if err != nil {
		t.Fatal("error sending real transaction:", err)
	}

	_, err = EnsureTxSucceeded(ctx, testNodeA.L2Client, realTx)
	if err != nil {
		t.Fatal("error ensuring real transaction succeeded:", err)
	}

	// Node B should get the transaction after NodeC posts a batch.
	_, err = WaitForTx(ctx, l2clientB, realTx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal("error waiting for transaction to get to node b:", err)
	}
	l2balanceFraudAcct, err := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal("error getting fraud balance:", err)
	}
	if l2balanceFraudAcct.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Unexpected balance (fraud acct should be empty) was:", l2balanceFraudAcct)
	}

	l2balanceRealAcct, err := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("RealUser"), nil)
	if err != nil {
		t.Fatal("error getting real balance:", err)
	}
	if l2balanceRealAcct.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance of real account:", l2balanceRealAcct)
	}
}

func TestLyingSequencer(t *testing.T) {
	testLyingSequencer(t, "onchain")
}

func TestLyingSequencerLocalDAS(t *testing.T) {
	testLyingSequencer(t, "files")
}
