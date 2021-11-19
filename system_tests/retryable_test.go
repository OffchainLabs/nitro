//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
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

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

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
