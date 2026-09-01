// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package messaging holds L1-backed tests: delayed inbox, deposits, bridging.
package messaging

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var delayedInboxTests = []systest.Scenario{
	systest.Test(testRunDelayInboxSimple, systest.WithL1()),
	systest.Test(testRunDelayInboxBatch, systest.WithL1()),
	systest.Test(testRunDepositETH, systest.WithL1()),
}

func testRunDelayInboxSimple(env *systest.Env) {
	env.L2.Info.GenerateAccount("User2")

	delayedTx := env.L2.Info.PrepareTx("Owner", "User2", 50001, big.NewInt(1e6), nil)
	env.SendSignedTxViaL1(delayedTx)

	bal := env.L2.BalanceAt(env.L2.Info.GetAddress("User2"))
	env.EqualBig(big.NewInt(1e6), bal, "unexpected balance after delayed-inbox transfer")
}

// testRunDelayInboxBatch posts several signed L2 txs through the delayed inbox
// in one L1 message and checks they all land.
func testRunDelayInboxBatch(env *systest.Env) {
	env.L2.Info.GenerateAccount("User2")
	user2 := env.L2.Info.GetAddress("User2")

	var txes types.Transactions
	for range 3 {
		txes = append(txes, env.L2.Info.PrepareTx("Owner", "User2", env.L2.Info.TransferGas, big.NewInt(1e12), nil))
	}
	receipts := env.SendSignedTxBatchViaL1(txes)
	env.Len(receipts, 3, "batched delayed tx receipts")
	env.EqualBig(big.NewInt(3e12), env.L2.BalanceAt(user2), "balance after batched delayed txs")
}

// Migrated from system_tests/retryable_test.go::TestDepositETH (without the
// flat-call-tracer tail)
func testRunDepositETH(env *systest.Env) {
	// Deposit from L1 "User" (untouched on L2) so the credit can't overflow
	// uint256 like the near-max Faucet would.
	txOpts := env.L1.TransactOpts("User")
	txOpts.Value = big.NewInt(13)

	// EOA deposits credit the sender un-aliased; same address on L1/L2.
	depositTarget := txOpts.From
	oldBalance, err := env.L2.Client.BalanceAt(env.Ctx, depositTarget, nil)
	env.Require(err, "L2 BalanceAt")

	l1tx, err := env.L1.DelayedInbox().DepositEth439370b1(&txOpts)
	env.Require(err, "DepositEth")
	l1Receipt := env.L1.EnsureTxSucceeded(l1tx)

	env.L1.WaitForDelayBlocks()
	l2Receipt := env.L2.EnsureTxSucceeded(env.LookupL2Tx(l1Receipt))

	newBalance, err := env.L2.Client.BalanceAt(env.Ctx, depositTarget, l2Receipt.BlockNumber)
	env.Require(err, "L2 BalanceAt")
	env.EqualBig(txOpts.Value, new(big.Int).Sub(newBalance, oldBalance), "unexpected deposited amount")
}
