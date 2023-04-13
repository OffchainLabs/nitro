package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
)

func TestSequencerPause(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info1, nodeA, client := CreateTestL2(t, ctx)
	defer nodeA.StopAndWait()

	const numUsers = 100

	prechecker, ok := nodeA.TxPublisher.(*arbnode.TxPreChecker)
	if !ok {
		t.Error("prechecker not found on node")
	}
	sequencer, ok := prechecker.TransactionPublisher.(*arbnode.Sequencer)
	if !ok {
		t.Error("sequencer not found on node")
	}

	var users []string

	for num := 0; num < numUsers; num++ {
		userName := fmt.Sprintf("My_User_%d", num)
		l2info1.GenerateAccount(userName)
		users = append(users, userName)
	}

	for _, userName := range users {
		tx := l2info1.PrepareTx("Owner", userName, l2info1.TransferGas, big.NewInt(1e16), nil)
		err := client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
	}

	sequencer.Pause()

	var txs types.Transactions

	for _, userName := range users {
		tx := l2info1.PrepareTx(userName, "Owner", l2info1.TransferGas, big.NewInt(2), nil)
		txs = append(txs, tx)
	}

	for _, tx := range txs {
		go func(ptx *types.Transaction) {
			err := sequencer.PublishTransaction(ctx, ptx, nil)
			Require(t, err)
		}(tx)
	}

	_, err := EnsureTxSucceededWithTimeout(ctx, client, txs[0], time.Second)
	if err == nil {
		t.Error("tx passed while sequencer paused")
	}

	sequencer.Activate()

	for _, tx := range txs {
		_, err := EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
	}
}
