// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

// defaultTestTimeout is the per-scenario wall-clock backstop when no
// override is given.
const defaultTestTimeout = 10 * time.Minute

// Run is the framework entry point: it schedules the registered tests and
// runs them through the weighted worker pool. Call it
// from a single test in a package that blank-imports the test subpackages.
//
// Run is the sole concurrency authority — workers call t.Parallel(), scenarios
// never do. Names appear as TestRunner/worker-N/Foo[/matrix-suffix]; a single
// scheduled test collapses to TestRunner/Foo.
func Run(t *testing.T) {
	sp := parseCLI()
	items := schedule(sp)

	if len(items) == 0 {
		t.Skip("no work scheduled (registry empty or filtered out)")
	}

	if isDryRun() {
		printDryRun(items)
		t.Skip("dry run: no tests executed")
	}

	skipped := scheduleStats(items)
	t.Logf("scheduled %d test runs (%d skipped)", len(items)-skipped, skipped)

	base := baseCapacity()
	capacity := poolCapacity(base, items)
	if capacity > base {
		t.Logf("systest: capacity %d raised to %d to fit the heaviest scheduled test", base, capacity)
	}
	runPool(t, capacity, items, runOne)
}

// RunTestMain is the shared TestMain body.
func RunTestMain(m *testing.M) {
	code := m.Run()
	validators.shutdown()
	os.Exit(code)
}

// RunScenario runs every registration of scenario — each Named/matrix variant
// the func was registered under, with its own options.
func RunScenario(t *testing.T, scenario Scenario) {
	RunGroup(t, []Scenario{scenario})
}

// RunGroup is the IDE entry point: it schedules the registered tests through
// the same schedule the runner uses, then runs those matching the given
// scenario funcs. Filters, pins, and category gating are ignored; registered
// options (matrix, WithValidation, …) are honored; dedupes by func.
func RunGroup(t *testing.T, scenarios []Scenario) {
	if os.Getenv("CI") != "" {
		t.Skip("IDE wrapper: covered by TestRunner on CI")
	}
	sp := scheduleParams{AllCategories: true, DefaultStateScheme: envDefaultScheme()}
	matched, missing := groupSchedule(schedule(sp), scenarios)
	for _, item := range matched {
		t.Run(item.Spec.Name, func(t *testing.T) { runOne(t, item) })
	}
	for _, p := range missing {
		t.Errorf("systest: scenario %s not registered", runtime.FuncForPC(p).Name())
	}
}

func groupSchedule(items []scheduledTest, scenarios []Scenario) (matched []scheduledTest, missing []uintptr) {
	want := map[uintptr]bool{}
	for _, s := range scenarios {
		want[reflect.ValueOf(s).Pointer()] = true
	}
	ran := map[uintptr]bool{}
	for _, item := range items {
		p := reflect.ValueOf(item.Scenario).Pointer()
		if !want[p] {
			continue
		}
		ran[p] = true
		matched = append(matched, item)
	}
	for _, s := range scenarios {
		p := reflect.ValueOf(s).Pointer()
		if !ran[p] {
			missing = append(missing, p)
		}
	}
	return matched, missing
}

func scheduleStats(items []scheduledTest) (skipped int) {
	for _, it := range items {
		if it.SkipReason != "" {
			skipped++
		}
	}
	return skipped
}

// poolCapacity floors base to the heaviest scheduled weight to avoid deadlock.
func poolCapacity(base int, items []scheduledTest) int {
	return max(base, int(maxWeight(items)))
}

// buildFunc constructs the node layout for a spec. buildNode in production; a stub
// in unit tests so runOneWith's teardown/recover logic is exercisable without
// spinning a real node.
type buildFunc func(context.Context, Spec, overrides) (*Env, func())

func runOne(t *testing.T, item scheduledTest) {
	runOneWith(t, item, func(ctx context.Context, spec Spec, ov overrides) (*Env, func()) {
		return buildNode(t, ctx, spec, ov)
	})
}

func runOneWith(t testing.TB, item scheduledTest, build buildFunc) {
	t.Helper()

	if item.SkipReason != "" {
		t.Skip(item.SkipReason)
		return
	}

	// Backstop for ctx-aware hangs in build/scenario: fail this one item at its
	// deadline instead of riding the package -timeout; teardown and non-ctx-aware
	// calls stay unbounded. Spec.Timeout overrides the default; a
	// zero default disables the backstop.
	timeout := testTimeout()
	if item.Spec.Timeout > 0 {
		timeout = item.Spec.Timeout
	}

	runCtx := t.Context()
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(runCtx, timeout)
	} else {
		ctx, cancel = context.WithCancel(runCtx)
	}
	// satisfies vet's lostcancel even if buildNode t.Fatalfs; the inner defer
	// also calls cancel before env.wait so ctx-aware goroutines exit promptly.
	defer cancel()

	var (
		env        *Env
		cleanup    = func() {}
		scenarioOk bool
	)
	// Teardown defers, registered before buildNode so a construction panic fails
	// only this test, not the worker binary. LIFO: scenario-recover -> post-hooks
	// (ctx live unless a failure cancelled it) -> cancel/env.wait/cleanup; cleanup
	// runs last so a post-hook Goexit (fatal sugar) can't skip node teardown.
	defer func() {
		defer recoverAndReport(t, "teardown for %q", item.Spec.Name)
		cancel()
		if env != nil {
			env.wait(runCtx)
		}
		cleanup()
	}()
	defer func() {
		defer recoverAndReport(t, "post-hooks for %q", item.Spec.Name)
		if scenarioOk {
			env.Spec = item.Spec
			for _, h := range item.PostHooks {
				runPostHook(t, env, h)
			}
		}
	}()
	defer recoverAndReport(t, "scenario %q", item.Spec.Name)

	env, cleanup = build(ctx, item.Spec, item.overrides)
	item.Scenario(env)

	// A scenario that ignores ctx can return "passing" after its deadline; fail
	// it here rather than trusting every scenario to check.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Errorf("scenario %q: %v", item.Spec.Name, context.Cause(ctx))
		return
	}
	scenarioOk = true
}

func runPostHook(t testing.TB, env *Env, h Hook) {
	defer recoverAndReport(t, "post-hook")
	if err := h(env); err != nil {
		if errors.Is(err, context.Canceled) && errors.Is(env.Ctx.Err(), context.Canceled) {
			t.Logf("post-hook suppressed (env.Ctx cancelled by an earlier failure): %v", err)
			return
		}
		t.Errorf("post-hook: %v", err)
	}
}

func recoverAndReport(t testing.TB, subject string, subjectArgs ...any) {
	if r := recover(); r != nil {
		msg := fmt.Sprintf(subject, subjectArgs...)
		t.Errorf("%s panicked: %v\n%s", msg, r, debug.Stack())
	}
}
