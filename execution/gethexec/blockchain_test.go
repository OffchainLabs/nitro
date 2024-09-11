// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"testing"
	"time"
)

func TestGetStateHistory(t *testing.T) {
	maxBlockSpeed := time.Millisecond * 250
	expectedStateHistory := uint64(345600)
	actualStateHistory := getStateHistory(maxBlockSpeed)
	if actualStateHistory != expectedStateHistory {
		t.Errorf("Expected state history to be %d, but got %d", expectedStateHistory, actualStateHistory)
	}
}
