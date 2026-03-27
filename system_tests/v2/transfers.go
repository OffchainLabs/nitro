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

// testConfigTransfer: L2-only, lightweight, works with any scheme/engine/version.
// The empty Schemes/DBEngines/Min/Max means "I support everything" —
// the runner will expand this across whatever the CLI matrix requests.
func testConfigTransfer(_ TestParams) []*BuilderSpec {
	return []*BuilderSpec{{
		Weight:         WeightLight,
		Parallelizable: true,
		// Schemes:   nil → supports all
		// DBEngines: nil → supports all
		// Min/MaxArbOSVersion: 0 → no constraint
	}}
}

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
