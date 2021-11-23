//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
	"math/big"
	"testing"
	"time"
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
	if err != nil {
		t.Fatal(err)
	}
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
		big.NewInt(params.InitialBaseFee * 2),
		[]byte{},
	)
	if err != nil {
		t.Fatal(err)
	}
	l1receipt, err := arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	if err != nil {
		t.Fatal(err)
	}
	if l1receipt.Status != 1 {
		t.Fatal("l1receipt indicated failure")
	}

	inboxFilterer, err := bridgegen.NewInboxFilterer(l1info.GetAddress("Inbox"), l1client)
	if err != nil {
		t.Fatal(err)
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
		t.Fatal()
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := arbnode.WaitForTx(ctx, l2client, *l2TxId, time.Second*5)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != 1 {
		t.Fatal()
	}

	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balance.Cmp(big.NewInt(1e6)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
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
	if err != nil {
		t.Fatal(err)
	}
	usertxopts := l1info.GetDefaultTransactOpts("faucet")
	usertxopts.Value = new(big.Int).Mul(big.NewInt(1e12), big.NewInt(1e12))

	l1tx, err := delayedInboxContract.CreateRetryableTicket(
		&usertxopts,
		user2Address,
		big.NewInt(1e6),
		big.NewInt(1e6),
		user2Address,
		user2Address,
		big.NewInt(1),   // send inadequate L2 gas
		big.NewInt(params.InitialBaseFee * 2),
		[]byte{},
	)
	if err != nil {
		t.Fatal(err)
	}
	l1receipt, err := arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	if err != nil {
		t.Fatal(err)
	}
	if l1receipt.Status != 1 {
		t.Fatal("l1receipt indicated failure")
	}

	inboxFilterer, err := bridgegen.NewInboxFilterer(l1info.GetAddress("Inbox"), l1client)
	if err != nil {
		t.Fatal(err)
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
		t.Fatal()
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := arbnode.WaitForTx(ctx, l2client, *l2TxId, time.Second*5)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != 0 {
		t.Fatal()
	}

	// send tx to redeem the retryable
	arbRetryableTxAbi, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}
	arbRetryableAddress := common.BigToAddress(big.NewInt(0x6e))
	txData := &types.DynamicFeeTx{
		To:        &arbRetryableAddress,
		Gas:       100001,
		GasFeeCap: big.NewInt(params.InitialBaseFee * 2),
		Value:     big.NewInt(0),
		Nonce:     0,
		Data:      append(arbRetryableTxAbi.Methods["redeem"].ID, make([]byte, 32)...),
	}
	tx := l2info.SignTxAs("Owner", txData)
	txbytes, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	usertxopts = l1info.GetDefaultTransactOpts("faucet")
	fmt.Println("============== submitting redeem message to delayed inbox")
	l1tx, err = delayedInboxContract.SendL2Message(&usertxopts, txwrapped)
	if err != nil {
		t.Fatal(err)
	}
	_, err = arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	if err != nil {
		t.Fatal(err)
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)
	receipt, err = arbnode.WaitForTx(ctx, l2client, tx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != 1 {
		t.Fatal()
	}

	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balance.Cmp(big.NewInt(1e6)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
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

