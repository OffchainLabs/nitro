// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
)

// L1 Pricer pool address gets something when the sequencer posts batches
func TestSequencerCompensation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	feedErrChan := make(chan error, 10)
	l2info, node1, l2clientA, l2stackA, l1info, _, l1client, l1stack := CreateTestNodeOnL1(t, ctx, true, feedErrChan)
	defer requireClose(t, l1stack)
	defer requireClose(t, l2stackA)

	l2clientB, _, l2stackB := Create2ndNode(t, ctx, node1, l1stack, &l2info.ArbInitData, nil, feedErrChan)
	defer requireClose(t, l2stackB)

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err := l2clientA.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2clientA, tx, feedErrChan)
	Require(t, err)

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		}, feedErrChan)
	}

	_, err = WaitForTx(ctx, l2clientB, tx.Hash(), feedErrChan, time.Second*5)
	Require(t, err)

	// clientB sees balance means sequencer message was sent
	l2balance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}

	initialSeqBalance, err := l2clientB.BalanceAt(ctx, l1pricing.BatchPosterAddress, big.NewInt(0))
	Require(t, err)
	if initialSeqBalance.Sign() != 0 {
		Fail(t, "Unexpected initial sequencer balance:", initialSeqBalance)
	}
}
