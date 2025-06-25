// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
)

func TestExecutionClientOnly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()
	seqTestClient := builder.L2

	replicaExecutionClientOnlyConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	replicaExecutionClientOnlyTestClient, replicaExecutionClientOnlyCleanup := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: replicaExecutionClientOnlyConfig, useExecutionClientOnly: true})
	defer replicaExecutionClientOnlyCleanup()

	builder.L2Info.GenerateAccount("User2")
	for i := 0; i < 3; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := seqTestClient.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = seqTestClient.EnsureTxSucceeded(tx)
		Require(t, err)
		_, err = WaitForTx(ctx, replicaExecutionClientOnlyTestClient.Client, tx.Hash(), time.Second*15)
		Require(t, err)
	}

	replicaBalance, err := replicaExecutionClientOnlyTestClient.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if replicaBalance.Cmp(big.NewInt(3e12)) != 0 {
		t.Fatal("Unexpected balance:", replicaBalance)
	}
}
