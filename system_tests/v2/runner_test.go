// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"context"
	"fmt"
	"runtime"
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
// Test names appear as: TestRunner/worker-N/TestFoo
// or TestRunner/TestFoo (sequential).
func TestRunner(t *testing.T) {
	params := ParseCLIParams()

	type workItem struct {
		name string
		spec *BuilderSpec
		run  func(*TestEnv)
	}

	var parallelWork []workItem
	var seqWork []workItem

	for _, entry := range GetRegistry() {
		specs := entry.Config(params)
		for _, spec := range specs {
			item := workItem{
				name: TestName(entry.Name, spec),
				spec: spec,
				run:  entry.Run,
			}
			if spec.Parallelizable {
				parallelWork = append(parallelWork, item)
			} else {
				seqWork = append(seqWork, item)
			}
		}
	}

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

	// Feed all parallel work into a channel that workers drain.
	ch := make(chan workItem, len(parallelWork))
	for _, item := range parallelWork {
		ch <- item
	}
	close(ch)

	// Spawn workers. Each worker calls t.Parallel() once; it then runs multiple
	// tests as subtests, back-to-back. Tests inside a worker never call
	// t.Parallel() themselves — that is the key to the concurrency model.
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount > len(parallelWork) {
		workerCount = len(parallelWork)
	}
	for i := range workerCount {
		t.Run(fmt.Sprintf("worker-%d", i), func(t *testing.T) {
			t.Parallel()
			for item := range ch {
				item := item
				t.Run(item.name, func(t *testing.T) {
					runOne(t, item.spec, item.run)
				})
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
