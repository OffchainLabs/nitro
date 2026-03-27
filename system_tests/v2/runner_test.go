// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"context"
	"fmt"
	"runtime"
	"testing"
)

// TestRunner is the single Go test entry point for all v2-registered tests.
//
// Concurrency model: rather than calling t.Parallel() per test (which leaves
// goroutines "running" from Go's perspective while blocked on a semaphore),
// we use a fixed worker pool. Workers call t.Parallel(); individual tests
// do NOT. This gives real concurrency control independent of -parallel.
//
// Test names appear as:
//
//	TestRunner/TestFoo                         (sequential)
//	TestRunner/worker-N/TestFoo                (parallel, no matrix)
//	TestRunner/worker-N/TestFoo/hash/arbos50   (parallel, matrix expanded)
func TestRunner(t *testing.T) {
	dims := parseMatrixDimensions()
	params := TestParams{Dims: dims}
	enabledCats := EnabledCategories()

	type workItem struct {
		name string
		spec *BuilderSpec
		run  func(*TestEnv)
	}

	var parallelWork []workItem
	var seqWork []workItem

	// Collect individual tests.
	for _, entry := range GetRegistry() {
		specs := entry.Config(params)
		for _, spec := range specs {
			// Category filtering.
			if !categoryEnabled(spec.Category, enabledCats) {
				continue
			}

			expanded := expandSpec(spec, dims)
			for _, es := range expanded {
				item := workItem{
					name: TestName(entry.Name, es),
					spec: es,
					run:  entry.Run,
				}
				if es.Parallelizable {
					parallelWork = append(parallelWork, item)
				} else {
					seqWork = append(seqWork, item)
				}
			}
		}
	}

	// Collect suite tests — each scenario becomes a workItem, but all scenarios
	// within a suite share a single node build (handled in runSuiteItem).
	for _, suite := range GetSuiteRegistry() {
		specs := suite.Config(params)
		for _, spec := range specs {
			if !categoryEnabled(spec.Category, enabledCats) {
				continue
			}

			expanded := expandSpec(spec, dims)
			for _, es := range expanded {
				// Capture suite scenarios for this spec.
				scenarios := suite.Scenarios
				suiteName := suite.Name
				specCopy := es

				item := workItem{
					name: TestName(suiteName, es),
					spec: specCopy,
					run: func(env *TestEnv) {
						runSuiteScenarios(env, scenarios)
					},
				}
				if es.Parallelizable {
					parallelWork = append(parallelWork, item)
				} else {
					seqWork = append(seqWork, item)
				}
			}
		}
	}

	t.Logf("scheduled %d parallel + %d sequential tests", len(parallelWork), len(seqWork))

	// Sequential tests run first.
	for _, item := range seqWork {
		t.Run(item.name, func(t *testing.T) {
			runOne(t, item.spec, item.run)
		})
	}

	if len(parallelWork) == 0 {
		return
	}

	// Weight-aware worker pool with diagnostic scheduler.
	capacity := MaxWeight()
	if capacity <= 0 {
		capacity = runtime.GOMAXPROCS(0)
	}

	scheduler := NewScheduler(capacity, t)

	ch := make(chan workItem, len(parallelWork))
	for _, item := range parallelWork {
		ch <- item
	}
	close(ch)

	workerCount := min(capacity, len(parallelWork))

	for i := range workerCount {
		t.Run(fmt.Sprintf("worker-%d", i), func(t *testing.T) {
			t.Parallel()
			for item := range ch {
				weight := max(int(item.spec.Weight), 1)

				ctx, cancel := context.WithCancel(context.Background())
				if err := scheduler.Acquire(ctx, weight, item.name); err != nil {
					cancel()
					t.Logf("skipping %s: %v", item.name, err)
					continue
				}

				t.Run(item.name, func(t *testing.T) {
					defer cancel()
					defer scheduler.Release(weight, item.name)
					runOne(t, item.spec, item.run)
				})
			}
		})
	}
}

// runOne is the per-test lifecycle manager.
// It builds the node, injects TestEnv, runs the test, then executes
// universal post-run checks.
func runOne(t *testing.T, spec *BuilderSpec, run func(*TestEnv)) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var env *TestEnv
	var cleanup func()
	if spec.NeedsL1 {
		env, cleanup = buildL1L2Node(t, ctx, spec)
	} else {
		env, cleanup = buildL2Node(t, ctx, spec)
	}
	defer cleanup()

	run(env)

	// --- Post-test hooks ---
	if spec.ValidateBlocks {
		validateAllBlocks(t, ctx, env)
	}
	if spec.CheckMultiNode {
		checkMultiNodeConsistency(t, ctx, env)
	}
}

// runSuiteScenarios runs multiple scenarios on a single TestEnv.
// The node is built once by runOne; this function runs each scenario
// as a sub-operation. If any scenario fails, subsequent scenarios still run.
func runSuiteScenarios(env *TestEnv, scenarios []Scenario) {
	for _, sc := range scenarios {
		env.T.Run(sc.Name, func(t *testing.T) {
			// Create a sub-env with the subtest's *testing.T.
			subEnv := &TestEnv{
				T:      t,
				Ctx:    env.Ctx,
				L2:     env.L2,
				L2Info: env.L2Info,
				Spec:   env.Spec,
			}
			sc.Run(subEnv)
		})
	}
}

// categoryEnabled checks if a spec's category is in the enabled set.
func categoryEnabled(cat TestCategory, enabled map[TestCategory]bool) bool {
	if cat == CategoryDefault || cat == "" {
		return enabled[CategoryDefault]
	}
	return enabled[cat]
}

// --- Post-test hook stubs ---
// These will be fully implemented when validation infrastructure is ported.

func validateAllBlocks(t *testing.T, _ context.Context, _ *TestEnv) {
	t.Helper()
	t.Log("POST-HOOK: validateAllBlocks (not yet implemented)")
	// Future: use StatelessBlockValidator to validate every block produced during the test.
	// This requires validation node setup (valnode.Config, machine locator).
	// The runner will automatically validate all blocks from genesis to the latest
	// block at the time testRunX completed.
}

func checkMultiNodeConsistency(t *testing.T, _ context.Context, _ *TestEnv) {
	t.Helper()
	t.Log("POST-HOOK: checkMultiNodeConsistency (not yet implemented)")
	// Future: spin up a 2nd node (follower), let it sync, then compare
	// the block hash of the latest block on both nodes.
	// This catches state divergence bugs that a single-node test misses.
}
