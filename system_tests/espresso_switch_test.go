package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
)

func TestEspressoSwitch(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, l2Node, l2Info, cleanup := runNodes(ctx, t)
	defer cleanup()
	node := builder.L2
	// Inacitivating the delayed sequencer for convenient sake
	node.ConsensusNode.DelayedSequencer.StopAndWait()
	l2Node.ConsensusNode.DelayedSequencer.StopAndWait()

	seq := l2Node.ExecNode.TxPublisher
	err := seq.SetMode(ctx, false)
	Require(t, err)

	currMsg := arbutil.MessageIndex(0)
	// Wait for the switch to be totally finished
	err = waitForWith(t, ctx, 2*time.Minute, 15*time.Second, func() bool {
		msg, err := node.ConsensusNode.TxStreamer.GetMessageCount()
		if err != nil {
			return false
		}
		if currMsg == msg {
			return true
		}

		currMsg = msg
		return false
	})
	Require(t, err)

	// Make sure it is a totally new account
	newAccount := "User10"
	l2Info.GenerateAccount(newAccount)
	addr := l2Info.GetAddress(newAccount)
	balance := l2Node.GetBalance(t, addr)
	if balance.Cmp(big.NewInt(0)) > 0 {
		Fatal(t, "empty account")
	}

	// Check if the tx is executed correctly
	transferAmount := big.NewInt(1e16)
	tx := l2Info.PrepareTx("Faucet", newAccount, 3e7, transferAmount, nil)
	err = l2Node.Client.SendTransaction(ctx, tx)
	Require(t, err)

	err = waitFor(t, ctx, func() bool {
		balance := l2Node.GetBalance(t, addr)
		log.Info("waiting for balance", "addr", addr, "balance", balance)
		return balance.Cmp(transferAmount) >= 0
	})
	Require(t, err)

	msg, err := node.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)

	if msg != currMsg+1 {
		t.Fatal("")
	}

	err = waitForWith(t, ctx, 60*time.Second, 5*time.Second, func() bool {
		validatedCnt := node.ConsensusNode.BlockValidator.Validated(t)
		return validatedCnt >= msg
	})
	Require(t, err)

	err = seq.SetMode(ctx, true)
	Require(t, err)

	expectedMsg := msg + 10
	err = waitForWith(t, ctx, 120*time.Second, 5*time.Second, func() bool {
		msg, err := node.ConsensusNode.TxStreamer.GetMessageCount()
		if err != nil {
			return false
		}
		return msg >= expectedMsg
	})
	Require(t, err)
	err = waitForWith(t, ctx, 60*time.Second, 5*time.Second, func() bool {
		validatedCnt := node.ConsensusNode.BlockValidator.Validated(t)
		return validatedCnt >= expectedMsg
	})
	Require(t, err)
}
