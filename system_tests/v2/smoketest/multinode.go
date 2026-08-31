// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package smoketest

import (
	"math/big"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var multiNodeTests = []systest.Scenario{
	systest.Test(testRunMultiNodeSync, systest.WithMultiNode()),
}

func testRunMultiNodeSync(env *systest.Env) {
	env.L2.Info.GenerateAccount("User2")

	tx := env.L2.Info.PrepareTx("Owner", "User2", env.L2.Info.TransferGas, big.NewInt(1e12), nil)
	env.L2.SendTx(tx)
	env.L2.EnsureTxSucceeded(tx)

	env.WaitForFollowersSync()

	bal := env.Follower().BalanceAt(env.L2.Info.GetAddress("User2"))
	env.EqualBig(big.NewInt(1e12), bal, "follower balance")
}
