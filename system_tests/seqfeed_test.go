//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/broadcastclient"
	"github.com/offchainlabs/arbstate/wsbroadcastserver"
)

var broadcasterConfigTest = wsbroadcastserver.BroadcasterConfig{
	Addr:          "127.0.0.1",
	IOTimeout:     5 * time.Second,
	Port:          "9642",
	Ping:          5 * time.Second,
	ClientTimeout: 15 * time.Second,
	Queue:         100,
	Workers:       100,
}

var broadcastClientConfigTest = broadcastclient.BroadcastClientConfig{
	URL:     "ws://localhost:9642/feed",
	Timeout: 20 * time.Second,
}

func TestSequencerFeed(t *testing.T) {
	glogger := log.Root().GetHandler().(*log.GlogHandler)
	glogger.Verbosity(log.LvlTrace)
	defer func() { glogger.Verbosity(log.LvlInfo) }()

	seqNodeConfig := arbnode.NodeConfigL2Test
	seqNodeConfig.BatchPoster = true
	seqNodeConfig.Broadcaster = true
	seqNodeConfig.BroadcasterConfig = broadcasterConfigTest

	clientNodeConfig := arbnode.NodeConfigL2Test
	clientNodeConfig.BroadcastClient = true
	clientNodeConfig.BroadcastClientConfig = broadcastClientConfigTest

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info1, _ := CreateTestL2WithConfig(t, ctx, &seqNodeConfig)
	l2info2, _ := CreateTestL2WithConfig(t, ctx, &clientNodeConfig)

	client1 := l2info1.Client
	l2info1.GenerateAccount("User2")

	tx := l2info1.PrepareTx("Owner", "User2", 30000, big.NewInt(1e12), nil)

	err := client1.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = arbnode.EnsureTxSucceeded(ctx, client1, tx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = arbnode.WaitForTx(ctx, l2info2.Client, tx.Hash(), time.Second*15)
	if err != nil {
		t.Fatal(err)
	}
	l2balance, err := l2info2.Client.BalanceAt(ctx, l2info1.GetAddress("User2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}
