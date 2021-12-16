//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/arbitrum/packages/arb-util/configuration"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/broadcastclient"
)

var feedOutputConfigTest = configuration.FeedOutput{
	Addr:          "127.0.0.1",
	IOTimeout:     5 * time.Second,
	Port:          "9642",
	Ping:          5 * time.Second,
	ClientTimeout: 15 * time.Second,
	Queue:         100,
	Workers:       100,
}

func TestSequencerFeed(t *testing.T) {
	// TODO have own config for this
	arbnode.NodeConfigL2Test.BatchPoster = true
	defer func() { arbnode.NodeConfigL2Test.BatchPoster = false }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, _ := CreateTestL2(t, ctx, &feedOutputConfigTest)

	client := l2info.Client

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", 30000, big.NewInt(1e12), nil)

	err := client.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	if err != nil {
		t.Fatal(err)
	}

	broadcastClient := broadcastclient.NewBroadcastClient("ws://127.0.0.1:9642/", nil, 20*time.Second)
	messageCount := 0

	// connect returns
	messageReceiver, err := broadcastClient.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// TODO make this test smarter, right now it just checks for receipt of 1 msg
	defer broadcastClient.Close()
	for {
		select {
		case <-messageReceiver:
			messageCount++
			if messageCount == 1 {
				return
			}
		case <-broadcastClient.ConfirmedSequenceNumberListener:
		case <-time.After(5 * time.Second):
			t.Errorf("Client expected %d mesages, only got %d messages\n", 1, messageCount)
			return
		}
	}
}
