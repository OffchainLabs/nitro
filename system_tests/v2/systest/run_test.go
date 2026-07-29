// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordingT captures Errorf/Skip without failing the enclosing test. Embeds sealed
// testing.TB; the nil TB is never called — runner and testify failure path only touch stubbed methods.
type recordingT struct {
	testing.TB
	ctx     context.Context
	mu      sync.Mutex
	errors  []string
	skipped bool
}

func (r *recordingT) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

func (r *recordingT) Helper()              {}
func (r *recordingT) Name() string         { return "recordingT" }
func (r *recordingT) FailNow()             { runtime.Goexit() }
func (r *recordingT) Skip(...any)          { r.mu.Lock(); r.skipped = true; r.mu.Unlock() }
func (r *recordingT) Skipf(string, ...any) { r.mu.Lock(); r.skipped = true; r.mu.Unlock() }

func (r *recordingT) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *recordingT) errCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.errors)
}

func mustPanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if !strings.Contains(fmt.Sprint(r), want) {
			t.Fatalf("expected panic containing %q, got %v", want, r)
		}
	}()
	fn()
}

func TestRunOneSkipsWithoutBuilding(t *testing.T) {
	rt := &recordingT{}
	built := false
	build := func(context.Context, Spec, overrides) (*Env, func()) {
		built = true
		return &Env{t: rt}, func() {}
	}
	runOneWith(rt, scheduledTest{Spec: Spec{Name: "X"}, SkipReason: "nope"}, build)
	if !rt.skipped {
		t.Error("expected Skip on a skip-reason item")
	}
	if built {
		t.Error("build must not run for a skipped item")
	}
}

func TestRunOneSuccessRunsHooksAndCleanup(t *testing.T) {
	rt := &recordingT{}
	cleanups, hooks := 0, 0
	item := scheduledTest{
		Spec:      Spec{Name: "X"},
		Scenario:  func(*Env) {},
		PostHooks: []Hook{func(*Env) error { hooks++; return nil }},
	}
	runOneWith(rt, item, func(context.Context, Spec, overrides) (*Env, func()) {
		return &Env{t: rt}, func() { cleanups++ }
	})
	if rt.errCount() != 0 {
		t.Errorf("unexpected errors: %v", rt.errors)
	}
	if hooks != 1 {
		t.Errorf("hooks ran %d times, want 1", hooks)
	}
	if cleanups != 1 {
		t.Errorf("cleanup ran %d times, want 1", cleanups)
	}
}

func TestRunOneScenarioPanicIsolated(t *testing.T) {
	rt := &recordingT{}
	cleanups, hooks := 0, 0
	item := scheduledTest{
		Spec:      Spec{Name: "X"},
		Scenario:  func(*Env) { panic("scenario boom") },
		PostHooks: []Hook{func(*Env) error { hooks++; return nil }},
	}
	runOneWith(rt, item, func(context.Context, Spec, overrides) (*Env, func()) {
		return &Env{t: rt}, func() { cleanups++ }
	})
	if rt.errCount() != 1 {
		t.Errorf("scenario panic should report 1 error, got %d: %v", rt.errCount(), rt.errors)
	}
	if hooks != 0 {
		t.Error("post-hooks must not run after a scenario panic")
	}
	if cleanups != 1 {
		t.Errorf("cleanup must still run after a scenario panic, got %d", cleanups)
	}
}

func TestRunOneBuildPanicIsolated(t *testing.T) {
	rt := &recordingT{}
	hooks := 0
	item := scheduledTest{
		Spec:      Spec{Name: "X"},
		Scenario:  func(*Env) {},
		PostHooks: []Hook{func(*Env) error { hooks++; return nil }},
	}
	runOneWith(rt, item, func(context.Context, Spec, overrides) (*Env, func()) {
		panic("build boom")
	})
	if rt.errCount() != 1 {
		t.Errorf("build panic should report 1 error, got %d: %v", rt.errCount(), rt.errors)
	}
	if hooks != 0 {
		t.Error("post-hooks must not run after a build panic")
	}
}

func TestRunOneBuildGoexitStillReturns(t *testing.T) {
	rt := &recordingT{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runOneWith(rt, scheduledTest{Spec: Spec{Name: "X"}, Scenario: func(*Env) {}}, func(context.Context, Spec, overrides) (*Env, func()) {
			runtime.Goexit()
			return nil, nil
		})
	}()
	<-done
	if rt.errCount() != 0 {
		t.Errorf("unexpected errors: %v", rt.errors)
	}
}

func TestGroupSchedule(t *testing.T) {
	a := Scenario(func(*Env) {})
	b := Scenario(func(*Env) {})
	c := Scenario(func(*Env) {})
	items := []scheduledTest{
		{Spec: Spec{Name: "A"}, Scenario: a},
		{Spec: Spec{Name: "A[v=1]"}, Scenario: a},
		{Spec: Spec{Name: "B"}, Scenario: b},
	}
	matched, missing := groupSchedule(items, []Scenario{a, a, c})
	if len(matched) != 2 || matched[0].Spec.Name != "A" || matched[1].Spec.Name != "A[v=1]" {
		t.Fatalf("want both registrations of a exactly once, got %d matched", len(matched))
	}
	if len(missing) != 1 || missing[0] != reflect.ValueOf(c).Pointer() {
		t.Fatalf("want exactly the unregistered scenario missing, got %v", missing)
	}
}

func TestRunOneZeroTimeoutMeansNoDeadline(t *testing.T) {
	saved := *flagTestTimeout
	t.Cleanup(func() { *flagTestTimeout = saved })
	*flagTestTimeout = 0

	rt := &recordingT{}
	var hasDeadline bool
	item := scheduledTest{
		Spec: Spec{Name: "X"},
		Scenario: func(e *Env) {
			_, hasDeadline = e.Ctx.Deadline()
		},
	}
	runOneWith(rt, item, func(ctx context.Context, _ Spec, _ overrides) (*Env, func()) {
		return &Env{t: rt, Ctx: ctx}, func() {}
	})
	if hasDeadline {
		t.Error("zero default timeout must disable the deadline backstop")
	}
	if rt.errCount() != 0 {
		t.Errorf("unexpected errors: %v", rt.errors)
	}
}

func TestRunOneScenarioDeadlineReachesEnv(t *testing.T) {
	rt := &recordingT{}
	// The scenario blocks on its ctx; it can only unblock if Spec.Timeout
	// produced a ctx with a live deadline. A broken wiring (no deadline) hangs
	// here and trips the package -timeout — the very failure this guards.
	item := scheduledTest{
		Spec: Spec{Name: "X", Timeout: 30 * time.Millisecond},
		Scenario: func(e *Env) {
			<-e.Ctx.Done()
		},
	}
	runOneWith(rt, item, func(ctx context.Context, _ Spec, _ overrides) (*Env, func()) {
		return &Env{t: rt, Ctx: ctx}, func() {}
	})
	if rt.errCount() != 1 {
		t.Errorf("runner should report the expired scenario once, got %d: %v", rt.errCount(), rt.errors)
	}
}

func TestRunOneCancelsBeforeWait(t *testing.T) {
	rt := &recordingT{}
	exited := make(chan struct{})
	item := scheduledTest{
		Spec: Spec{Name: "X", Timeout: time.Minute},
		Scenario: func(e *Env) {
			e.Go(func() error {
				<-e.Ctx.Done()
				close(exited)
				return e.Ctx.Err()
			})
		},
	}
	done := make(chan struct{})
	go func() {
		runOneWith(rt, item, func(ctx context.Context, _ Spec, _ overrides) (*Env, func()) {
			return &Env{t: rt, Ctx: ctx}, func() {}
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runOneWith did not return; cancel() likely not called before env.Wait()")
	}
	select {
	case <-exited:
	default:
		t.Error("env.Go goroutine did not observe ctx cancellation")
	}
	if rt.errCount() != 0 {
		t.Errorf("unexpected errors: %v", rt.errors)
	}
}

// TestRunOneAbortedRunCtxStopsWaitFast proves t.Context() reaches env.Wait: a cancelled
// run ctx aborts the wait (not riding envWaitTimeout) and reports an in-flight goroutine.
func TestRunOneAbortedRunCtxStopsWaitFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // whole run already aborting
	rt := &recordingT{ctx: ctx}

	release := make(chan struct{})
	defer close(release)
	item := scheduledTest{
		Spec: Spec{Name: "X", Timeout: time.Minute},
		Scenario: func(e *Env) {
			e.Go(func() error { <-release; return nil }) // ignores ctx, outlives a normal wait
		},
	}

	done := make(chan struct{})
	go func() {
		runOneWith(rt, item, func(c context.Context, _ Spec, _ overrides) (*Env, func()) {
			return &Env{t: rt, Ctx: c}, func() {}
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runOneWith blocked on env.Wait; cancelled run ctx not wired through")
	}
	if rt.errCount() != 1 {
		t.Errorf("aborted wait with in-flight goroutine should report once, got %d: %v", rt.errCount(), rt.errors)
	}
}

// TestRunOnePostHookSeesLiveCtx guards the LIFO teardown order: post-hooks must
// run before cancel(), since validateToHead waits on env.Ctx. A future reorder
// (cancel before hooks) would silently break validation.
func TestRunOnePostHookSeesLiveCtx(t *testing.T) {
	rt := &recordingT{}
	hookRan := false
	var ctxErr error
	item := scheduledTest{
		Spec:     Spec{Name: "X", Timeout: time.Minute},
		Scenario: func(*Env) {},
		PostHooks: []Hook{func(e *Env) error {
			hookRan = true
			ctxErr = e.Ctx.Err()
			return nil
		}},
	}
	runOneWith(rt, item, func(ctx context.Context, _ Spec, _ overrides) (*Env, func()) {
		return &Env{t: rt, Ctx: ctx}, func() {}
	})
	if !hookRan {
		t.Fatal("post-hook did not run")
	}
	if ctxErr != nil {
		t.Errorf("post-hook saw cancelled ctx (%v); hooks must run before cancel", ctxErr)
	}
}

func TestRunOneTeardownPanicIsolated(t *testing.T) {
	rt := &recordingT{}
	item := scheduledTest{Spec: Spec{Name: "X"}, Scenario: func(*Env) {}}
	// cleanup panics (the "close of closed channel" class in node shutdown):
	// must be recovered, not crash the worker binary.
	runOneWith(rt, item, func(context.Context, Spec, overrides) (*Env, func()) {
		return &Env{t: rt}, func() { panic("close of closed channel") }
	})
	if rt.errCount() != 1 {
		t.Errorf("teardown panic should report 1 error, got %d: %v", rt.errCount(), rt.errors)
	}
}

func TestScenarioAssertionFailureCleansUp(t *testing.T) {
	rt := &recordingT{}
	cleanedUp := false
	hooks := 0
	item := scheduledTest{
		Spec:      Spec{Name: "X"},
		Scenario:  func(e *Env) { e.Require(errors.New("boom")) },
		PostHooks: []Hook{func(*Env) error { hooks++; return nil }},
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runOneWith(rt, item, func(context.Context, Spec, overrides) (*Env, func()) {
			return &Env{t: rt}, func() { cleanedUp = true }
		})
	}()
	<-done
	if rt.errCount() != 1 {
		t.Fatalf("want exactly 1 recorded error from the failed assertion, got %d: %v", rt.errCount(), rt.errors)
	}
	if hooks != 0 {
		t.Fatal("post-hooks ran after a failed scenario")
	}
	if !cleanedUp {
		t.Fatal("cleanup skipped after a scenario assertion failure — nodes would leak")
	}
}

func TestPostHookGoexitStillCleansUp(t *testing.T) {
	rt := &recordingT{}
	cleanedUp := false
	item := scheduledTest{
		Spec:      Spec{Name: "X"},
		Scenario:  func(*Env) {},
		PostHooks: []Hook{func(*Env) error { runtime.Goexit(); return nil }},
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runOneWith(rt, item, func(context.Context, Spec, overrides) (*Env, func()) {
			return &Env{t: rt}, func() { cleanedUp = true }
		})
	}()
	<-done
	if !cleanedUp {
		t.Fatal("cleanup skipped after a post-hook Goexit (t.Fatal) — nodes would leak")
	}
}

func TestPostHookErrorAndPanicIsolated(t *testing.T) {
	rt := &recordingT{}
	ran := 0
	item := scheduledTest{
		Spec:     Spec{Name: "X"},
		Scenario: func(*Env) {},
		PostHooks: []Hook{
			func(*Env) error { ran++; return errors.New("hook boom") },
			func(*Env) error { ran++; panic("hook panic") },
			func(*Env) error { ran++; return nil },
		},
	}
	runOneWith(rt, item, func(ctx context.Context, _ Spec, _ overrides) (*Env, func()) {
		return &Env{t: rt, Ctx: ctx}, func() {}
	})
	if ran != 3 {
		t.Fatalf("want all 3 hooks to run despite error and panic, got %d", ran)
	}
	if rt.errCount() != 2 {
		t.Fatalf("want 2 recorded failures (error + panic), got %d", rt.errCount())
	}
}
