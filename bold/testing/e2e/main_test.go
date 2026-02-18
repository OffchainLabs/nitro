// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package e2e

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/bold/testing/setup"
)

func TestMain(m *testing.M) {
	if err := setup.InitMockCreatorCache(); err != nil {
		log.Crit("failed to initialize mock creator cache", "err", err)
	}
	os.Exit(m.Run())
}
