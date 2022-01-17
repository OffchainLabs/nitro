//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
)

func TestSubmitRetryableImmediateSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, _, l1info, _, stack := CreateTestNodeOnL1(t, ctx, true)
	defer stack.Close()

	l2client := l2info.Client
	l1client := l1info.Client
	l2info.GenerateAccount("User2")
	user2Address := l2info.GetAddress("User2")

	delayedInboxContract, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)

	usertxopts := l1info.GetDefaultTransactOpts("faucet")
	usertxopts.Value = new(big.Int).Mul(big.NewInt(1e12), big.NewInt(1e12))

	l1tx, err := delayedInboxContract.CreateRetryableTicket(
		&usertxopts,
		user2Address,
		big.NewInt(1e6),
		big.NewInt(1e6),
		user2Address,
		user2Address,
		big.NewInt(50001),
		big.NewInt(params.InitialBaseFee*2),
		[]byte{},
	)
	Require(t, err)

	l1receipt, err := arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, "l1receipt indicated failure")
	}

	inboxFilterer, err := bridgegen.NewInboxFilterer(l1info.GetAddress("Inbox"), l1client)
	if err != nil {
		Fail(t, err)
	}
	var l2TxId *common.Hash
	for _, log := range l1receipt.Logs {
		msg, _ := inboxFilterer.ParseInboxMessageDelivered(*log)
		if msg != nil {
			id := common.BigToHash(msg.MessageNum)
			l2TxId = &id
		}
	}
	if l2TxId == nil {
		Fail(t)
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := arbnode.WaitForTx(ctx, l2client, *l2TxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t)
	}

	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Cmp(big.NewInt(1e6)) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}
}

func TestSubmitRetryableFailThenRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, _, l1info, _, stack := CreateTestNodeOnL1(t, ctx, true)
	defer stack.Close()

	l2client := l2info.Client
	l1client := l1info.Client
	l2info.GenerateAccount("User2")
	user2Address := l2info.GetAddress("User2")

	delayedInboxContract, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)

	usertxopts := l1info.GetDefaultTransactOpts("faucet")
	usertxopts.Value = new(big.Int).Mul(big.NewInt(1e12), big.NewInt(1e12))

	l1tx, err := delayedInboxContract.CreateRetryableTicket(
		&usertxopts,
		user2Address,
		big.NewInt(1e6),
		big.NewInt(1e6),
		user2Address,
		user2Address,
		big.NewInt(1), // send inadequate L2 gas
		big.NewInt(params.InitialBaseFee*2),
		[]byte{},
	)
	Require(t, err)

	l1receipt, err := arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, "l1receipt indicated failure")
	}

	inboxFilterer, err := bridgegen.NewInboxFilterer(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)

	var l2TxId *common.Hash
	for _, log := range l1receipt.Logs {
		msg, _ := inboxFilterer.ParseInboxMessageDelivered(*log)
		if msg != nil {
			id := common.BigToHash(msg.MessageNum)
			l2TxId = &id
		}
	}
	if l2TxId == nil {
		Fail(t)
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := arbnode.WaitForTx(ctx, l2client, *l2TxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t)
	}
	ticketId := receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]

	// get receipt for the auto-redeem, make sure it failed
	receipt, err = arbnode.WaitForTx(ctx, l2client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fail(t)
	}

	// send tx to redeem the retryable
	arbRetryableTxAbi, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	Require(t, err)

	arbRetryableAddress := common.BigToAddress(big.NewInt(0x6e))
	txData := &types.DynamicFeeTx{
		To:        &arbRetryableAddress,
		Gas:       10000001,
		GasFeeCap: big.NewInt(params.InitialBaseFee * 2),
		Value:     big.NewInt(0),
		Nonce:     0,
		Data:      append(arbRetryableTxAbi.Methods["redeem"].ID, ticketId.Bytes()...),
	}
	tx := l2info.SignTxAs("Owner", txData)
	txbytes, err := tx.MarshalBinary()
	Require(t, err)

	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	usertxopts = l1info.GetDefaultTransactOpts("faucet")
	l1tx, err = delayedInboxContract.SendL2Message(&usertxopts, txwrapped)
	Require(t, err)

	_, err = arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)

	// wait for redeem transaction to complete successfully
	waitForL1DelayBlocks(t, ctx, l1client, l1info)
	receipt, err = arbnode.WaitForTx(ctx, l2client, tx.Hash(), time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, *receipt)
	}
	retryTxId := receipt.Logs[0].Topics[2]

	// verify that balance transfer happened, so we know the retry succeeded
	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Cmp(big.NewInt(1e6)) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}

	// check the receipt for the retry
	receipt, err = arbnode.WaitForTx(ctx, l2client, retryTxId, time.Second*1)
	Require(t, err)
	if receipt.Status != 1 {
		Fail(t)
	}
}

func waitForL1DelayBlocks(t *testing.T, ctx context.Context, l1client *ethclient.Client, l1info *BlockchainTestInfo) {
	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}
}
