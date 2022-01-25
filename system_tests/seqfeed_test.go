//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/broadcastclient"
	"github.com/offchainlabs/arbstate/wsbroadcastserver"
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
	port := 9642
	seqNodeConfig := arbnode.NodeConfigL2Test
	seqNodeConfig.Broadcaster = true
	seqNodeConfig.BroadcasterConfig = *newBroadcasterConfigTest(port)

	clientNodeConfig := arbnode.NodeConfigL2Test
	clientNodeConfig.BroadcastClient = true
	clientNodeConfig.BroadcastClientConfig = *newBroadcastClientConfigTest(port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info1, _, client1 := CreateTestL2WithConfig(t, ctx, nil, &seqNodeConfig, true)
	_, _, client2 := CreateTestL2WithConfig(t, ctx, nil, &clientNodeConfig, false)

	l2info1.GenerateAccount("User2")

	tx := l2info1.PrepareTx("Owner", "User2", l2TxGas, big.NewInt(1e12), nil)

	err := client1.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = arbnode.EnsureTxSucceeded(ctx, client1, tx)
	Require(t, err)

	_, err = arbnode.WaitForTx(ctx, client2, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client2.BalanceAt(ctx, l2info1.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}

func TestLyingSequencer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := 9643

	// The truthful sequencer
	nodeConfigA := arbnode.NodeConfigL1Test
	nodeConfigA.BatchPoster = true
	nodeConfigA.Broadcaster = false
	l2infoA, nodeA, l2clientA, _, _, _, l1stack := CreateTestNodeOnL1WithConfig(t, ctx, true, &nodeConfigA)
	defer l1stack.Close()

	// The lying sequencer
	nodeConfigC := arbnode.NodeConfigL1Test
	nodeConfigC.BatchPoster = false
	nodeConfigC.Broadcaster = true
	nodeConfigC.BroadcasterConfig = *newBroadcasterConfigTest(port)
	l2clientC, _ := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2infoA.ArbInitData, &nodeConfigC)

	// The client node, connects to lying sequencer's feed
	nodeConfigB := arbnode.NodeConfigL1Test
	nodeConfigB.Broadcaster = false
	nodeConfigB.BatchPoster = false
	nodeConfigB.BroadcastClient = true
	nodeConfigB.BroadcastClientConfig = *newBroadcastClientConfigTest(port)
	l2clientB, _ := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2infoA.ArbInitData, &nodeConfigB)

	l2infoA.GenerateAccount("FraudUser")
	l2infoA.GenerateAccount("RealUser")

	fraudTx := l2infoA.PrepareTx("Owner", "FraudUser", l2TxGas, big.NewInt(1e12), nil)
	l2infoA.GetInfoWithPrivKey("Owner").Nonce -= 1 // Use same l2info object for different l2s
	realTx := l2infoA.PrepareTx("Owner", "RealUser", l2TxGas, big.NewInt(1e12), nil)

	err := l2clientC.SendTransaction(ctx, fraudTx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = arbnode.EnsureTxSucceeded(ctx, l2clientC, fraudTx)
	if err != nil {
		t.Fatal(err)
	}

	// Node B should get the transaction immediately from the sequencer feed
	_, err = arbnode.WaitForTx(ctx, l2clientB, fraudTx.Hash(), time.Second*5)
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

	_, err = arbnode.EnsureTxSucceeded(ctx, l2clientA, realTx)
	if err != nil {
		t.Fatal(err)
	}

	// Node B should get the transaction after NodeC posts a batch.
	_, err = arbnode.WaitForTx(ctx, l2clientB, realTx.Hash(), time.Second*5)
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
}
