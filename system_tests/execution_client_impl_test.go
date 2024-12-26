// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
)

func TestExecutionClientImplAsFullExecutionClient(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()
	seqTestClient := builder.L2

	replicaConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: replicaConfig, useExecutionClientImplAsFullExecutionClient: true})
	defer replicaCleanup()

	builder.L2Info.GenerateAccount("User2")
	for i := 0; i < 3; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := seqTestClient.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = seqTestClient.EnsureTxSucceeded(tx)
		Require(t, err)
		_, err = WaitForTx(ctx, replicaTestClient.Client, tx.Hash(), time.Second*15)
		Require(t, err)
	}

	replicaBalance, err := replicaTestClient.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if replicaBalance.Cmp(big.NewInt(3e12)) != 0 {
		t.Fatal("Unexpected balance:", replicaBalance)
	}
}
