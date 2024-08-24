// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
)

func TestPendingBlockTimeAndNumberAdvance(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)

	_, _, testTimeAndNr, err := mocksgen.DeployPendingBlkTimeAndNrAdvanceCheck(&auth, builder.L2.Client)
	Require(t, err)

	time.Sleep(1 * time.Second)

	_, err = testTimeAndNr.IsAdvancing(&auth)
	Require(t, err)
}

func TestPendingBlockArbBlockHashReturnsLatest(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)

	_, _, pendingBlk, err := mocksgen.DeployPendingBlkTimeAndNrAdvanceCheck(&auth, builder.L2.Client)
	Require(t, err)

	header, err := builder.L2.Client.HeaderByNumber(ctx, nil)
	Require(t, err)

	_, err = pendingBlk.CheckArbBlockHashReturnsLatest(&auth, header.Hash())
	Require(t, err)
}
