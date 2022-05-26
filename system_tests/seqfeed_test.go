// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/relay"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func newBroadcasterConfigTest(port int) *wsbroadcastserver.BroadcasterConfig {
	return &wsbroadcastserver.BroadcasterConfig{
		Enable:        true,
		Addr:          "127.0.0.1",
		IOTimeout:     5 * time.Second,
		Port:          strconv.Itoa(port),
		Ping:          5 * time.Second,
		ClientTimeout: 15 * time.Second,
		Queue:         100,
		Workers:       100,
	}
}

func newBroadcastClientConfigTest(port int) *broadcastclient.BroadcastClientConfig {
	return &broadcastclient.BroadcastClientConfig{
		URLs:    []string{fmt.Sprintf("ws://localhost:%d/feed", port)},
		Timeout: 20 * time.Second,
	}
}

func TestSequencerFeed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seqNodeConfig := arbnode.ConfigDefaultL2Test()
	seqNodeConfig.Feed.Output = *newBroadcasterConfigTest(0)
	l2info1, nodeA, client1 := CreateTestL2WithConfig(t, ctx, nil, seqNodeConfig, true)

	clientNodeConfig := arbnode.ConfigDefaultL2Test()
	port := nodeA.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)

	_, nodeB, client2 := CreateTestL2WithConfig(t, ctx, nil, clientNodeConfig, false)

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
	nodeA.StopAndWait()
	nodeB.StopAndWait()
}

func TestRelayedSequencerFeed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seqNodeConfig := arbnode.ConfigDefaultL2Test()
	seqNodeConfig.Feed.Output = *newBroadcasterConfigTest(0)
	l2info1, nodeA, client1 := CreateTestL2WithConfig(t, ctx, nil, seqNodeConfig, true)

	relayServerConf := *newBroadcasterConfigTest(0)
	port := nodeA.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	relayClientConf := *newBroadcastClientConfigTest(port)

	relay := relay.NewRelay(relayServerConf, relayClientConf)
	err := relay.Start(ctx)
	Require(t, err)

	clientNodeConfig := arbnode.ConfigDefaultL2Test()
	port = relay.GetListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	_, nodeC, client3 := CreateTestL2WithConfig(t, ctx, nil, clientNodeConfig, false)

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

	nodeA.StopAndWait()
	relay.StopAndWait()
	nodeC.StopAndWait()
}

func testLyingSequencer(t *testing.T, dasModeStr string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The truthful sequencer
	chainConfig, nodeConfigA, _, dasSignerKey := setupConfigWithDAS(t, dasModeStr)
	nodeConfigA.BatchPoster.Enable = true
	nodeConfigA.Feed.Output.Enable = false
	nodeConfigA.DataAvailability.AllowStoreOrigination = true
	l2infoA, nodeA, l2clientA, l1info, _, l1client, l1stack := CreateTestNodeOnL1WithConfig(t, ctx, true, nodeConfigA, chainConfig)
	defer l1stack.Close()

	authorizeDASKeyset(t, ctx, dasSignerKey, l1info, l1client)

	// The lying sequencer
	nodeConfigC := arbnode.ConfigDefaultL1Test()
	nodeConfigC.BatchPoster.Enable = false
	nodeConfigC.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigC.Feed.Output = *newBroadcasterConfigTest(0)
	l2clientC, nodeC := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2infoA.ArbInitData, nodeConfigC)

	port := nodeC.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port

	// The client node, connects to lying sequencer's feed
	nodeConfigB := arbnode.ConfigDefaultL1Test()
	nodeConfigB.Feed.Output.Enable = false
	nodeConfigB.BatchPoster.Enable = false
	nodeConfigB.Feed.Input = *newBroadcastClientConfigTest(port)
	nodeConfigB.DataAvailability = nodeConfigA.DataAvailability
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2infoA.ArbInitData, nodeConfigB)

	l2infoA.GenerateAccount("FraudUser")
	l2infoA.GenerateAccount("RealUser")

	fraudTx := l2infoA.PrepareTx("Owner", "FraudUser", l2infoA.TransferGas, big.NewInt(1e12), nil)
	l2infoA.GetInfoWithPrivKey("Owner").Nonce -= 1 // Use same l2info object for different l2s
	realTx := l2infoA.PrepareTx("Owner", "RealUser", l2infoA.TransferGas, big.NewInt(1e12), nil)

	err := l2clientC.SendTransaction(ctx, fraudTx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = EnsureTxSucceeded(ctx, l2clientC, fraudTx)
	if err != nil {
		t.Fatal(err)
	}

	// Node B should get the transaction immediately from the sequencer feed
	_, err = WaitForTx(ctx, l2clientB, fraudTx.Hash(), time.Second*15)
	if err != nil {
		t.Fatal(err)
	}
	l2balance, err := l2clientB.BalanceAt(ctx, l2infoA.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	// Send the real transaction to client A
	err = l2clientA.SendTransaction(ctx, realTx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = EnsureTxSucceeded(ctx, l2clientA, realTx)
	if err != nil {
		t.Fatal(err)
	}

	// Node B should get the transaction after NodeC posts a batch.
	_, err = WaitForTx(ctx, l2clientB, realTx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal(err)
	}
	l2balanceFraudAcct, err := l2clientB.BalanceAt(ctx, l2infoA.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balanceFraudAcct.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Unexpected balance (fraud acct should be empty) was:", l2balanceFraudAcct)
	}

	l2balanceRealAcct, err := l2clientB.BalanceAt(ctx, l2infoA.GetAddress("RealUser"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balanceRealAcct.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balanceRealAcct)
	}

	nodeA.StopAndWait()
	nodeB.StopAndWait()
	nodeC.StopAndWait()
}

func TestLyingSequencer(t *testing.T) {
	testLyingSequencer(t, "onchain")
}

func TestLyingSequencerLocalDAS(t *testing.T) {
	testLyingSequencer(t, "files")
}
