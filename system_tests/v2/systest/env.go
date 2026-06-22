// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Env is the runtime handle passed to a Scenario.
type Env struct {
	T    testing.TB
	Ctx  context.Context
	L2   *L2Handle
	Spec Spec

	goWG sync.WaitGroup
	// running counts in-flight env.Go goroutines, reported on a Wait timeout.
	running atomic.Int64
	// asyncMu guards writes to T. dead flips true after Wait returns; later
	// writes are dropped so a leaked goroutine can't hit a finished subtest
	// (which panics the test binary in the worker-pool model).
	asyncMu sync.Mutex
	dead    bool
}

func (e *Env) Require(err error, msgAndArgs ...any) {
	e.T.Helper()
	e.guarded(func() { require.NoError(e.T, err, msgAndArgs...) })
}

func (e *Env) Equal(expected, actual any, msgAndArgs ...any) {
	e.T.Helper()
	e.guarded(func() { require.Equal(e.T, expected, actual, msgAndArgs...) })
}

func (e *Env) EqualBig(expected, actual *big.Int, msgAndArgs ...any) {
	e.T.Helper()
	if expected == nil || actual == nil || expected.Cmp(actual) != 0 {
		e.guarded(func() {
			require.Fail(e.T, fmt.Sprintf("Not equal: expected %s, actual %s", expected, actual), msgAndArgs...)
		})
	}
}

func (e *Env) Len(object any, length int, msgAndArgs ...any) {
	e.T.Helper()
	e.guarded(func() { require.Len(e.T, object, length, msgAndArgs...) })
}

func (e *Env) Zero(i any, msgAndArgs ...any) {
	e.T.Helper()
	e.guarded(func() { require.Zero(e.T, i, msgAndArgs...) })
}

func (e *Env) Empty(object any, msgAndArgs ...any) {
	e.T.Helper()
	e.guarded(func() { require.Empty(e.T, object, msgAndArgs...) })
}

func (e *Env) NotEmpty(object any, msgAndArgs ...any) {
	e.T.Helper()
	e.guarded(func() { require.NotEmpty(e.T, object, msgAndArgs...) })
}

func (e *Env) NotNil(object any, msgAndArgs ...any) {
	e.T.Helper()
	e.guarded(func() { require.NotNil(e.T, object, msgAndArgs...) })
}

func (e *Env) ErrorContains(err error, contains string, msgAndArgs ...any) {
	e.T.Helper()
	e.guarded(func() { require.ErrorContains(e.T, err, contains, msgAndArgs...) })
}

// WaitFor polls fn until true or env.Ctx cancels. Fails the test with a
// descriptive message on timeout.
func (e *Env) WaitFor(desc string, fn func() bool) {
	e.T.Helper()
	e.Require(waitFor(e.Ctx, desc, fn))
}

// Go spawns fn in a goroutine. Errors and panics surface via t.Errorf;
// context.Canceled / DeadlineExceeded are ignored once env.Ctx is done. Joined
// by the runner via wait() after env.Ctx is cancelled, so fn must observe ctx to exit.
func (e *Env) Go(fn func() error) {
	e.running.Add(1)
	e.goWG.Go(func() {
		defer e.running.Add(-1)
		defer func() {
			if r := recover(); r != nil {
				e.guarded(func() { e.T.Errorf("env.Go panic: %v\n%s", r, debug.Stack()) })
			}
		}()
		err := fn()
		if err == nil {
			return
		}
		// Suppress ctx errors only when our own ctx is done (teardown/timeout); a
		// wrapped deadline from an unrelated call mid-test is a real failure.
		if suppressedAtShutdown(e.Ctx, err) {
			return
		}
		e.guarded(func() { e.T.Errorf("env.Go: %v", err) })
	})
}

// wait joins Go goroutines, aborting on ctx cancel or envWaitTimeout; reports an
// error if any are still in flight, then marks the env dead to silence late writes.
func (e *Env) wait(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		e.goWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		e.asyncMu.Lock()
		e.dead = true
		e.asyncMu.Unlock()
	case <-ctx.Done():
		e.asyncMu.Lock()
		if n := e.running.Load(); n > 0 {
			e.T.Errorf("env.Wait aborted (%v): %d env.Go goroutine(s) still running; their pending failures are now suppressed", context.Cause(ctx), n)
		}
		e.dead = true
		e.asyncMu.Unlock()
	case <-time.After(envWaitTimeout):
		// Record the timeout and mark dead in one critical section so the
		// failure is logged before late goroutines are silenced. A goroutine
		// that ignores ctx leaks (Go can't force-kill it) but can no longer
		// write to the finished subtest.
		e.asyncMu.Lock()
		e.T.Errorf("env.Wait timed out after %v: %d env.Go goroutine(s) still running; their pending failures are now suppressed — raise envWaitTimeout and rerun to surface the real error", envWaitTimeout, e.running.Load())
		e.dead = true
		e.asyncMu.Unlock()
	}
}

// guarded runs fn under the dead guard: once wait has marked the env dead, fn
// is dropped and the calling goroutine stopped instead of hitting a finished subtest.
func (e *Env) guarded(fn func()) {
	e.asyncMu.Lock()
	defer e.asyncMu.Unlock()
	if e.dead {
		runtime.Goexit()
	}
	fn()
}

// suppressedAtShutdown reports whether err is a ctx error to swallow because
// ctx itself is already done.
func suppressedAtShutdown(ctx context.Context, err error) bool {
	return ctx != nil && ctx.Err() != nil &&
		(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}

// envWaitTimeout bounds how long env.wait() will block.
var envWaitTimeout = 30 * time.Second
