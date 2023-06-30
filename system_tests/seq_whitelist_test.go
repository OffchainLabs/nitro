// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

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
	l2info, l2node, client := CreateTestL2WithConfig(t, ctx, nil, config, true)
	defer l2node.StopAndWait()

	l2info.GenerateAccount("User")
	l2info.GenerateAccount("User2")

	// Owner is on the whitelist
	TransferBalance(t, "Owner", "User", big.NewInt(params.Ether), l2info, client, ctx)
	TransferBalance(t, "Owner", "User2", big.NewInt(params.Ether), l2info, client, ctx)

	// User is on the whitelist
	TransferBalance(t, "User", "User2", big.NewInt(params.Ether/10), l2info, client, ctx)

	// User2 is *not* on the whitelist, therefore this should fail
	tx := l2info.PrepareTx("User2", "User", l2info.TransferGas, big.NewInt(params.Ether/10), nil)
	err := client.SendTransaction(ctx, tx)
	if err == nil {
		Fatal(t, "transaction from user not on whitelist accepted")
	}
}
