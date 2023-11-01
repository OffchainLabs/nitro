// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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

	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestDelayInboxLong(t *testing.T) {
	t.Parallel()
	addLocalLoops := 3
	messagesPerAddLocal := 1000
	messagesPerDelayed := 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	fundsPerDelayed := int64(1000000)
	delayedMessages := int64(0)

	var lastDelayedMessage *types.Transaction

	for i := 0; i < addLocalLoops; i++ {
		l1Txs := make([]*types.Transaction, 0, messagesPerAddLocal)
		for len(l1Txs) < messagesPerAddLocal {
			randNum := rand.Int() % messagesPerDelayed
			var l1tx *types.Transaction
			if randNum == 0 {
				delayedTx := builder.L2Info.PrepareTx("Owner", "User2", 50001, big.NewInt(fundsPerDelayed), nil)
				l1tx = WrapL2ForDelayed(t, delayedTx, builder.L1Info, "User", 100000)
				lastDelayedMessage = delayedTx
				delayedMessages++
			} else {
				l1tx = builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil)
			}
			l1Txs = append(l1Txs, l1tx)
		}
		wrappedL1Txs := make([]*txpool.Transaction, 0, messagesPerAddLocal)
		for _, tx := range l1Txs {
			wrappedL1Txs = append(wrappedL1Txs, &txpool.Transaction{Tx: tx})
		}
		// adding multiple messages in the same Add with local=true to get them in the same L1 block
		errs := builder.L1.L1Backend.TxPool().Add(wrappedL1Txs, true, false)
		for _, err := range errs {
			Require(t, err)
		}
		// Checking every tx is expensive, so we just check the last, assuming that the others succeeded too
		confirmLatestBlock(ctx, t, builder.L1Info, builder.L1.Client)
		_, err := builder.L1.EnsureTxSucceeded(l1Txs[len(l1Txs)-1])
		Require(t, err)
	}

	t.Log("Done sending", delayedMessages, "delayedMessages")
	if delayedMessages == 0 {
		Fatal(t, "No delayed messages sent!")
	}

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 100; i++ {
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err := WaitForTx(ctx, builder.L2.Client, lastDelayedMessage.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(fundsPerDelayed*delayedMessages)) != 0 {
		Fatal(t, "Unexpected balance:", "balance", l2balance, "expected", fundsPerDelayed*delayedMessages)
	}
}
