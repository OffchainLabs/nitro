// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package smoketest

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var l1Tests = []systest.Scenario{
	systest.Test(testRunDepositETH, systest.WithL1()),
	systest.Test(testRunSendSignedTxViaL1, systest.WithL1()),
}

// testRunDepositETH drives an ETH deposit through the delayed inbox, resolving
// it on L2 via LookupL2Tx.
func testRunDepositETH(env *systest.Env) {
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

// testRunSendSignedTxViaL1 drives signed L2 txs through the delayed inbox,
// single then batched, and checks they land on L2.
func testRunSendSignedTxViaL1(env *systest.Env) {
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
