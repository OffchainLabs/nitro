// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"math/big"
)

// This file demonstrates the suite grouping pattern.
// Multiple scenarios share a single node build, saving setup time.

func init() {
	RegisterSuite(SuiteEntry{
		Name:   "BalanceOperations",
		Config: L2Light(),
		Scenarios: []Scenario{
			{Name: "TransferToNewAccount", Run: scenarioTransferToNew},
			{Name: "TransferBackToOwner", Run: scenarioTransferBack},
			{Name: "CheckFaucetBalance", Run: scenarioCheckFaucet},
		},
	})
}

func scenarioTransferToNew(env *TestEnv) {
	env.L2Info.GenerateAccount("SuiteUser")
	tx := env.L2Info.PrepareTx("Owner", "SuiteUser", env.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := env.L2.Client.SendTransaction(env.Ctx, tx)
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	bal := env.L2.BalanceAt(env.Ctx, env.L2Info.GetAddress("SuiteUser"))
	if bal.Cmp(big.NewInt(1e12)) != 0 {
		env.Fatal("expected 1e12, got", bal)
	}
}

func scenarioTransferBack(env *TestEnv) {
	// This scenario reuses the "SuiteUser" created by the previous scenario.
	// This is the key benefit of suites: shared state across scenarios.
	addr := env.L2Info.GetAddress("SuiteUser")
	bal := env.L2.BalanceAt(env.Ctx, addr)
	if bal.Sign() <= 0 {
		env.Fatal("expected SuiteUser to have a balance from previous scenario")
	}
}

func scenarioCheckFaucet(env *TestEnv) {
	bal := env.L2.BalanceAt(env.Ctx, env.L2Info.GetAddress("Faucet"))
	if bal.Sign() <= 0 {
		env.Fatal("Faucet should have a positive balance")
	}
}
