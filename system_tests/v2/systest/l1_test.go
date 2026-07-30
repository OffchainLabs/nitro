// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

// TestL1HelperOnL2OnlyEnvFails pins the requireL1 guard: L1 helpers on an
// L2-only scenario fail with a pointer to WithL1.
func TestL1HelperOnL2OnlyEnvFails(t *testing.T) {
	tb := &recordingT{}
	e := &Env{t: tb}
	done := make(chan struct{})
	go func() {
		defer close(done)
		e.ParentChain()
	}()
	<-done
	if tb.errCount() != 1 {
		t.Fatalf("L1 helper on an L2-only env must record exactly 1 error, got %d", tb.errCount())
	}
	if !strings.Contains(tb.errors[0], "systest.WithL1()") {
		t.Fatalf("failure must point at WithL1, got %q", tb.errors[0])
	}
}

// TestBuildL1L2DepositAndLookupL2Tx builds the WithL1 topology and drives an
// ETH deposit through the delayed inbox, resolving it on L2 via LookupL2Tx.
func TestBuildL1L2DepositAndLookupL2Tx(t *testing.T) {
	ctx := t.Context()

	b := newBuilder()
	b.name = "L1L2Deposit"
	WithL1()(b)
	env, cleanup := buildNode(t, ctx, b.freeze(""), overrides{})
	defer cleanup()

	if env.L1 == nil {
		t.Fatal("WithL1 topology must populate env.L1")
	}

	txOpts := env.ParentChain().TransactOpts("User")
	txOpts.Value = big.NewInt(13)
	oldBalance := env.L2.BalanceAt(txOpts.From)

	l1tx, err := env.DelayedInbox().DepositEth439370b1(&txOpts)
	env.Require(err, "DepositEth")
	l1Receipt := env.ParentChain().EnsureTxSucceeded(l1tx)
	env.WaitForL1DelayBlocks()

	env.L2.EnsureTxSucceeded(env.LookupL2Tx(l1Receipt))

	newBalance := env.L2.BalanceAt(txOpts.From)
	env.EqualBig(new(big.Int).Add(oldBalance, txOpts.Value), newBalance, "L2 balance after deposit")
}

// TestSendSignedTxViaL1 drives signed L2 txs through the delayed inbox, single
// then batched, and checks they land on L2.
func TestSendSignedTxViaL1(t *testing.T) {
	ctx := t.Context()

	b := newBuilder()
	b.name = "SendViaL1"
	WithL1()(b)
	env, cleanup := buildNode(t, ctx, b.freeze(""), overrides{})
	defer cleanup()

	env.L2.Info.GenerateAccount("User2")
	user2 := env.L2.Info.GetAddress("User2")

	tx := env.L2.Info.PrepareTx("Owner", "User2", env.L2.Info.TransferGas, big.NewInt(1e12), nil)
	env.SendSignedTxViaL1(tx)
	env.EqualBig(big.NewInt(1e12), env.L2.BalanceAt(user2), "balance after single delayed tx")

	var txes types.Transactions
	for range 3 {
		txes = append(txes, env.L2.Info.PrepareTx("Owner", "User2", env.L2.Info.TransferGas, big.NewInt(1e12), nil))
	}
	receipts := env.SendSignedTxBatchViaL1(txes)
	env.Len(receipts, 3, "batched delayed tx receipts")
	env.EqualBig(big.NewInt(4e12), env.L2.BalanceAt(user2), "balance after batched delayed txs")
}
