// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"fmt"
	"math/big"
)

func init() {
	RegisterTest("TestTransfer", testConfigTransfer, testRunTransfer)
}

// testConfigTransfer declares the requirements for TestTransfer.
//
// This test is L2-only (no L1 needed), lightweight, and safe to run in
// parallel. It has no ArbOS version constraint and no state-scheme preference,
// so it always returns exactly one BuilderSpec regardless of TestParams.
func testConfigTransfer(_ TestParams) []*BuilderSpec {
	return []*BuilderSpec{{
		NeedsL1:        false,
		Weight:         WeightLight,
		Parallelizable: true,
	}}
}

// testRunTransfer contains only the test logic for TestTransfer.
//
// It receives a fully-built TestEnv from the runner. It never creates or
// cancels contexts, never calls cleanup — the runner owns all of that.
// The logic is a direct port of the original TestTransfer in
// system_tests/transfer_test.go.
func testRunTransfer(env *TestEnv) {
	env.L2Info.GenerateAccount("User2")

	tx := env.L2Info.PrepareTx("Owner", "User2", env.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := env.L2.Client.SendTransaction(env.Ctx, tx)
	env.Require(err)

	env.L2.WaitForTx(env.T, env.Ctx, tx)

	bal := env.L2.BalanceAt(env.Ctx, env.L2Info.GetAddress("Owner"))
	fmt.Println("Owner balance is:", bal)

	bal2 := env.L2.BalanceAt(env.Ctx, env.L2Info.GetAddress("User2"))
	if bal2.Cmp(big.NewInt(1e12)) != 0 {
		env.T.Fatalf("unexpected recipient balance: got %v, want 1e12", bal2)
	}
}
