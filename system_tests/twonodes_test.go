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

	testNodeA := NewNodeBuilder(ctx).SetNodeConfig(l1NodeConfigA).SetChainConfig(chainConfig).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNodeA.L1Stack)
	defer testNodeA.L2Node.StopAndWait()

	authorizeDASKeyset(t, ctx, dasSignerKey, testNodeA.L1Info, testNodeA.L1Client)
	l1NodeConfigBDataAvailability := l1NodeConfigA.DataAvailability
	l1NodeConfigBDataAvailability.RPCAggregator.Enable = false
	l2clientB, nodeB := Create2ndNode(t, ctx, testNodeA.L2Node, testNodeA.L1Stack, testNodeA.L1Info, &testNodeA.L2Info.ArbInitData, &l1NodeConfigBDataAvailability)
	defer nodeB.StopAndWait()

	testNodeA.L2Info.GenerateAccount("User2")

	tx := testNodeA.L2Info.PrepareTx("Owner", "User2", testNodeA.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := testNodeA.L2Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, testNodeA.L2Client, tx)
	Require(t, err)

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, testNodeA.L1Client, []*types.Transaction{
			testNodeA.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err = WaitForTx(ctx, l2clientB, tx.Hash(), time.Second*5)
	Require(t, err)

	l2balance, err := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("User2"), nil)
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
