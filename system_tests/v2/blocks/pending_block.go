// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package blocks holds tests covering block production, numbering, and hashes.
package blocks

import (
	"time"

	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var pendingBlockTests = []systest.Scenario{
	systest.Test(testRunPendingBlockArbBlockHashReturnsLatest),
	systest.Test(testRunPendingBlockTimeAndNumberAdvance),
}

func testRunPendingBlockTimeAndNumberAdvance(env *systest.Env) {
	auth := env.L2.TransactOpts("Faucet")

	_, tx, pendingBlkTimeAndNrAdvanceCheck, err := localgen.DeployPendingBlkTimeAndNrAdvanceCheck(&auth, env.L2.Client)
	env.Require(err)

	deployBlock := env.L2.HeaderByNumber(env.L2.EnsureTxSucceeded(tx).BlockNumber)
	env.WaitFor("wall clock to pass the deploy block timestamp", func() bool {
		// #nosec G115 -- current Unix time is non-negative
		return uint64(time.Now().Unix()) > deployBlock.Time
	})

	tx, err = pendingBlkTimeAndNrAdvanceCheck.IsAdvancing(&auth)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)
}

func testRunPendingBlockArbBlockHashReturnsLatest(env *systest.Env) {
	auth := env.L2.TransactOpts("Faucet")

	_, tx, pendingBlkTimeAndNrAdvanceCheck, err := localgen.DeployPendingBlkTimeAndNrAdvanceCheck(&auth, env.L2.Client)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)

	header := env.L2.HeaderByNumber(nil)

	tx, err = pendingBlkTimeAndNrAdvanceCheck.CheckArbBlockHashReturnsLatest(&auth, header.Hash())
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)
}
