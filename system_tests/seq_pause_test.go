package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

func TestSequencerPause(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	const numUsers = 100

	prechecker, ok := builder.L2.ExecNode.TxPublisher.(*gethexec.TxPreChecker)
	if !ok {
		t.Error("prechecker not found on node")
	}
	sequencer, ok := prechecker.TransactionPublisher.(*gethexec.Sequencer)
	if !ok {
		t.Error("sequencer not found on node")
	}

	var users []string

	for num := 0; num < numUsers; num++ {
		userName := fmt.Sprintf("My_User_%d", num)
		builder.L2Info.GenerateAccount(userName)
		users = append(users, userName)
	}

	for _, userName := range users {
		tx := builder.L2Info.PrepareTx("Owner", userName, builder.L2Info.TransferGas, big.NewInt(1e16), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	sequencer.Pause()

	var txs types.Transactions

	for _, userName := range users {
		tx := builder.L2Info.PrepareTx(userName, "Owner", builder.L2Info.TransferGas, big.NewInt(2), nil)
		txs = append(txs, tx)
	}

	for _, tx := range txs {
		go func(ptx *types.Transaction) {
			err := sequencer.PublishTransaction(ctx, ptx, nil)
			Require(t, err)
		}(tx)
	}

	_, err := builder.L2.EnsureTxSucceededWithTimeout(txs[0], time.Second)
	if err == nil {
		t.Error("tx passed while sequencer paused")
	}

	sequencer.Activate()

	for _, tx := range txs {
		_, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}
}
