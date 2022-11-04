// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestReorgResequencing(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	startMsgCount, err := node.TxStreamer.GetMessageCount()
	Require(t, err)

	l2info.GenerateAccount("Intermediate")
	l2info.GenerateAccount("User1")
	l2info.GenerateAccount("User2")
	l2info.GenerateAccount("User3")
	TransferBalance(t, "Owner", "User1", big.NewInt(params.Ether), l2info, client, ctx)
	TransferBalance(t, "Owner", "Intermediate", big.NewInt(params.Ether*3), l2info, client, ctx)
	TransferBalance(t, "Intermediate", "User2", big.NewInt(params.Ether), l2info, client, ctx)
	TransferBalance(t, "Intermediate", "User3", big.NewInt(params.Ether), l2info, client, ctx)

	// Intermediate does not have exactly 1 ether because of fees
	accountsWithBalance := []string{"User1", "User2", "User3"}
	verifyBalances := func(scenario string) {
		for _, account := range accountsWithBalance {
			balance, err := client.BalanceAt(ctx, l2info.GetAddress(account), nil)
			Require(t, err)
			if balance.Int64() != params.Ether {
				Fail(t, "expected account", account, "to have a balance of 1 ether but instead it has", balance, "wei "+scenario)
			}
		}
	}
	verifyBalances("before reorg")

	err = node.TxStreamer.ReorgTo(startMsgCount)
	Require(t, err)

	verifyBalances("after reorg")
}
