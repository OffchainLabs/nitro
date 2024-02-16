// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestSequencerWhitelist(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.execConfig.Sequencer.SenderWhitelist = GetTestAddressForAccountName(t, "Owner").String() + "," + GetTestAddressForAccountName(t, "User").String()
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User")
	builder.L2Info.GenerateAccount("User2")

	// Owner is on the whitelist
	builder.L2.TransferBalance(t, "Owner", "User", big.NewInt(params.Ether), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "User2", big.NewInt(params.Ether), builder.L2Info)

	// User is on the whitelist
	builder.L2.TransferBalance(t, "User", "User2", big.NewInt(params.Ether/10), builder.L2Info)

	// User2 is *not* on the whitelist, therefore this should fail
	tx := builder.L2Info.PrepareTx("User2", "User", builder.L2Info.TransferGas, big.NewInt(params.Ether/10), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	if err == nil {
		Fatal(t, "transaction from user not on whitelist accepted")
	}
}
