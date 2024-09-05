package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
)

func TestSimpleL3(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanupL1AndL2 := builder.Build(t)
	defer cleanupL1AndL2()

	cleanupL3FirstNode := builder.BuildL3OnL2(t)
	defer cleanupL3FirstNode()
	firstNodeTestClient := builder.L3

	secondNodeNodeConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	secondNodeTestClient, cleanupL3SecondNode := builder.Build2ndNodeOnL3(t, &SecondNodeParams{nodeConfig: secondNodeNodeConfig})
	defer cleanupL3SecondNode()

	accountName := "User2"
	builder.L3Info.GenerateAccount(accountName)
	tx := builder.L3Info.PrepareTx("Owner", accountName, builder.L3Info.TransferGas, big.NewInt(1e12), nil)

	err := firstNodeTestClient.Client.SendTransaction(ctx, tx)
	Require(t, err)

	// Checks that first node has the correct balance
	_, err = firstNodeTestClient.EnsureTxSucceeded(tx)
	Require(t, err)
	l2balance, err := firstNodeTestClient.Client.BalanceAt(ctx, builder.L3Info.GetAddress(accountName), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	// Checks that second node has the correct balance
	_, err = WaitForTx(ctx, secondNodeTestClient.Client, tx.Hash(), time.Second*15)
	Require(t, err)
	l2balance, err = secondNodeTestClient.Client.BalanceAt(ctx, builder.L3Info.GetAddress(accountName), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}
