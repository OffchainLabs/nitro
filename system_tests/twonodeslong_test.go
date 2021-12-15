//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbnode"
)

func TestTwoNodesLong(t *testing.T) {
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
	l2info, node1, l1info, l1backend, l1stack := CreateTestNodeOnL1(t, ctx, true)
	defer l1stack.Close()

	l2clientB, nodeB := Create2ndNode(t, ctx, node1, l1stack)

	l2info.GenerateAccount("DelayedFaucet")
	l2info.GenerateAccount("DelayedReceiver")
	l2info.GenerateAccount("DirectReceiver")

	l2info.GenerateAccount("ErrorTxSender")

	SendWaitTestTransactions(t, ctx, l2info.Client, []*types.Transaction{
		l2info.PrepareTx("Faucet", "ErrorTxSender", 30000, big.NewInt(params.InitialBaseFee*30000), nil),
	})

	delayedMsgsToSendMax := big.NewInt(int64(largeLoops * avgDelayedMessagesPerLoop * 10))
	delayedFaucetNeeds := new(big.Int).Mul(new(big.Int).Add(fundsPerDelayed, new(big.Int).SetUint64(params.InitialBaseFee*100000)), delayedMsgsToSendMax)
	SendWaitTestTransactions(t, ctx, l2info.Client, []*types.Transaction{
		l2info.PrepareTx("Faucet", "DelayedFaucet", 30000, delayedFaucetNeeds, nil),
	})
	delayedFaucetBalance, err := l2info.Client.BalanceAt(ctx, l2info.GetAddress("DelayedFaucet"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if delayedFaucetBalance.Cmp(delayedFaucetNeeds) != 0 {
		t.Fatalf("Unexpected balance, has %v, expects %v", delayedFaucetBalance, delayedFaucetNeeds)
	}
	t.Logf("DelayedFaucet has %v, per delayd: %v, baseprice: %v", delayedFaucetBalance, fundsPerDelayed, params.InitialBaseFee)

	if avgTotalL1MessagesPerLoop < avgDelayedMessagesPerLoop {
		t.Fatal("bad params, avgTotalL1MessagesPerLoop should include avgDelayedMessagesPerLoop")
	}
	for i := 0; i < largeLoops; i++ {
		l1TxsThisTime := rand.Int() % (avgTotalL1MessagesPerLoop * 2)
		l1Txs := make([]*types.Transaction, 0, l1TxsThisTime)
		for len(l1Txs) < l1TxsThisTime {
			randNum := rand.Int() % avgTotalL1MessagesPerLoop
			var l1tx *types.Transaction
			if randNum < avgDelayedMessagesPerLoop {
				delayedTx := l2info.PrepareTx("DelayedFaucet", "DelayedReceiver", 30001, fundsPerDelayed, nil)
				l1tx = WrapL2ForDelayed(t, delayedTx, l1info, "User", 100000)
				delayedTxs = append(delayedTxs, delayedTx)
				delayedTransfers++
			} else {
				l1tx = l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil)
			}
			l1Txs = append(l1Txs, l1tx)
		}
		// adding multiple messages in the same AddLocal to get them in the same L1 block
		errs := l1backend.TxPool().AddLocals(l1Txs)
		for _, err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		l2TxsThisTime := rand.Int() % (avgL2MsgsPerLoop * 2)
		l2Txs := make([]*types.Transaction, 0, l2TxsThisTime)
		for len(l2Txs) < l2TxsThisTime {
			l2Txs = append(l2Txs, l2info.PrepareTx("Faucet", "DirectReceiver", 30000, fundsPerDirect, nil))
		}
		SendWaitTestTransactions(t, ctx, l2info.Client, l2Txs)
		directTransfers += int64(l2TxsThisTime)
		if len(l1Txs) > 0 {
			_, err := arbnode.EnsureTxSucceeded(ctx, l1info.Client, l1Txs[len(l1Txs)-1])
			if err != nil {
				t.Fatal(err)
			}
		}
		// create bad tx on delayed inbox
		l2info.GetInfoWithPrivKey("ErrorTxSender").Nonce = 10
		SendWaitTestTransactions(t, ctx, l1info.Client, []*types.Transaction{
			WrapL2ForDelayed(t, l2info.PrepareTx("ErrorTxSender", "DelayedReceiver", 30002, delayedFaucetNeeds, nil), l1info, "User", 100000),
		})

		extrBlocksThisTime := rand.Int() % (avgExtraBlocksPerLoop * 2)
		for i := 0; i < extrBlocksThisTime; i++ {
			SendWaitTestTransactions(t, ctx, l1info.Client, []*types.Transaction{
				l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil),
			})
		}
	}

	t.Log("Done sending", delayedTransfers, "delayed transfers", directTransfers, "direct transfers")
	if (delayedTransfers + directTransfers) == 0 {
		t.Fatal("No transfers sent!")
	}

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < finalPropagateLoops; i++ {
		var tx *types.Transaction
		for j := 0; j < 30; j++ {
			tx = l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil)
			err := l1info.Client.SendTransaction(ctx, tx)
			if err != nil {
				t.Fatal(err)
			}
		}
		_, err := arbnode.EnsureTxSucceeded(ctx, l1info.Client, tx)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = arbnode.EnsureTxSucceededWithTimeout(ctx, l2info.Client, delayedTxs[len(delayedTxs)-1], time.Second*10)
	if err != nil {
		t.Fatal("Failed waiting for Tx on main node", "err", err)
	}
	_, err = arbnode.EnsureTxSucceededWithTimeout(ctx, l2clientB, delayedTxs[len(delayedTxs)-1], time.Second*10)
	if err != nil {
		t.Fatal("Failed waiting for Tx on secondary node", "err", err)
	}
	delayedBalance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("DelayedReceiver"), nil)
	if err != nil {
		t.Fatal(err)
	}
	directBalance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("DirectReceiver"), nil)
	if err != nil {
		t.Fatal(err)
	}
	delayedExpectd := new(big.Int).Mul(fundsPerDelayed, big.NewInt(delayedTransfers))
	directExpectd := new(big.Int).Mul(fundsPerDirect, big.NewInt(directTransfers))
	if (delayedBalance.Cmp(delayedExpectd) != 0) || (directBalance.Cmp(directExpectd) != 0) {
		t.Error("delayed balance", delayedBalance, "expected", delayedExpectd, "transfers", delayedTransfers)
		t.Error("direct balance", directBalance, "expected", directExpectd, "transfers", directTransfers)
		ownerBalance, _ := l2clientB.BalanceAt(ctx, l2info.GetAddress("Owner"), nil)
		delayedFaucetBalance, _ := l2clientB.BalanceAt(ctx, l2info.GetAddress("DelayedFaucet"), nil)
		t.Error("owner balance", ownerBalance, "delayed faucet", delayedFaucetBalance)
		t.Fatal("Unexpected balance")
	}
	if nodeB.BlockValidator != nil {
		lastBlockHeader, err := l2clientB.HeaderByNumber(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		testDeadLine, deadlineExist := t.Deadline()
		var timeout time.Duration
		if deadlineExist {
			timeout = time.Until(testDeadLine) - (time.Second * 10)
		} else {
			timeout = time.Hour * 12
		}
		if !nodeB.BlockValidator.WaitForBlock(lastBlockHeader.Number.Uint64(), timeout) {
			t.Fatal("did not validate all blocks")
		}
	}
}
