// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"sync"
	"sync/atomic"
	"testing"
)

func mustAdmitConditional(t *testing.T, set *pendingConditionalTxs, limit int) func() {
	t.Helper()
	release, ok := set.admit(limit)
	if !ok {
		t.Fatalf("admit(%d) rejected with %d in flight", limit, set.count.Load())
	}
	return release
}

func TestPendingConditionalTxsRejectsAtLimit(t *testing.T) {
	var set pendingConditionalTxs
	releaseFirst := mustAdmitConditional(t, &set, 2)
	releaseSecond := mustAdmitConditional(t, &set, 2)
	if _, ok := set.admit(2); ok {
		t.Fatal("admit(2) admitted a third tx")
	}
	if n := set.count.Load(); n != 2 {
		t.Fatalf("count after a rejection = %d, want 2", n)
	}

	releaseFirst()
	releaseThird := mustAdmitConditional(t, &set, 2)
	releaseSecond()
	releaseThird()
	if n := set.count.Load(); n != 0 {
		t.Errorf("count after releasing every holder = %d, want 0", n)
	}
}

func TestPendingConditionalTxsZeroLimitDisables(t *testing.T) {
	var set pendingConditionalTxs
	var releases []func()
	for range 10 {
		releases = append(releases, mustAdmitConditional(t, &set, 0))
	}
	if n := set.count.Load(); n != 10 {
		t.Fatalf("count while disabled = %d, want 10", n)
	}
	if _, ok := set.admit(5); ok {
		t.Fatal("admit(5) admitted with 10 in flight")
	}
	for _, release := range releases {
		release()
	}
	if n := set.count.Load(); n != 0 {
		t.Errorf("count after releasing every holder = %d, want 0", n)
	}
}

func TestPendingConditionalTxsAdmitUnderContention(t *testing.T) {
	const limit, workers, rounds = 4, 32, 1000
	var set pendingConditionalTxs
	var maxSeen atomic.Int64
	var admitted atomic.Int64
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range rounds {
				release, ok := set.admit(limit)
				if !ok {
					continue
				}
				admitted.Add(1)
				for {
					seen := maxSeen.Load()
					n := set.count.Load()
					if n <= seen || maxSeen.CompareAndSwap(seen, n) {
						break
					}
				}
				release()
			}
		}()
	}
	wg.Wait()
	if admitted.Load() == 0 {
		t.Fatal("no admission succeeded")
	}
	if n := maxSeen.Load(); n > limit {
		t.Errorf("observed %d in flight, want at most %d", n, limit)
	}
	if n := set.count.Load(); n != 0 {
		t.Errorf("count after every worker returned = %d, want 0", n)
	}
}
