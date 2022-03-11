//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func newBroadcasterConfigTest(port int) *wsbroadcastserver.BroadcasterConfig {
	return &wsbroadcastserver.BroadcasterConfig{
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
		URL:     fmt.Sprintf("ws://localhost:%d/feed", port),
		Timeout: 20 * time.Second,
	}
}

func TestSequencerFeed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seqNodeConfig := arbnode.NodeConfigL2Test
	seqNodeConfig.Broadcaster = true
	seqNodeConfig.BroadcasterConfig = *newBroadcasterConfigTest(0)
	l2info1, nodeA, client1 := CreateTestL2WithConfig(t, ctx, nil, &seqNodeConfig, nil, true)

	clientNodeConfig := arbnode.NodeConfigL2Test
	clientNodeConfig.BroadcastClient = true
	port := nodeA.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.BroadcastClientConfig = *newBroadcastClientConfigTest(port)

	_, nodeB, client2 := CreateTestL2WithConfig(t, ctx, nil, &clientNodeConfig, nil, false)

	l2info1.GenerateAccount("User2")

	tx := l2info1.PrepareTx("Owner", "User2", l2info1.TransferGas, big.NewInt(1e12), nil)

	err := client1.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = arbutil.EnsureTxSucceeded(ctx, client1, tx)
	Require(t, err)

	_, err = arbutil.WaitForTx(ctx, client2, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client2.BalanceAt(ctx, l2info1.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
	nodeA.StopAndWait()
	nodeB.StopAndWait()
}

func testLyingSequencer(t *testing.T, dasMode das.DataAvailabilityMode) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The truthful sequencer
	nodeConfigA := arbnode.NodeConfigL1Test
	nodeConfigA.BatchPoster = true
	nodeConfigA.Broadcaster = false
	nodeConfigA.DataAvailabilityMode = dasMode
	var dbPath string
	var err error
	defer os.RemoveAll(dbPath)
	if dasMode == das.LocalDataAvailability {
		dbPath, err = ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		nodeConfigA.DataAvailabilityConfig.LocalDiskDataDir = dbPath
	}
	l2infoA, nodeA, l2clientA, _, _, _, l1stack := CreateTestNodeOnL1WithConfig(t, ctx, true, &nodeConfigA)
	defer l1stack.Close()

	// The lying sequencer
	nodeConfigC := arbnode.NodeConfigL1Test
	nodeConfigC.BatchPoster = false
	nodeConfigC.Broadcaster = true
	nodeConfigC.DataAvailabilityMode = dasMode
	nodeConfigC.DataAvailabilityConfig.LocalDiskDataDir = dbPath
	nodeConfigC.BroadcasterConfig = *newBroadcasterConfigTest(0)
	l2clientC, nodeC := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2infoA.ArbInitData, &nodeConfigC)

	port := nodeC.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port

	// The client node, connects to lying sequencer's feed
	nodeConfigB := arbnode.NodeConfigL1Test
	nodeConfigB.Broadcaster = false
	nodeConfigB.BatchPoster = false
	nodeConfigB.BroadcastClient = true
	nodeConfigB.BroadcastClientConfig = *newBroadcastClientConfigTest(port)
	nodeConfigB.DataAvailabilityMode = dasMode
	nodeConfigB.DataAvailabilityConfig.LocalDiskDataDir = dbPath
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2infoA.ArbInitData, &nodeConfigB)

	l2infoA.GenerateAccount("FraudUser")
	l2infoA.GenerateAccount("RealUser")

	fraudTx := l2infoA.PrepareTx("Owner", "FraudUser", l2infoA.TransferGas, big.NewInt(1e12), nil)
	l2infoA.GetInfoWithPrivKey("Owner").Nonce -= 1 // Use same l2info object for different l2s
	realTx := l2infoA.PrepareTx("Owner", "RealUser", l2infoA.TransferGas, big.NewInt(1e12), nil)

	err = l2clientC.SendTransaction(ctx, fraudTx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = arbutil.EnsureTxSucceeded(ctx, l2clientC, fraudTx)
	if err != nil {
		t.Fatal(err)
	}

	// Node B should get the transaction immediately from the sequencer feed
	_, err = arbutil.WaitForTx(ctx, l2clientB, fraudTx.Hash(), time.Second*15)
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

	_, err = arbutil.EnsureTxSucceeded(ctx, l2clientA, realTx)
	if err != nil {
		t.Fatal(err)
	}

	// Node B should get the transaction after NodeC posts a batch.
	_, err = arbutil.WaitForTx(ctx, l2clientB, realTx.Hash(), time.Second*5)
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
	testLyingSequencer(t, das.OnchainDataAvailability)
}

func TestLyingSequencerLocalDAS(t *testing.T) {
	testLyingSequencer(t, das.LocalDataAvailability)
}
