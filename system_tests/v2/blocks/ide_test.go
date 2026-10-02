// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package blocks

import (
	"testing"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

func TestMain(m *testing.M) {
	systest.RunTestMain(m)
}

func TestPending(t *testing.T) { systest.RunGroup(t, pendingBlockTests) }

func TestHistoricalBlockHash(t *testing.T) { systest.RunGroup(t, historicalBlockHashTests) }
