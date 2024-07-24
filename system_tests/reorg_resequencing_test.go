// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// This is a flaky test.
// During a reorg:
// - TransactionStreamer, holding insertionMutex lock, calls ExecutionEngine,
// which then adds old messages to a channel, so they can be resequenced asynchronously by ExecutionEngine.
// After that, and before releasing the lock, TransactionStreamer does more computations.
// - Asynchronously, ExecutionEngine reads from this channel and calls TransactionStreamer,
// which expects that insertionMutex is free in order to succeed. Which cause then this error:
// 'failed to re-sequence old user message removed by reorg err="insert lock taken"'
func TestReorgResequencing(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	startMsgCount, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)

	builder.L2Info.GenerateAccount("Intermediate")
	builder.L2Info.GenerateAccount("User1")
	builder.L2Info.GenerateAccount("User2")
	builder.L2Info.GenerateAccount("User3")
	builder.L2Info.GenerateAccount("User4")
	builder.L2.TransferBalance(t, "Owner", "User1", big.NewInt(params.Ether), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Intermediate", big.NewInt(params.Ether*3), builder.L2Info)
	builder.L2.TransferBalance(t, "Intermediate", "User2", big.NewInt(params.Ether), builder.L2Info)
	builder.L2.TransferBalance(t, "Intermediate", "User3", big.NewInt(params.Ether), builder.L2Info)

	// Intermediate does not have exactly 1 ether because of fees
	accountsWithBalance := []string{"User1", "User2", "User3"}
	verifyBalances := func(scenario string) {
		for _, account := range accountsWithBalance {
			balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress(account), nil)
			Require(t, err)
			if balance.Int64() != params.Ether {
				Fatal(t, "expected account", account, "to have a balance of 1 ether but instead it has", balance, "wei "+scenario)
			}
		}
	}
	verifyBalances("before reorg")

	err = builder.L2.ConsensusNode.TxStreamer.ReorgTo(startMsgCount)
	Require(t, err)

	_, err = builder.L2.ExecNode.ExecEngine.HeadMessageNumberSync(t)
	Require(t, err)

	verifyBalances("after empty reorg")
	compareAllMsgResultsFromConsensusAndExecution(t, builder.L2, "after empty reorg")

	prevMessage, err := builder.L2.ConsensusNode.TxStreamer.GetMessage(startMsgCount - 1)
	Require(t, err)
	delayedIndexHash := common.BigToHash(big.NewInt(int64(prevMessage.DelayedMessagesRead)))
	newMessage := &arbostypes.L1IncomingMessage{
		Header: &arbostypes.L1IncomingMessageHeader{
			Kind:        arbostypes.L1MessageType_EthDeposit,
			Poster:      [20]byte{},
			BlockNumber: 0,
			Timestamp:   0,
			RequestId:   &delayedIndexHash,
			L1BaseFee:   common.Big0,
		},
		L2msg: append(builder.L2Info.GetAddress("User4").Bytes(), arbmath.Uint64ToU256Bytes(params.Ether)...),
	}
	err = builder.L2.ConsensusNode.TxStreamer.AddMessages(startMsgCount, true, []arbostypes.MessageWithMetadata{{
		Message:             newMessage,
		DelayedMessagesRead: prevMessage.DelayedMessagesRead + 1,
	}})
	Require(t, err)

	_, err = builder.L2.ExecNode.ExecEngine.HeadMessageNumberSync(t)
	Require(t, err)

	accountsWithBalance = append(accountsWithBalance, "User4")

	verifyBalances("after reorg with new deposit")
	compareAllMsgResultsFromConsensusAndExecution(t, builder.L2, "after reorg with new deposit")

	err = builder.L2.ConsensusNode.TxStreamer.ReorgTo(startMsgCount)
	Require(t, err)

	_, err = builder.L2.ExecNode.ExecEngine.HeadMessageNumberSync(t)
	Require(t, err)

	verifyBalances("after second empty reorg")
	compareAllMsgResultsFromConsensusAndExecution(t, builder.L2, "after second empty reorg")
}
