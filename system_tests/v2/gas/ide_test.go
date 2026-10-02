// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// IDE entry points: one Test wrapper per scenario group.
package gas

import (
	"testing"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

func TestMain(m *testing.M) {
	systest.RunTestMain(m)
}

func TestEstimation(t *testing.T) { systest.RunGroup(t, estimationTests) }

func TestMultigas(t *testing.T) { systest.RunGroup(t, multigasTests) }

func TestTipCollection(t *testing.T) { systest.RunGroup(t, tipCollectionTests) }
