package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/execution"
)

func TestSequencerPause(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testNode := NewNodeBuilder(ctx).SetNodeConfig(arbnode.ConfigDefaultL2Test()).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()

	const numUsers = 100

	prechecker, ok := testNode.L2Node.Execution.TxPublisher.(*execution.TxPreChecker)
	if !ok {
		t.Error("prechecker not found on node")
	}
	sequencer, ok := prechecker.TransactionPublisher.(*execution.Sequencer)
	if !ok {
		t.Error("sequencer not found on node")
	}

	var users []string

	for num := 0; num < numUsers; num++ {
		userName := fmt.Sprintf("My_User_%d", num)
		testNode.L2Info.GenerateAccount(userName)
		users = append(users, userName)
	}

	for _, userName := range users {
		tx := testNode.L2Info.PrepareTx("Owner", userName, testNode.L2Info.TransferGas, big.NewInt(1e16), nil)
		err := testNode.L2Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, testNode.L2Client, tx)
		Require(t, err)
	}

	sequencer.Pause()

	var txs types.Transactions

	for _, userName := range users {
		tx := testNode.L2Info.PrepareTx(userName, "Owner", testNode.L2Info.TransferGas, big.NewInt(2), nil)
		txs = append(txs, tx)
	}

	for _, tx := range txs {
		go func(ptx *types.Transaction) {
			err := sequencer.PublishTransaction(ctx, ptx, nil)
			Require(t, err)
		}(tx)
	}

	_, err := EnsureTxSucceededWithTimeout(ctx, testNode.L2Client, txs[0], time.Second)
	if err == nil {
		t.Error("tx passed while sequencer paused")
	}

	sequencer.Activate()

	for _, tx := range txs {
		_, err := EnsureTxSucceeded(ctx, testNode.L2Client, tx)
		Require(t, err)
	}
}
