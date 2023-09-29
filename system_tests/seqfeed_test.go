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
	l2info1, nodeA, client1 := CreateTestL2WithConfig(t, ctx, nil, seqNodeConfig, true, nil)
	defer nodeA.StopAndWait()
	clientNodeConfig := arbnode.ConfigDefaultL2Test()
	port := nodeA.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)

	_, nodeB, client2 := CreateTestL2WithConfig(t, ctx, nil, clientNodeConfig, false, nil)
	defer nodeB.StopAndWait()

	l2info1.GenerateAccount("User2")

	tx := l2info1.PrepareTx("Owner", "User2", l2info1.TransferGas, big.NewInt(1e12), nil)

	err := client1.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, client1, tx)
	Require(t, err)

	_, err = WaitForTx(ctx, client2, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client2.BalanceAt(ctx, l2info1.GetAddress("User2"), nil)
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
	l2info1, nodeA, client1 := CreateTestL2WithConfig(t, ctx, nil, seqNodeConfig, true, nil)
	defer nodeA.StopAndWait()

	bigChainId, err := client1.ChainID(ctx)
	Require(t, err)

	config := relay.ConfigDefault
	port := nodeA.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
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
	_, nodeC, client3 := CreateTestL2WithConfig(t, ctx, nil, clientNodeConfig, false, nil)
	defer nodeC.StopAndWait()
	StartWatchChanErr(t, ctx, feedErrChan, nodeC)

	l2info1.GenerateAccount("User2")

	tx := l2info1.PrepareTx("Owner", "User2", l2info1.TransferGas, big.NewInt(1e12), nil)

	err = client1.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, client1, tx)
	Require(t, err)

	_, err = WaitForTx(ctx, client3, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client3.BalanceAt(ctx, l2info1.GetAddress("User2"), nil)
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
	l2infoA, nodeA, l2clientA, l1info, _, l1client, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nodeConfigA, chainConfig, nil)
	defer requireClose(t, l1stack, "unable to close l1stack")
	defer nodeA.StopAndWait()

	authorizeDASKeyset(t, ctx, dasSignerKey, l1info, l1client)

	// The lying sequencer
	nodeConfigC := arbnode.ConfigDefaultL1Test()
	nodeConfigC.BatchPoster.Enable = false
	nodeConfigC.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigC.DataAvailability.RPCAggregator.Enable = false
	nodeConfigC.Feed.Output = *newBroadcasterConfigTest()
	l2clientC, nodeC := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, l1info, &l2infoA.ArbInitData, nodeConfigC, nil)
	defer nodeC.StopAndWait()

	port := nodeC.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port

	// The client node, connects to lying sequencer's feed
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.Feed.Output.Enable = false
	nodeConfigB.Feed.Input = *newBroadcastClientConfigTest(port)
	nodeConfigB.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigB.DataAvailability.RPCAggregator.Enable = false
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, l1info, &l2infoA.ArbInitData, nodeConfigB, nil)
	defer nodeB.StopAndWait()

	l2infoA.GenerateAccount("FraudUser")
	l2infoA.GenerateAccount("RealUser")

	fraudTx := l2infoA.PrepareTx("Owner", "FraudUser", l2infoA.TransferGas, big.NewInt(1e12), nil)
	l2infoA.GetInfoWithPrivKey("Owner").Nonce -= 1 // Use same l2info object for different l2s
	realTx := l2infoA.PrepareTx("Owner", "RealUser", l2infoA.TransferGas, big.NewInt(1e12), nil)

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
	l2balance, err := l2clientB.BalanceAt(ctx, l2infoA.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal("error getting balance:", err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	// Send the real transaction to client A
	err = l2clientA.SendTransaction(ctx, realTx)
	if err != nil {
		t.Fatal("error sending real transaction:", err)
	}

	_, err = EnsureTxSucceeded(ctx, l2clientA, realTx)
	if err != nil {
		t.Fatal("error ensuring real transaction succeeded:", err)
	}

	// Node B should get the transaction after NodeC posts a batch.
	_, err = WaitForTx(ctx, l2clientB, realTx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal("error waiting for transaction to get to node b:", err)
	}
	l2balanceFraudAcct, err := l2clientB.BalanceAt(ctx, l2infoA.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal("error getting fraud balance:", err)
	}
	if l2balanceFraudAcct.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Unexpected balance (fraud acct should be empty) was:", l2balanceFraudAcct)
	}

	l2balanceRealAcct, err := l2clientB.BalanceAt(ctx, l2infoA.GetAddress("RealUser"), nil)
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
