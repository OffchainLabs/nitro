// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbutil"

	"github.com/ethereum/go-ethereum/core/types"
)

func testTwoNodesLong(t *testing.T, dasModeStr string) {
	t.Parallel()
	largeLoops := 8
	avgL2MsgsPerLoop := 30
	avgDelayedMessagesPerLoop := 10
	avgTotalL1MessagesPerLoop := 12 // including delayed
	avgExtraBlocksPerLoop := 20
	finalPropagateLoops := 15

	fundsPerDelayed := big.NewInt(int64(100000000000000001))
	fundsPerDirect := big.NewInt(int64(200003))
	directTransfers := int64(0)
	delayedTransfers := int64(0)

	delayedTxs := make([]*types.Transaction, 0, largeLoops*avgDelayedMessagesPerLoop*2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainConfig, l1NodeConfigA, lifecycleManager, _, dasSignerKey := setupConfigWithDAS(t, ctx, dasModeStr)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	testNodeA := NewNodeBuilder(ctx).SetNodeConfig(l1NodeConfigA).SetChainConfig(chainConfig).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNodeA.L1Stack)

	authorizeDASKeyset(t, ctx, dasSignerKey, testNodeA.L1Info, testNodeA.L1Client)

	l1NodeConfigBDataAvailability := l1NodeConfigA.DataAvailability
	l1NodeConfigBDataAvailability.RPCAggregator.Enable = false
	l2clientB, nodeB := Create2ndNode(t, ctx, testNodeA.L2Node, testNodeA.L1Stack, testNodeA.L1Info, &testNodeA.L2Info.ArbInitData, &l1NodeConfigBDataAvailability)
	defer nodeB.StopAndWait()

	testNodeA.L2Info.GenerateAccount("DelayedFaucet")
	testNodeA.L2Info.GenerateAccount("DelayedReceiver")
	testNodeA.L2Info.GenerateAccount("DirectReceiver")

	testNodeA.L2Info.GenerateAccount("ErrorTxSender")

	SendWaitTestTransactions(t, ctx, testNodeA.L2Client, []*types.Transaction{
		testNodeA.L2Info.PrepareTx("Faucet", "ErrorTxSender", testNodeA.L2Info.TransferGas, big.NewInt(l2pricing.InitialBaseFeeWei*int64(testNodeA.L2Info.TransferGas)), nil),
	})

	delayedMsgsToSendMax := big.NewInt(int64(largeLoops * avgDelayedMessagesPerLoop * 10))
	delayedFaucetNeeds := new(big.Int).Mul(new(big.Int).Add(fundsPerDelayed, new(big.Int).SetUint64(l2pricing.InitialBaseFeeWei*100000)), delayedMsgsToSendMax)
	SendWaitTestTransactions(t, ctx, testNodeA.L2Client, []*types.Transaction{
		testNodeA.L2Info.PrepareTx("Faucet", "DelayedFaucet", testNodeA.L2Info.TransferGas, delayedFaucetNeeds, nil),
	})
	delayedFaucetBalance, err := testNodeA.L2Client.BalanceAt(ctx, testNodeA.L2Info.GetAddress("DelayedFaucet"), nil)
	Require(t, err)

	if delayedFaucetBalance.Cmp(delayedFaucetNeeds) != 0 {
		t.Fatalf("Unexpected balance, has %v, expects %v", delayedFaucetBalance, delayedFaucetNeeds)
	}
	t.Logf("DelayedFaucet has %v, per delayd: %v, baseprice: %v", delayedFaucetBalance, fundsPerDelayed, l2pricing.InitialBaseFeeWei)

	if avgTotalL1MessagesPerLoop < avgDelayedMessagesPerLoop {
		Fatal(t, "bad params, avgTotalL1MessagesPerLoop should include avgDelayedMessagesPerLoop")
	}
	for i := 0; i < largeLoops; i++ {
		l1TxsThisTime := rand.Int() % (avgTotalL1MessagesPerLoop * 2)
		l1Txs := make([]*types.Transaction, 0, l1TxsThisTime)
		for len(l1Txs) < l1TxsThisTime {
			randNum := rand.Int() % avgTotalL1MessagesPerLoop
			var l1tx *types.Transaction
			if randNum < avgDelayedMessagesPerLoop {
				delayedTx := testNodeA.L2Info.PrepareTx("DelayedFaucet", "DelayedReceiver", 30001, fundsPerDelayed, nil)
				l1tx = WrapL2ForDelayed(t, delayedTx, testNodeA.L1Info, "User", 100000)
				delayedTxs = append(delayedTxs, delayedTx)
				delayedTransfers++
			} else {
				l1tx = testNodeA.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil)
			}
			l1Txs = append(l1Txs, l1tx)
		}
		// adding multiple messages in the same AddLocal to get them in the same L1 block
		errs := testNodeA.L1Backend.TxPool().AddLocals(l1Txs)
		for _, err := range errs {
			if err != nil {
				Fatal(t, err)
			}
		}
		l2TxsThisTime := rand.Int() % (avgL2MsgsPerLoop * 2)
		l2Txs := make([]*types.Transaction, 0, l2TxsThisTime)
		for len(l2Txs) < l2TxsThisTime {
			l2Txs = append(l2Txs, testNodeA.L2Info.PrepareTx("Faucet", "DirectReceiver", testNodeA.L2Info.TransferGas, fundsPerDirect, nil))
		}
		SendWaitTestTransactions(t, ctx, testNodeA.L2Client, l2Txs)
		directTransfers += int64(l2TxsThisTime)
		if len(l1Txs) > 0 {
			_, err := EnsureTxSucceeded(ctx, testNodeA.L1Client, l1Txs[len(l1Txs)-1])
			if err != nil {
				Fatal(t, err)
			}
		}
		// create bad tx on delayed inbox
		testNodeA.L2Info.GetInfoWithPrivKey("ErrorTxSender").Nonce = 10
		SendWaitTestTransactions(t, ctx, testNodeA.L1Client, []*types.Transaction{
			WrapL2ForDelayed(t, testNodeA.L2Info.PrepareTx("ErrorTxSender", "DelayedReceiver", 30002, delayedFaucetNeeds, nil), testNodeA.L1Info, "User", 100000),
		})

		extrBlocksThisTime := rand.Int() % (avgExtraBlocksPerLoop * 2)
		for i := 0; i < extrBlocksThisTime; i++ {
			SendWaitTestTransactions(t, ctx, testNodeA.L1Client, []*types.Transaction{
				testNodeA.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
			})
		}
	}

	t.Log("Done sending", delayedTransfers, "delayed transfers", directTransfers, "direct transfers")
	if (delayedTransfers + directTransfers) == 0 {
		Fatal(t, "No transfers sent!")
	}

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < finalPropagateLoops; i++ {
		var tx *types.Transaction
		for j := 0; j < 30; j++ {
			tx = testNodeA.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil)
			err := testNodeA.L1Client.SendTransaction(ctx, tx)
			if err != nil {
				Fatal(t, err)
			}
			_, err = EnsureTxSucceeded(ctx, testNodeA.L1Client, tx)
			if err != nil {
				Fatal(t, err)
			}
		}
	}

	_, err = EnsureTxSucceededWithTimeout(ctx, testNodeA.L2Client, delayedTxs[len(delayedTxs)-1], time.Second*10)
	Require(t, err, "Failed waiting for Tx on main node")
	_, err = EnsureTxSucceededWithTimeout(ctx, l2clientB, delayedTxs[len(delayedTxs)-1], time.Second*10)
	Require(t, err, "Failed waiting for Tx on secondary node")
	delayedBalance, err := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("DelayedReceiver"), nil)
	Require(t, err)
	directBalance, err := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("DirectReceiver"), nil)
	Require(t, err)
	delayedExpectd := new(big.Int).Mul(fundsPerDelayed, big.NewInt(delayedTransfers))
	directExpectd := new(big.Int).Mul(fundsPerDirect, big.NewInt(directTransfers))
	if (delayedBalance.Cmp(delayedExpectd) != 0) || (directBalance.Cmp(directExpectd) != 0) {
		t.Error("delayed balance", delayedBalance, "expected", delayedExpectd, "transfers", delayedTransfers)
		t.Error("direct balance", directBalance, "expected", directExpectd, "transfers", directTransfers)
		ownerBalance, _ := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("Owner"), nil)
		delayedFaucetBalance, _ := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("DelayedFaucet"), nil)
		t.Error("owner balance", ownerBalance, "delayed faucet", delayedFaucetBalance)
		Fatal(t, "Unexpected balance")
	}

	testNodeA.L2Node.StopAndWait()

	if nodeB.BlockValidator != nil {
		lastBlockHeader, err := l2clientB.HeaderByNumber(ctx, nil)
		Require(t, err)
		timeout := getDeadlineTimeout(t, time.Minute*30)
		// messageindex is same as block number here
		if !nodeB.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(lastBlockHeader.Number.Uint64()), timeout) {
			Fatal(t, "did not validate all blocks")
		}
	}
}

func TestTwoNodesLong(t *testing.T) {
	testTwoNodesLong(t, "onchain")
}

func TestTwoNodesLongLocalDAS(t *testing.T) {
	testTwoNodesLong(t, "files")
}
