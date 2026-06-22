// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"
)

func TestEqualBig(t *testing.T) {
	tb := &recordingT{}
	e := &Env{T: tb}
	e.EqualBig(big.NewInt(5), big.NewInt(5))
	if tb.errCount() != 0 {
		t.Fatalf("equal values must not record errors, got %d", tb.errCount())
	}
	for _, tc := range []struct{ expected, actual *big.Int }{
		{big.NewInt(5), big.NewInt(6)},
		{big.NewInt(5), nil},
		{nil, big.NewInt(5)},
		{nil, nil},
	} {
		tb := &recordingT{}
		e := &Env{T: tb}
		done := make(chan struct{})
		go func() {
			defer close(done)
			e.EqualBig(tc.expected, tc.actual)
		}()
		<-done
		if tb.errCount() != 1 {
			t.Fatalf("EqualBig(%v, %v) must record exactly 1 error, got %d", tc.expected, tc.actual, tb.errCount())
		}
	}
}

func TestEnvGoAssertionRecordsAndExits(t *testing.T) {
	tb := &recordingT{}
	e := &Env{T: tb, Ctx: context.Background()}
	e.Go(func() error {
		e.Require(errors.New("boom"))
		return nil
	})
	e.wait(context.Background())
	if tb.errCount() != 1 {
		t.Fatalf("assertion in env.Go should record exactly 1 error, got %d", tb.errCount())
	}
	lateAssert(e, "late")
	if tb.errCount() != 1 {
		t.Fatalf("assertion on a dead env must be dropped, got %d", tb.errCount())
	}
}

// lateAssert runs e.Require on a goroutine and joins it; on a dead env the
// guard must drop the failure and Goexit.
func lateAssert(e *Env, msg string) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Require(errors.New(msg))
	}()
	<-done
}

func TestEnvGoRecoversAndReportsPanic(t *testing.T) {
	tb := &recordingT{}
	e := &Env{T: tb}
	e.Go(func() error { panic("boom") })
	e.wait(context.Background())
	if tb.errCount() != 1 {
		t.Fatalf("panic in env.Go should report exactly 1 error, got %d", tb.errCount())
	}
}

func TestEnvGoSuppressesCtxErrorsOnlyAtShutdown(t *testing.T) {
	// ctx done (teardown): ctx errors are noise → suppressed.
	tb := &recordingT{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := &Env{T: tb, Ctx: ctx}
	e.Go(func() error { return context.Canceled })
	e.Go(func() error { return fmt.Errorf("shutting down: %w", context.DeadlineExceeded) })
	e.wait(context.Background())
	if tb.errCount() != 0 {
		t.Fatalf("ctx errors during shutdown must be suppressed, got %d reports", tb.errCount())
	}

	// ctx live (mid-test): a wrapped deadline is a real failure → reported.
	tb2 := &recordingT{}
	e2 := &Env{T: tb2, Ctx: context.Background()}
	e2.Go(func() error { return fmt.Errorf("rpc timed out: %w", context.DeadlineExceeded) })
	e2.wait(context.Background())
	if tb2.errCount() != 1 {
		t.Fatalf("mid-test deadline error must be reported, got %d", tb2.errCount())
	}
}

func TestEnvGoReportsRealError(t *testing.T) {
	tb := &recordingT{}
	e := &Env{T: tb}
	e.Go(func() error { return errors.New("real failure") })
	e.wait(context.Background())
	if tb.errCount() != 1 {
		t.Fatalf("real env.Go error should report exactly 1 error, got %d", tb.errCount())
	}
}

func TestAssertionDroppedAfterWait(t *testing.T) {
	tb := &recordingT{}
	e := &Env{T: tb}
	e.wait(context.Background()) // no goroutines: closes done immediately and marks dead
	lateAssert(e, "late error")
	if tb.errCount() != 0 {
		t.Fatalf("assertion after Wait must be dropped, got %d", tb.errCount())
	}
}

func TestEnvWaitCtxCancelReportsInFlight(t *testing.T) {
	tb := &recordingT{}
	e := &Env{T: tb}
	release := make(chan struct{})
	defer close(release)
	e.Go(func() error { <-release; return nil }) // would outlive a normal wait

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // run aborting before Wait

	e.wait(ctx)
	if tb.errCount() != 1 {
		t.Fatalf("ctx-cancel with in-flight goroutine should report once, got %d", tb.errCount())
	}
	lateAssert(e, "after abort") // dead flag set by the ctx path → dropped
	if tb.errCount() != 1 {
		t.Fatalf("dead env should drop later reports, got %d", tb.errCount())
	}
}

func TestEnvWaitCtxCancelQuietWhenIdle(t *testing.T) {
	tb := &recordingT{}
	e := &Env{T: tb}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e.wait(ctx) // no in-flight goroutines: abort is not a failure
	if tb.errCount() != 0 {
		t.Fatalf("idle ctx-cancel must not report an error, got %d", tb.errCount())
	}
}

func TestEnvWaitTimeoutReportsAndMarksDead(t *testing.T) {
	tb := &recordingT{}
	e := &Env{T: tb}
	release := make(chan struct{})
	defer close(release)
	e.Go(func() error { <-release; return nil }) // outlives the wait timeout

	saved := envWaitTimeout
	envWaitTimeout = 20 * time.Millisecond
	defer func() { envWaitTimeout = saved }()

	e.wait(context.Background())
	if tb.errCount() != 1 {
		t.Fatalf("timeout should report exactly 1 error, got %d", tb.errCount())
	}
	lateAssert(e, "after timeout") // dead flag set by the timeout path → dropped
	if tb.errCount() != 1 {
		t.Fatalf("dead env should drop later reports, got %d", tb.errCount())
	}
}
