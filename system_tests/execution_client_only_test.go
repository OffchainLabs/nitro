// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
)

func testExecutionClientOnly(t *testing.T, executionClientMode ExecutionClientMode) {
	ctx := t.Context()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()
	seqTestClient := builder.L2

	replicaConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	replicaParams := &SecondNodeParams{
		nodeConfig:             replicaConfig,
		useExecutionClientOnly: true,
		executionClientMode:    executionClientMode,
	}

	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, replicaParams)
	defer replicaCleanup()
	replicaClient := replicaTestClient.Client

	builder.L2Info.GenerateAccount("User2")
	for range 3 {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := seqTestClient.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = seqTestClient.EnsureTxSucceeded(tx)
		Require(t, err)

		_, err = WaitForTx(ctx, replicaClient, tx.Hash(), time.Second*15)
		Require(t, err)
	}

	expectedBalance := big.NewInt(3e12)
	replicaBalance, err := replicaClient.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if replicaBalance.Cmp(expectedBalance) != 0 {
		t.Fatalf("Final balance mismatch. Got: %s, expected: %s", replicaBalance, expectedBalance)
	}
}

func TestExecutionClientOnlyInternal(t *testing.T) {
	testExecutionClientOnly(t, ExecutionClientModeInternal)
}

func TestExecutionClientOnlyExternal(t *testing.T) {
	testExecutionClientOnly(t, ExecutionClientModeExternal)
}

func TestExecutionClientOnlyComparison(t *testing.T) {
	testExecutionClientOnly(t, ExecutionClientModeComparison)
}
