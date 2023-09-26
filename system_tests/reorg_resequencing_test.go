// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

func TestReorgResequencing(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testNode := NewNodeBuilder(ctx).SetNodeConfig(arbnode.ConfigDefaultL2Test()).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()

	startMsgCount, err := testNode.L2Node.TxStreamer.GetMessageCount()
	Require(t, err)

	testNode.L2Info.GenerateAccount("Intermediate")
	testNode.L2Info.GenerateAccount("User1")
	testNode.L2Info.GenerateAccount("User2")
	testNode.L2Info.GenerateAccount("User3")
	testNode.L2Info.GenerateAccount("User4")
	TransferBalance(t, "Owner", "User1", big.NewInt(params.Ether), testNode.L2Info, testNode.L2Client, ctx)
	TransferBalance(t, "Owner", "Intermediate", big.NewInt(params.Ether*3), testNode.L2Info, testNode.L2Client, ctx)
	TransferBalance(t, "Intermediate", "User2", big.NewInt(params.Ether), testNode.L2Info, testNode.L2Client, ctx)
	TransferBalance(t, "Intermediate", "User3", big.NewInt(params.Ether), testNode.L2Info, testNode.L2Client, ctx)

	// Intermediate does not have exactly 1 ether because of fees
	accountsWithBalance := []string{"User1", "User2", "User3"}
	verifyBalances := func(scenario string) {
		for _, account := range accountsWithBalance {
			balance, err := testNode.L2Client.BalanceAt(ctx, testNode.L2Info.GetAddress(account), nil)
			Require(t, err)
			if balance.Int64() != params.Ether {
				Fatal(t, "expected account", account, "to have a balance of 1 ether but instead it has", balance, "wei "+scenario)
			}
		}
	}
	verifyBalances("before reorg")

	err = testNode.L2Node.TxStreamer.ReorgTo(startMsgCount)
	Require(t, err)

	_, err = testNode.L2Node.Execution.ExecEngine.HeadMessageNumberSync(t)
	Require(t, err)

	verifyBalances("after empty reorg")

	prevMessage, err := testNode.L2Node.TxStreamer.GetMessage(startMsgCount - 1)
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
		L2msg: append(testNode.L2Info.GetAddress("User4").Bytes(), math.U256Bytes(big.NewInt(params.Ether))...),
	}
	err = testNode.L2Node.TxStreamer.AddMessages(startMsgCount, true, []arbostypes.MessageWithMetadata{{
		Message:             newMessage,
		DelayedMessagesRead: prevMessage.DelayedMessagesRead + 1,
	}})
	Require(t, err)

	_, err = testNode.L2Node.Execution.ExecEngine.HeadMessageNumberSync(t)
	Require(t, err)

	accountsWithBalance = append(accountsWithBalance, "User4")
	verifyBalances("after reorg with new deposit")

	err = testNode.L2Node.TxStreamer.ReorgTo(startMsgCount)
	Require(t, err)

	_, err = testNode.L2Node.Execution.ExecEngine.HeadMessageNumberSync(t)
	Require(t, err)

	verifyBalances("after second empty reorg")
}
