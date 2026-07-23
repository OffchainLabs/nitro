// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// IDE entry points: one Test wrapper per scenario group.
package rpc

import (
	"testing"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

func TestMain(m *testing.M) {
	systest.RunTestMain(m)
}

func TestSimulate(t *testing.T) { systest.RunGroup(t, simulateTests) }
