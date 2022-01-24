//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
)

var inboxABI abi.ABI

func init() {
	var err error
	inboxABI, err = abi.JSON(strings.NewReader(bridgegen.InboxABI))
	if err != nil {
		panic(err)
	}
}

func WrapL2ForDelayed(t *testing.T, l2Tx *types.Transaction, l1info *BlockchainTestInfo, delayedSender string, gas uint64) *types.Transaction {
	txbytes, err := l2Tx.MarshalBinary()
	Require(t, err)
	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	delayedInboxTxData, err := inboxABI.Pack("sendL2Message", txwrapped)
	Require(t, err)
	return l1info.PrepareTx(delayedSender, "Inbox", gas, big.NewInt(0), delayedInboxTxData)
}

func TestDelayInboxSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, _, l2client, l1info, _, l1client, stack := CreateTestNodeOnL1(t, ctx, true)
	defer stack.Close()

	l2info.GenerateAccount("User2")

	delayedTx := l2info.PrepareTx("Owner", "User2", 50001, big.NewInt(1e6), nil)

	delayedInboxContract, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)
	usertxopts := l1info.GetDefaultTransactOpts("User")
	txbytes, err := delayedTx.MarshalBinary()
	Require(t, err)
	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	l1tx, err := delayedInboxContract.SendL2Message(&usertxopts, txwrapped)
	Require(t, err)
	_, err = arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}
	_, err = arbnode.WaitForTx(ctx, l2client, delayedTx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e6)) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}
}
