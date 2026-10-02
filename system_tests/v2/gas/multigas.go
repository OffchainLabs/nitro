// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gas

import (
	"math/big"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var multigasTests = []systest.Scenario{
	systest.Test(testRunMultigasDataFromReceipts),
	systest.Test(testRunMultigasDataCanBeDisabled,
		systest.WithDBEngine(systest.DBEnginePebble),
		systest.WithExecConfigOverride(func(cfg *gethexec.Config) { cfg.ExposeMultiGas = false })),
}

func testRunMultigasDataFromReceipts(env *systest.Env) {
	env.L2.Info.GenerateAccount("Alice")
	for i := range 20 {
		// unique value to avoid duplicate txs
		value := big.NewInt(1e12 + int64(i))
		tx := env.L2.Info.PrepareTx("Owner", "Alice", env.L2.Info.TransferGas, value, nil)
		env.L2.SendTx(tx)
		rcpt := env.L2.EnsureTxSucceeded(tx)
		env.Equal(rcpt.GasUsed, rcpt.MultiGasUsed.SingleGas(), "GasUsed != MultiGasUsed.SingleGas")
	}
}

func testRunMultigasDataCanBeDisabled(env *systest.Env) {
	tx := env.L2.Info.PrepareTx("Owner", "Owner", env.L2.Info.TransferGas, big.NewInt(1), nil)
	env.L2.SendTx(tx)
	receipt := env.L2.EnsureTxSucceeded(tx)
	env.True(receipt.MultiGasUsed.IsZero(), "expected MultiGasUsed to be zero when ExposeMultiGas is disabled")
}
