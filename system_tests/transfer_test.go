// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
)

func TestTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	bal, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), nil)
	Require(t, err)
	fmt.Println("Owner balance is: ", bal)
	bal2, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if bal2.Cmp(big.NewInt(1e12)) != 0 {
		Fatal(t, "Unexpected recipient balance: ", bal2)
	}
}
