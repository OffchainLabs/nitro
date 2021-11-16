//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/arbnode"
)

func TestDelayInboxLong(t *testing.T) {
	addLocalLoops := 3
	messagesPerAddLocal := 1000
	messagesPerDelayed := 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, _, l1info, l1backend, stack := CreateTestNodeOnL1(t, ctx, true)
	defer stack.Close()

	l2client := l2info.Client
	l1client := l1info.Client
	l2info.GenerateAccount("User2")

	fundsPerDelayed := int64(1000000)
	delayedMessages := int64(0)

	var lastDelayedMessage *types.Transaction

	for i := 0; i < addLocalLoops; i++ {
		l1Txs := make([]*types.Transaction, 0, messagesPerAddLocal)
		for len(l1Txs) < messagesPerAddLocal {
			randNum := rand.Int() % messagesPerDelayed
			var l1tx *types.Transaction
			if randNum == 0 {
				delayedTx := l2info.PrepareTx("Owner", "User2", 50001, big.NewInt(fundsPerDelayed), nil)
				l1tx = WrapL2ForDelayed(t, delayedTx, l1info, "User", 100000)
				lastDelayedMessage = delayedTx
				delayedMessages++
			} else {
				l1tx = l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil)
			}
			l1Txs = append(l1Txs, l1tx)
		}
		// adding multiple messages in the same AddLocal to get them in the same L1 block
		errs := l1backend.TxPool().AddLocals(l1Txs)
		for _, err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, l1tx := range l1Txs {
			_, err := arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	t.Log("Done sending", delayedMessages, "delayedMessages")
	if delayedMessages == 0 {
		t.Fatal("No delayed messages sent!")
	}

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 100; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil),
		})
		// give the inbox reader a bit of time to pick up the delayed message
		time.Sleep(time.Millisecond * 10)
	}

	_, err := arbnode.WaitForTx(ctx, l2client, lastDelayedMessage.Hash(), time.Second*5)
	if err != nil {
		t.Fatal(err)
	}
	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balance.Cmp(big.NewInt(fundsPerDelayed*delayedMessages)) != 0 {
		t.Fatal("Unexpected balance:", "balance", l2balance, "expected", fundsPerDelayed*delayedMessages)
	}
}
