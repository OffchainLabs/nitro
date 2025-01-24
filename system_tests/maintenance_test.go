// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
)

func TestMaintenance(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	numberOfTransfers := 10
	for i := 2; i < 3+numberOfTransfers; i++ {
		account := fmt.Sprintf("User%d", i)
		builder.L2Info.GenerateAccount(account)

		tx := builder.L2Info.PrepareTx("Owner", account, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	err := builder.L2.ExecNode.Maintenance()
	Require(t, err)

	for i := 2; i < 3+numberOfTransfers; i++ {
		account := fmt.Sprintf("User%d", i)
		balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress(account), nil)
		Require(t, err)
		if balance.Cmp(big.NewInt(int64(1e12))) != 0 {
			t.Fatal("Unexpected balance:", balance, "for account:", account)
		}
	}
}
