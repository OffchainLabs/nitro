// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"math/big"
	"testing"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
)

func TestBlockGasLimit(t *testing.T) {
	ctx := t.Context()

	noL1 := false
	b := NewNodeBuilder(ctx).DefaultConfig(t, noL1)

	cleanup := b.Build(t)
	defer cleanup()

	auth := b.L2Info.GetDefaultTransactOpts("Owner", ctx)

	_, bigMap := b.L2.DeployBigMap(t, auth)

	// It is crucial to set the gas limit to avoid estimating gas and having
	// the gas estimation fail when it runs out of gas. This test uses the
	// receipt from the execution of the transaction to check the gas used.
	auth.GasLimit = uint64(50_000_000)
	// Store enough values to use just over 32M gas
	toAdd := big.NewInt(1358)
	toClear := big.NewInt(0)

	tx, err := bigMap.ClearAndAddValues(&auth, toClear, toAdd)
	Require(t, err)
	r := EnsureTxFailed(t, ctx, b.L2.Client, tx)
	// Should run out of gas at the transaction limit
	got := r.GasUsed
	// This should be exactly the transaction gas limit as of ArbOS 50.
	want := l2pricing.InitialPerTxGasLimitV50
	if got != want {
		t.Fatalf("want: %d gas used, got: %d", want, got)
	}
}
