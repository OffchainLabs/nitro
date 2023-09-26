// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/offchainlabs/nitro/arbnode"
)

func TestTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testNode := NewNodeBuilder(ctx).SetNodeConfig(arbnode.ConfigDefaultL2Test()).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()

	testNode.L2Info.GenerateAccount("User2")

	tx := testNode.L2Info.PrepareTx("Owner", "User2", testNode.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := testNode.L2Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)

	bal, err := testNode.L2Client.BalanceAt(ctx, testNode.L2Info.GetAddress("Owner"), nil)
	Require(t, err)
	fmt.Println("Owner balance is: ", bal)
	bal2, err := testNode.L2Client.BalanceAt(ctx, testNode.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if bal2.Cmp(big.NewInt(1e12)) != 0 {
		Fatal(t, "Unexpected recipient balance: ", bal2)
	}
}
