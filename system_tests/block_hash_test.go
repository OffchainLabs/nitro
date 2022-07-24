// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
)

func TestBlockHash(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Even though we don't use the L1, we need to create this node on L1 to get accurate L1 block numbers
	l2info, _, l2client, l2stack, _, _, _, l1stack := CreateTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1stack)
	defer requireClose(t, l2stack)

	auth := l2info.GetDefaultTransactOpts("Faucet", ctx)

	_, _, simple, err := mocksgen.DeploySimple(&auth, l2client)
	Require(t, err)

	_, err = simple.CheckBlockHashes(&bind.CallOpts{Context: ctx})
	Require(t, err)
}
