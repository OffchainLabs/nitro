// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package sequencer

import (
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var nonceTests = []systest.Scenario{
	systest.Test(testRunSequencerParallelNonces, systest.WithoutChainOwner(),
		systest.WithDBEngine(systest.DBEnginePebble),
		systest.WithExecConfigOverride(func(cfg *gethexec.Config) { cfg.Sequencer.NonceFailureCacheExpiry = time.Minute })),
}

func testRunSequencerParallelNonces(env *systest.Env) {
	env.L2.Info.GenerateAccount("Destination")

	for range 10 {
		env.Go(func() error {
			for range 10 {
				tx := env.L2.Info.PrepareTx("Owner", "Destination", env.L2.Info.TransferGas, common.Big1, nil)
				// Sleep a random amount of time up to 20 milliseconds
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(20)))
				if err := env.L2.Client.SendTransaction(env.Ctx, tx); err != nil {
					return err
				}
			}
			return nil
		})
	}

	addr := env.L2.Info.GetAddress("Destination")
	env.WaitFor("destination balance to reach 100", func() bool {
		return env.L2.BalanceAt(addr).Cmp(big.NewInt(100)) == 0
	})
}
