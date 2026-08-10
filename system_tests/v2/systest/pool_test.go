// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/semaphore"
)

func TestAcquireOrReportDropReportsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sema := semaphore.NewWeighted(int64(weightMax))
	rt := &recordingT{}
	item := scheduledTest{Spec: Spec{Name: "X", Weight: weightLight}}
	if acquireOrReport(rt, ctx, sema, item) {
		t.Fatal("acquire on a cancelled context must return false")
	}
	if rt.errCount() != 1 {
		t.Fatalf("dropped item must report exactly one error, got %d", rt.errCount())
	}
}

func TestAcquireOrReportSucceeds(t *testing.T) {
	sema := semaphore.NewWeighted(int64(weightMax))
	rt := &recordingT{}
	item := scheduledTest{Spec: Spec{Name: "X", Weight: weightLight}}
	if !acquireOrReport(rt, context.Background(), sema, item) {
		t.Fatal("acquire with available capacity must return true")
	}
	if rt.errCount() != 0 {
		t.Fatalf("successful acquire must not report, got %d", rt.errCount())
	}
}

func poolItems(prefix string, n int, w weight) []scheduledTest {
	items := make([]scheduledTest, n)
	for i := range items {
		items[i] = scheduledTest{Spec: Spec{Name: fmt.Sprintf("%s%d", prefix, i), Weight: w}}
	}
	return items
}

func TestPlanPool(t *testing.T) {
	cases := []struct {
		name        string
		capacity    int
		items       []scheduledTest
		wantWorkers int
		wantErr     bool
	}{
		{name: "more items than capacity", capacity: 4, items: poolItems("T", 10, weightLight), wantWorkers: 4},
		{name: "fewer items than capacity", capacity: 8, items: poolItems("T", 3, weightLight), wantWorkers: 3},
		{name: "capacity equals heaviest weight", capacity: 4, items: poolItems("T", 2, weightMax), wantWorkers: 2},
		{name: "capacity below heaviest weight", capacity: 2, items: poolItems("T", 2, weightMax), wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			workers, err := planPool(c.capacity, c.items)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if workers != c.wantWorkers {
				t.Fatalf("workers = %d, want %d", workers, c.wantWorkers)
			}
		})
	}
}

// TestRunPoolRunsEachItemOnce checks the queue is drained exactly once across
// the worker pool — no dropped or double-run items.
func TestRunPoolRunsEachItemOnce(t *testing.T) {
	const n = 25
	items := poolItems("T", n, weightLight)
	var mu sync.Mutex
	counts := map[string]int{}
	// t.Run blocks until the pool's parallel workers finish, so counts is
	// complete (and race-free) once it returns.
	t.Run("pool", func(t *testing.T) {
		runPool(t, 4, items, func(t *testing.T, item scheduledTest) {
			mu.Lock()
			counts[item.Spec.Name]++
			mu.Unlock()
		})
	})
	if len(counts) != n {
		t.Fatalf("ran %d distinct items, want %d", len(counts), n)
	}
	for name, c := range counts {
		if c != 1 {
			t.Errorf("item %q ran %d times, want 1", name, c)
		}
	}
}

func TestRunPoolSingleItemSkipsWorkerLayer(t *testing.T) {
	ran := 0
	t.Run("pool", func(t *testing.T) {
		runPool(t, 1, poolItems("T", 1, weightLight), func(t *testing.T, item scheduledTest) {
			ran++
			if strings.Contains(t.Name(), "worker-") {
				t.Errorf("single item ran under a worker subtest: %s", t.Name())
			}
		})
	})
	if ran != 1 {
		t.Fatalf("ran %d times, want 1", ran)
	}
}

func TestPoolCapacityFloorsToMaxWeight(t *testing.T) {
	items := []scheduledTest{{Spec: Spec{Weight: weightLight}}, {Spec: Spec{Weight: weightMax}}}
	// A base below the heaviest weight (small runner) is raised so planPool can't deadlock.
	if got := poolCapacity(1, items); got != int(weightMax) {
		t.Fatalf("poolCapacity(1) = %d, want %d", got, int(weightMax))
	}
	// A base above the heaviest weight is preserved.
	if got := poolCapacity(8, items); got != 8 {
		t.Fatalf("poolCapacity(8) = %d, want 8", got)
	}
	if _, err := planPool(poolCapacity(1, items), items); err != nil {
		t.Fatalf("floored capacity must not deadlock planPool: %v", err)
	}
}

func TestRunPoolRespectsWeightCapacity(t *testing.T) {
	const capacity = 4
	var items []scheduledTest
	items = append(items, poolItems("light", 6, weightLight)...)
	items = append(items, poolItems("med", 4, weightMedium)...)
	items = append(items, poolItems("max", 2, weightMax)...)

	var mu sync.Mutex
	var cur, peak int
	t.Run("pool", func(t *testing.T) {
		runPool(t, capacity, items, func(t *testing.T, item scheduledTest) {
			mu.Lock()
			cur += int(item.Spec.Weight)
			if cur > peak {
				peak = cur
			}
			mu.Unlock()
			time.Sleep(time.Millisecond) // hold the slot so overlaps surface
			mu.Lock()
			cur -= int(item.Spec.Weight)
			mu.Unlock()
		})
	})
	if peak > capacity {
		t.Fatalf("peak concurrent weight %d exceeded capacity %d", peak, capacity)
	}
	if peak == 0 {
		t.Fatal("no items ran")
	}
}

func TestSkippedItemsDontConsumeCapacity(t *testing.T) {
	// A skipped weight-Max item must neither raise the required capacity nor
	// acquire slots: capacity 1 fits because only runnable items count.
	items := []scheduledTest{
		{Spec: Spec{Name: "SkippedHeavy", Weight: weightMax}, SkipReason: "unsupported state scheme"},
		{Spec: Spec{Name: "Light1", Weight: weightLight}},
		{Spec: Spec{Name: "Light2", Weight: weightLight}},
	}
	if got := maxWeight(items); got != weightLight {
		t.Fatalf("maxWeight = %d, want %d (skipped items excluded)", got, weightLight)
	}
	if _, err := planPool(1, items); err != nil {
		t.Fatalf("planPool(1) = %v, want nil (skipped heavy item must not require capacity)", err)
	}
	var mu sync.Mutex
	ran := map[string]bool{}
	t.Run("pool", func(t *testing.T) {
		runPool(t, 1, items, func(t *testing.T, item scheduledTest) {
			mu.Lock()
			ran[item.Spec.Name] = true
			mu.Unlock()
			if item.SkipReason != "" {
				t.Skip(item.SkipReason)
			}
		})
	})
	if len(ran) != 3 {
		t.Fatalf("ran %d items, want 3: %v", len(ran), ran)
	}
}
