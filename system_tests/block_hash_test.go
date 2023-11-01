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
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)

	_, _, simple, err := mocksgen.DeploySimple(&auth, builder.L2.Client)
	Require(t, err)

	_, err = simple.CheckBlockHashes(&bind.CallOpts{Context: ctx})
	Require(t, err)
}
