// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"fmt"
	"testing"

	"golang.org/x/sync/semaphore"
)

// runPool is the concurrency mechanism: it sizes a weighted semaphore to
// capacity and fans work across min(capacity, len(items)) parallel worker
// subtests, invoking run per item inside a per-item subtest. A single
// scheduled item skips the worker layer for output clarity. The per-item run
// is injected so the scheduling is testable with stubs (see pool_test.go).
func runPool(t *testing.T, capacity int, items []scheduledTest, run func(*testing.T, scheduledTest)) {
	t.Helper()
	if len(items) == 0 {
		return
	}
	if len(items) == 1 {
		t.Run(items[0].Spec.Name, func(t *testing.T) { run(t, items[0]) })
		return
	}

	workers, err := planPool(capacity, items)
	if err != nil {
		t.Fatal(err)
	}

	sema := semaphore.NewWeighted(int64(capacity))
	queue := make(chan scheduledTest, len(items))
	for _, it := range items {
		queue <- it
	}
	close(queue)

	for i := range workers {
		t.Run(fmt.Sprintf("worker-%d", i), func(t *testing.T) {
			t.Parallel()
			for item := range queue {
				runPoolItem(t, t.Context(), sema, item, run)
			}
		})
	}
}

// planPool returns the worker count for items at the given capacity, or an
// error when capacity can't fit the heaviest item — which would otherwise
// deadlock the weighted semaphore.
func planPool(capacity int, items []scheduledTest) (workers int, err error) {
	heaviest := maxWeight(items)
	if capacity < int(heaviest) {
		return 0, fmt.Errorf("scheduler capacity (%d) < max scheduled weight (%d) — raise the scheduler capacity or filter out heavier tests", capacity, heaviest)
	}
	return min(capacity, len(items)), nil
}

// runPoolItem acquires the weighted slot, runs the item in a subtest, and
// always releases — even on panic. Skipped items run without acquiring.
func runPoolItem(t *testing.T, ctx context.Context, sema *semaphore.Weighted, item scheduledTest, run func(*testing.T, scheduledTest)) {
	if item.SkipReason != "" {
		t.Run(item.Spec.Name, func(t *testing.T) {
			run(t, item)
		})
		return
	}
	w := int64(item.Spec.Weight)
	if !acquireOrReport(t, ctx, sema, item) {
		return
	}
	defer sema.Release(w)
	t.Run(item.Spec.Name, func(t *testing.T) {
		run(t, item)
	})
}

// acquireOrReport acquires the item's weighted slot. A failed Acquire (ctx
// cancelled mid-drain) is reported as an error and returns false instead of
// silently dropping the item, so a partial run can't be reported as PASS.
func acquireOrReport(t testing.TB, ctx context.Context, sema *semaphore.Weighted, item scheduledTest) bool {
	if err := sema.Acquire(ctx, int64(item.Spec.Weight)); err != nil {
		t.Errorf("scheduled test %q dropped before running: %v", item.Spec.Name, err)
		return false
	}
	return true
}
