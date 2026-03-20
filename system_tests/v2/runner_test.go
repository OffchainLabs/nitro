// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// TestRunner is the single Go test that runs all v2-registered tests.
//
// Concurrency model: rather than calling t.Parallel() per test (which leaves
// goroutines "running" from Go's perspective while blocked waiting for a
// semaphore), we use a fixed worker pool. Workers call t.Parallel(); individual
// tests do NOT. This means exactly <workerCount> tests are truly running at any
// given time, giving us real concurrency control independent of -parallel.
//
// Test names appear as: TestRunner/worker-N/TestFoo[/variant][/paramLabel]
// or TestRunner/TestFoo (sequential).
func TestRunner(t *testing.T) {
	paramSets := ParseCLIParams()
	if len(paramSets) > 1 {
		t.Logf("matrix expansion: %d parameter sets", len(paramSets))
	}

	type workItem struct {
		name   string
		spec   *BuilderSpec
		run    func(*TestEnv)
		weight ResourceWeight
	}

	var parallelWork []workItem
	var seqWork []workItem

	for _, entry := range GetRegistry() {
		for _, p := range paramSets {
			specs := entry.Config(p)
			for _, spec := range specs {
				// Apply params (state scheme, arbos version, db engine overrides).
				// Returns false if the spec is incompatible with this param set.
				if !spec.ApplyParams(p) {
					continue
				}

				w := spec.Weight
				if w == 0 {
					w = WeightLight
				}

				item := workItem{
					name:   TestName(entry.Name, spec, p.Label()),
					spec:   spec,
					run:    entry.Run,
					weight: w,
				}
				if spec.Parallelizable {
					parallelWork = append(parallelWork, item)
				} else {
					seqWork = append(seqWork, item)
				}
			}
		}
	}

	t.Logf("scheduled %d parallel + %d sequential tests", len(parallelWork), len(seqWork))

	// Sequential tests run first, in the parent goroutine — no workers needed.
	for _, item := range seqWork {
		item := item
		t.Run(item.name, func(t *testing.T) {
			runOne(t, item.spec, item.run)
		})
	}

	if len(parallelWork) == 0 {
		return
	}

	// Weight-aware worker pool.
	//
	// Total capacity = max-weight flag, or GOMAXPROCS if not set.
	// Each worker holds exactly 1 slot of capacity while waiting for work,
	// then acquires up to (item.weight - 1) more before running the test.
	// This means a WeightLight test runs immediately, while a WeightHeavy
	// test blocks until enough capacity is free.
	capacity := MaxWeight()
	if capacity <= 0 {
		capacity = runtime.GOMAXPROCS(0)
	}

	ch := make(chan workItem, len(parallelWork))
	for _, item := range parallelWork {
		ch <- item
	}
	close(ch)

	// Number of workers = capacity (one slot each), capped at work count.
	workerCount := capacity
	if workerCount > len(parallelWork) {
		workerCount = len(parallelWork)
	}

	// Semaphore for weight-based scheduling.
	// Each worker already holds 1 slot by existing. For tests heavier than
	// WeightLight, the worker acquires extra slots from this semaphore.
	// Total extra slots = capacity - workerCount (the rest after each worker's base slot).
	extraSlots := capacity - workerCount
	if extraSlots < 0 {
		extraSlots = 0
	}
	sema := newWeightedSemaphore(extraSlots)

	for i := range workerCount {
		t.Run(fmt.Sprintf("worker-%d", i), func(t *testing.T) {
			t.Parallel()
			for item := range ch {
				item := item
				extra := int(item.weight) - 1
				if extra > 0 {
					sema.Acquire(extra)
				}
				t.Run(item.name, func(t *testing.T) {
					runOne(t, item.spec, item.run)
				})
				if extra > 0 {
					sema.Release(extra)
				}
			}
		})
	}
}

// runOne is the per-test lifecycle manager.
// It builds the node according to spec, injects a TestEnv into the run function,
// and defers all cleanup. Universal post-run checks (block validation, multi-node
// hash parity) will be added here in future phases.
func runOne(t *testing.T, spec *BuilderSpec, run func(*TestEnv)) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, cleanup := buildL2Node(t, ctx, spec)
	defer cleanup()

	run(env)

	// Future: universal post-run checks go here, e.g.:
	//   if spec.ValidationEnabled { waitForAllBlocksValidated(t, ctx, env) }
	//   if spec.ExtraNodes != nil  { assertSameBlockHash(t, ctx, env) }
}

// weightedSemaphore is a simple counting semaphore for weight-based scheduling.
type weightedSemaphore struct {
	mu       sync.Mutex
	cond     *sync.Cond
	capacity int
	used     int
}

func newWeightedSemaphore(capacity int) *weightedSemaphore {
	s := &weightedSemaphore{capacity: capacity}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *weightedSemaphore) Acquire(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for s.used+n > s.capacity {
		s.cond.Wait()
	}
	s.used += n
}

func (s *weightedSemaphore) Release(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.used -= n
	s.cond.Broadcast()
}
