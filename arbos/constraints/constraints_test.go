// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package constraints

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

func TestResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()

	const (
		minuteSecs = 60
		daySecs    = 24 * 60 * 60
		weekSecs   = 7 * daySecs
		monthSecs  = 30 * daySecs
	)

	// Adds a few constraints
	rc.SetConstraint(multigas.ResourceKindComputation, minuteSecs, 5_000_000*minuteSecs)
	rc.SetConstraint(multigas.ResourceKindComputation, weekSecs, 3_000_000*weekSecs)
	rc.SetConstraint(multigas.ResourceKindHistoryGrowth, monthSecs, 1_000_000*monthSecs)
	if got, want := len(rc[multigas.ResourceKindComputation]), 2; got != want {
		t.Fatalf("unexpected number of computation constraints: got %v, want %v", got, want)
	}
	if got, want := rc[multigas.ResourceKindComputation][minuteSecs].Period, time.Duration(minuteSecs)*time.Second; got != want {
		t.Errorf("unexpected constraint period: got %v, want %v", got, want)
	}
	if got, want := rc[multigas.ResourceKindComputation][minuteSecs].Target, uint64(5_000_000); got != want {
		t.Errorf("unexpected constraint target: got %v, want %v", got, want)
	}
	if got, want := rc[multigas.ResourceKindComputation][weekSecs].Period, time.Duration(weekSecs)*time.Second; got != want {
		t.Errorf("unexpected constraint period: got %v, want %v", got, want)
	}
	if got, want := rc[multigas.ResourceKindComputation][weekSecs].Target, uint64(3_000_000); got != want {
		t.Errorf("unexpected constraint target: got %v, want %v", got, want)
	}
	if got, want := len(rc[multigas.ResourceKindHistoryGrowth]), 1; got != want {
		t.Fatalf("unexpected number of history growth constraints: got %v, want %v", got, want)
	}
	if got, want := rc[multigas.ResourceKindHistoryGrowth][monthSecs].Period, time.Duration(monthSecs)*time.Second; got != want {
		t.Errorf("unexpected constraint period: got %v, want %v", got, want)
	}
	if got, want := rc[multigas.ResourceKindHistoryGrowth][monthSecs].Target, uint64(1_000_000); got != want {
		t.Errorf("unexpected constraint target: got %v, want %v", got, want)
	}
	if got, want := len(rc[multigas.ResourceKindStorageAccess]), 0; got != want {
		t.Errorf("unexpected number of storage access constraints: got %v, want %v", got, want)
	}
	if got, want := len(rc[multigas.ResourceKindStorageGrowth]), 0; got != want {
		t.Errorf("unexpected number of storage growth constraints: got %v, want %v", got, want)
	}

	// Updates a constraint
	rc.SetConstraint(multigas.ResourceKindHistoryGrowth, monthSecs, 500_000*monthSecs)
	if got, want := len(rc[multigas.ResourceKindHistoryGrowth]), 1; got != want {
		t.Fatalf("unexpected number of history growth constraints: got %v, want %v", got, want)
	}
	if got, want := rc[multigas.ResourceKindHistoryGrowth][monthSecs].Target, uint64(500_000); got != want {
		t.Errorf("unexpected constraint target: got %v, want %v", got, want)
	}

	// Clear constraints
	rc.ClearConstraint(multigas.ResourceKindComputation, minuteSecs)
	rc.ClearConstraint(multigas.ResourceKindComputation, weekSecs)
	rc.ClearConstraint(multigas.ResourceKindHistoryGrowth, monthSecs)
	if got, want := len(rc[multigas.ResourceKindComputation]), 0; got != want {
		t.Errorf("unexpected number of computation constraints: got %v, want %v", got, want)
	}
	if got, want := len(rc[multigas.ResourceKindHistoryGrowth]), 0; got != want {
		t.Errorf("unexpected number of history growth constraints: got %v, want %v", got, want)
	}
	if got, want := len(rc[multigas.ResourceKindStorageAccess]), 0; got != want {
		t.Errorf("unexpected number of storage access constraints: got %v, want %v", got, want)
	}
	if got, want := len(rc[multigas.ResourceKindStorageGrowth]), 0; got != want {
		t.Errorf("unexpected number of storage growth constraints: got %v, want %v", got, want)
	}
}
