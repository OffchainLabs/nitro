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

	"github.com/ethereum/go-ethereum/core/txpool"
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

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig = l1NodeConfigA
	builder.chainConfig = chainConfig
	builder.L2Info = nil
	builder.Build(t)
	defer requireClose(t, builder.L1.Stack)

	authorizeDASKeyset(t, ctx, dasSignerKey, builder.L1Info, builder.L1.Client)

	l1NodeConfigBDataAvailability := l1NodeConfigA.DataAvailability
	l1NodeConfigBDataAvailability.RPCAggregator.Enable = false
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{dasConfig: &l1NodeConfigBDataAvailability})
	defer cleanupB()

	builder.L2Info.GenerateAccount("DelayedFaucet")
	builder.L2Info.GenerateAccount("DelayedReceiver")
	builder.L2Info.GenerateAccount("DirectReceiver")

	builder.L2Info.GenerateAccount("ErrorTxSender")

	builder.L2.SendWaitTestTransactions(t, []*types.Transaction{
		builder.L2Info.PrepareTx("Faucet", "ErrorTxSender", builder.L2Info.TransferGas, big.NewInt(l2pricing.InitialBaseFeeWei*int64(builder.L2Info.TransferGas)), nil),
	})

	delayedMsgsToSendMax := big.NewInt(int64(largeLoops * avgDelayedMessagesPerLoop * 10))
	delayedFaucetNeeds := new(big.Int).Mul(new(big.Int).Add(fundsPerDelayed, new(big.Int).SetUint64(l2pricing.InitialBaseFeeWei*100000)), delayedMsgsToSendMax)
	builder.L2.SendWaitTestTransactions(t, []*types.Transaction{
		builder.L2Info.PrepareTx("Faucet", "DelayedFaucet", builder.L2Info.TransferGas, delayedFaucetNeeds, nil),
	})
	delayedFaucetBalance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("DelayedFaucet"), nil)
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
		wrappedL1Txs := make([]*txpool.Transaction, 0, l1TxsThisTime)
		for len(wrappedL1Txs) < l1TxsThisTime {
			randNum := rand.Int() % avgTotalL1MessagesPerLoop
			var l1tx *types.Transaction
			if randNum < avgDelayedMessagesPerLoop {
				delayedTx := builder.L2Info.PrepareTx("DelayedFaucet", "DelayedReceiver", 30001, fundsPerDelayed, nil)
				l1tx = WrapL2ForDelayed(t, delayedTx, builder.L1Info, "User", 100000)
				delayedTxs = append(delayedTxs, delayedTx)
				delayedTransfers++
			} else {
				l1tx = builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil)
			}
			wrappedL1Txs = append(wrappedL1Txs, &txpool.Transaction{Tx: l1tx})
		}

		// adding multiple messages in the same Add with local=true to get them in the same L1 block
		errs := builder.L1.L1Backend.TxPool().Add(wrappedL1Txs, true, false)
		for _, err := range errs {
			if err != nil {
				Fatal(t, err)
			}
		}
		l2TxsThisTime := rand.Int() % (avgL2MsgsPerLoop * 2)
		l2Txs := make([]*types.Transaction, 0, l2TxsThisTime)
		for len(l2Txs) < l2TxsThisTime {
			l2Txs = append(l2Txs, builder.L2Info.PrepareTx("Faucet", "DirectReceiver", builder.L2Info.TransferGas, fundsPerDirect, nil))
		}
		builder.L2.SendWaitTestTransactions(t, l2Txs)
		directTransfers += int64(l2TxsThisTime)
		if len(wrappedL1Txs) > 0 {
			_, err := builder.L1.EnsureTxSucceeded(wrappedL1Txs[len(wrappedL1Txs)-1].Tx)
			if err != nil {
				Fatal(t, err)
			}
		}
		// create bad tx on delayed inbox
		builder.L2Info.GetInfoWithPrivKey("ErrorTxSender").Nonce = 10
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			WrapL2ForDelayed(t, builder.L2Info.PrepareTx("ErrorTxSender", "DelayedReceiver", 30002, delayedFaucetNeeds, nil), builder.L1Info, "User", 100000),
		})

		extrBlocksThisTime := rand.Int() % (avgExtraBlocksPerLoop * 2)
		for i := 0; i < extrBlocksThisTime; i++ {
			builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
				builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
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
			tx = builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil)
			err := builder.L1.Client.SendTransaction(ctx, tx)
			if err != nil {
				Fatal(t, err)
			}
			_, err = builder.L1.EnsureTxSucceeded(tx)
			if err != nil {
				Fatal(t, err)
			}
		}
	}

	_, err = builder.L2.EnsureTxSucceededWithTimeout(delayedTxs[len(delayedTxs)-1], time.Second*10)
	Require(t, err, "Failed waiting for Tx on main node")

	_, err = testClientB.EnsureTxSucceededWithTimeout(delayedTxs[len(delayedTxs)-1], time.Second*30)
	Require(t, err, "Failed waiting for Tx on secondary node")
	delayedBalance, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("DelayedReceiver"), nil)
	Require(t, err)
	directBalance, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("DirectReceiver"), nil)
	Require(t, err)
	delayedExpectd := new(big.Int).Mul(fundsPerDelayed, big.NewInt(delayedTransfers))
	directExpectd := new(big.Int).Mul(fundsPerDirect, big.NewInt(directTransfers))
	if (delayedBalance.Cmp(delayedExpectd) != 0) || (directBalance.Cmp(directExpectd) != 0) {
		t.Error("delayed balance", delayedBalance, "expected", delayedExpectd, "transfers", delayedTransfers)
		t.Error("direct balance", directBalance, "expected", directExpectd, "transfers", directTransfers)
		ownerBalance, _ := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), nil)
		delayedFaucetBalance, _ := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("DelayedFaucet"), nil)
		t.Error("owner balance", ownerBalance, "delayed faucet", delayedFaucetBalance)
		Fatal(t, "Unexpected balance")
	}

	builder.L2.ConsensusNode.StopAndWait()

	if testClientB.ConsensusNode.BlockValidator != nil {
		lastBlockHeader, err := testClientB.Client.HeaderByNumber(ctx, nil)
		Require(t, err)
		timeout := getDeadlineTimeout(t, time.Minute*30)
		// messageindex is same as block number here
		if !testClientB.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(lastBlockHeader.Number.Uint64()), timeout) {
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
