package arbtest

import (
	"context"
	"math/big"
	"testing"
)

func TestBasicL3(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanupL1AndL2 := builder.Build(t)
	defer cleanupL1AndL2()

	cleanupL3 := builder.BuildL3OnL2(t)
	defer cleanupL3()

	builder.L3Info.GenerateAccount("User2")
	tx := builder.L3Info.PrepareTx("Owner", "User2", builder.L3Info.TransferGas, big.NewInt(1e12), nil)

	err := builder.L3.Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builder.L3.EnsureTxSucceeded(tx)
	Require(t, err)

	l2balance, err := builder.L3.Client.BalanceAt(ctx, builder.L3Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}
