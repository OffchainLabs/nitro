// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"
)

func testTwoNodesSimple(t *testing.T, daModeStr string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainConfig, l1NodeConfigA, lifecycleManager, _, anyTrustSignerKey := setupConfigWithAnyTrust(t, ctx, daModeStr)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig = l1NodeConfigA
	builder.chainConfig = chainConfig
	builder.L2Info = nil
	cleanup := builder.Build(t)
	defer cleanup()

	authorizeAnyTrustKeyset(t, ctx, anyTrustSignerKey, builder.L1Info, builder.L1.Client)
	l1NodeConfigBDataAvailability := l1NodeConfigA.DA.AnyTrust
	l1NodeConfigBDataAvailability.RPCAggregator.Enable = false
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{anyTrustConfig: &l1NodeConfigBDataAvailability})
	defer cleanupB()

	builder.L2Info.GenerateAccount("User2")

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	stopL1, l1ErrChan := KeepL1Advancing(builder)
	_, err = WaitForTx(ctx, testClientB.Client, tx.Hash(), time.Second*30)
	Require(t, err)
	close(stopL1)
	Require(t, <-l1ErrChan)

	l2balance, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		Fatal(t, "Unexpected balance:", l2balance)
	}
}

func TestTwoNodesSimple(t *testing.T) {
	testTwoNodesSimple(t, "onchain")
}

func TestTwoNodesSimpleLocalAnyTrust(t *testing.T) {
	testTwoNodesSimple(t, "files")
}
