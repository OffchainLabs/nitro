// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
)

func TestSequencerWhitelist(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := arbnode.ConfigDefaultL2Test()
	config.Sequencer.SenderWhitelist = GetTestAddressForAccountName(t, "Owner").String() + "," + GetTestAddressForAccountName(t, "User").String()
	testNode := NewNodeBuilder(ctx).SetNodeConfig(config).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()

	testNode.L2Info.GenerateAccount("User")
	testNode.L2Info.GenerateAccount("User2")

	// Owner is on the whitelist
	TransferBalance(t, "Owner", "User", big.NewInt(params.Ether), testNode.L2Info, testNode.L2Client, ctx)
	TransferBalance(t, "Owner", "User2", big.NewInt(params.Ether), testNode.L2Info, testNode.L2Client, ctx)

	// User is on the whitelist
	TransferBalance(t, "User", "User2", big.NewInt(params.Ether/10), testNode.L2Info, testNode.L2Client, ctx)

	// User2 is *not* on the whitelist, therefore this should fail
	tx := testNode.L2Info.PrepareTx("User2", "User", testNode.L2Info.TransferGas, big.NewInt(params.Ether/10), nil)
	err := testNode.L2Client.SendTransaction(ctx, tx)
	if err == nil {
		Fatal(t, "transaction from user not on whitelist accepted")
	}
}
