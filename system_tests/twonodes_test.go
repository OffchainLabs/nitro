// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

func testTwoNodesSimple(t *testing.T, dasModeStr string) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainConfig, l1NodeConfigA, lifecycleManager, _, dasSignerKey := setupConfigWithDAS(t, ctx, dasModeStr)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig = l1NodeConfigA
	builder.chainConfig = chainConfig
	builder.L2Info = nil
	cleanup := builder.Build(t)
	defer cleanup()

	authorizeDASKeyset(t, ctx, dasSignerKey, builder.L1Info, builder.L1.Client)
	l1NodeConfigBDataAvailability := l1NodeConfigA.DataAvailability
	l1NodeConfigBDataAvailability.RPCAggregator.Enable = false
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{dasConfig: &l1NodeConfigBDataAvailability})
	defer cleanupB()

	builder.L2Info.GenerateAccount("User2")

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err = WaitForTx(ctx, testClientB.Client, tx.Hash(), time.Second*5)
	Require(t, err)

	l2balance, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		Fatal(t, "Unexpected balance:", l2balance)
	}
}

func TestTwoNodesSimple(t *testing.T) {
	testTwoNodesSimple(t, "onchain")
}

func TestTwoNodesSimpleLocalDAS(t *testing.T) {
	testTwoNodesSimple(t, "files")
}
