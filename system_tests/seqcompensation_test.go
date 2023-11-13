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
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	TestClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	builder.L2Info.GenerateAccount("User2")

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err = WaitForTx(ctx, TestClientB.Client, tx.Hash(), time.Second*5)
	Require(t, err)

	// clientB sees balance means sequencer message was sent
	l2balance, err := TestClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		Fatal(t, "Unexpected balance:", l2balance)
	}

	initialSeqBalance, err := TestClientB.Client.BalanceAt(ctx, l1pricing.BatchPosterAddress, big.NewInt(0))
	Require(t, err)
	if initialSeqBalance.Sign() != 0 {
		Fatal(t, "Unexpected initial sequencer balance:", initialSeqBalance)
	}
}
