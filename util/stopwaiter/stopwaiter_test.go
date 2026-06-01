// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package stopwaiter

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

const testStopDelayWarningTimeout = 350 * time.Millisecond

type TestStruct struct{}

type TestChild struct {
	StopWaiter
}

func (c *TestChild) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)
}

func TestStopWaiterStopAndWaitTimeoutShouldWarn(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)
	sw := StopWaiter{}
	testCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sw.Start(context.Background(), &TestStruct{})
	sw.LaunchThread(func(ctx context.Context) {
		<-testCtx.Done()
	})
	go func() {
		err := sw.stopAndWaitImpl(testStopDelayWarningTimeout)
		testhelpers.RequireImpl(t, err)
	}()
	time.Sleep(testStopDelayWarningTimeout + 100*time.Millisecond)
	if !logHandler.WasLogged("taking too long to stop") {
		testhelpers.FailImpl(t, "Failed to log about waiting long on StopAndWait")
	}
}

func TestStopWaiterStopAndWaitTimeoutShouldNotWarn(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)
	sw := StopWaiter{}
	sw.Start(context.Background(), &TestStruct{})
	sw.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
	})
	sw.StopAndWait()
	if logHandler.WasLogged("taking too long to stop") {
		testhelpers.FailImpl(t, "Incorrectly logged about waiting long on StopAndWait")
	}
}

func TestStopWaiterStopAndWaitBeforeStart(t *testing.T) {
	sw := StopWaiter{}
	sw.StopAndWait()
}

func TestStopWaiterStopAndWaitAfterStop(t *testing.T) {
	sw := StopWaiter{}
	sw.Start(context.Background(), &TestStruct{})
	ctx := sw.GetContext()
	sw.StopOnly()
	<-ctx.Done()
	sw.StopAndWait()
}

func TestStopWaiterStopAndWaitMultipleTimes(t *testing.T) {
	sw := StopWaiter{}
	sw.StopAndWait()
	sw.StopAndWait()
	sw.StopAndWait()
	sw.Start(context.Background(), &TestStruct{})
	sw.StopAndWait()
	sw.StopAndWait()
	sw.StopAndWait()
}

func TestTrackChildStopAndWaitOrder(t *testing.T) {
	var order []string

	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})
	parent.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
		order = append(order, "parent")
	})

	child1 := StopWaiter{}
	child1.Start(parent.GetContext(), &TestStruct{})
	child1.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
		order = append(order, "child1")
	})
	parent.TrackChild(&child1)

	child2 := StopWaiter{}
	child2.Start(parent.GetContext(), &TestStruct{})
	child2.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
		order = append(order, "child2")
	})
	parent.TrackChild(&child2)

	parent.StopAndWait()

	if len(order) != 3 || order[0] != "child2" || order[1] != "child1" || order[2] != "parent" {
		t.Errorf("expected LIFO order [child2, child1, parent], got %v", order)
	}
}

func TestTrackChildStopOnly(t *testing.T) {
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})

	child := StopWaiter{}
	child.Start(parent.GetContext(), &TestStruct{})
	parent.TrackChild(&child)

	parent.StopOnly()

	if !child.Stopped() {
		t.Error("child should be stopped after parent StopOnly")
	}
}

func TestTrackChildStopAndWaitMultipleTimes(t *testing.T) {
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})

	child := StopWaiter{}
	child.Start(parent.GetContext(), &TestStruct{})
	child.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
	})
	parent.TrackChild(&child)

	parent.StopAndWait()
	parent.StopAndWait() // should not panic
}

func TestTrackChildAfterStop(t *testing.T) {
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})
	parent.StopAndWait()

	child := StopWaiter{}
	child.Start(context.Background(), &TestStruct{})
	parent.TrackChild(&child)

	if !child.Stopped() {
		t.Error("child should be stopped immediately when tracked after parent is stopped")
	}
}

func TestTrackChildContextCancellation(t *testing.T) {
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})

	child := StopWaiter{}
	child.Start(parent.GetContext(), &TestStruct{})
	childCtx := child.GetContext()
	parent.TrackChild(&child)

	parent.StopOnly()

	select {
	case <-childCtx.Done():
	default:
		t.Error("child context should be cancelled after parent StopOnly")
	}
}

func TestStartAndTrackChild(t *testing.T) {
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})

	child := TestChild{}
	parent.StartAndTrackChild(&child)

	if !child.Started() {
		t.Error("child should be started")
	}

	parent.StopAndWait()

	if !child.Stopped() {
		t.Error("child should be stopped after parent StopAndWait")
	}
}

func TestStopOnlyThenStopAndWaitWithChildren(t *testing.T) {
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})

	child := StopWaiter{}
	child.Start(parent.GetContext(), &TestStruct{})
	child.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
	})
	parent.TrackChild(&child)

	parent.StopOnly()
	if !child.Stopped() {
		t.Error("child should be stopped after parent StopOnly")
	}

	// StopAndWait should still work and wait for child goroutines
	parent.StopAndWait()
}

func TestGrandchildHierarchy(t *testing.T) {
	var order []string

	grandparent := StopWaiter{}
	grandparent.Start(context.Background(), &TestStruct{})

	parent := StopWaiter{}
	parent.Start(grandparent.GetContext(), &TestStruct{})
	parent.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
		order = append(order, "parent")
	})
	grandparent.TrackChild(&parent)

	child := StopWaiter{}
	child.Start(parent.GetContext(), &TestStruct{})
	child.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
		order = append(order, "child")
	})
	parent.TrackChild(&child)

	grandparent.StopAndWait()

	if len(order) != 2 || order[0] != "child" || order[1] != "parent" {
		t.Errorf("expected [child, parent], got %v", order)
	}
}

func TestConcurrentTrackAndStop(t *testing.T) {
	t.Parallel()
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})

	const n = 100
	children := make([]*StopWaiter, n)
	for i := range children {
		children[i] = &StopWaiter{}
		children[i].Start(context.Background(), &TestStruct{})
	}

	// Track all children concurrently with stop.
	done := make(chan struct{})
	go func() {
		for _, child := range children {
			parent.TrackChild(child)
		}
		close(done)
	}()

	parent.StopAndWait()
	<-done

	// Every child must be stopped: either taken by takeChildren, or caught by
	// the TrackChild-after-stop safety net (ChildrenTaken=true → immediate StopAndWait).
	for i, child := range children {
		if !child.Stopped() {
			t.Errorf("child %d was not stopped", i)
		}
	}
}

func TestStopWaiterStopOnlyThenStopAndWait(t *testing.T) {
	t.Parallel()
	sw := StopWaiter{}
	sw.Start(context.Background(), &TestStruct{})
	var threadStopping atomic.Bool
	sw.LaunchThread(func(context.Context) {
		time.Sleep(time.Second)
		threadStopping.Store(true)
	})
	sw.StopOnly()
	sw.StopAndWait()
	if !threadStopping.Load() {
		t.Error("StopAndWait returned before background thread stopped")
	}
}

func TestStartAndTrackChildAfterStop(t *testing.T) {
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})
	parent.StopAndWait()

	// StartAndTrackChild on a stopped parent: child.Start receives a cancelled context,
	// and TrackChild's safety net (ChildrenTaken=true) immediately calls child.StopAndWait.
	child := TestChild{}
	parent.StartAndTrackChild(&child)

	if !child.Stopped() {
		t.Error("child should be stopped when started on an already-stopped parent")
	}
}

// TestStopOnlyThenStopAndWaitIndependentGoroutine documents that StopAndWait does NOT
// wait for goroutines launched via LaunchUntrackedThread — those are outside the managed
// lifecycle and the parent shuts down without blocking on them.
func TestStopOnlyThenStopAndWaitIndependentGoroutine(t *testing.T) {
	t.Parallel()
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})

	child := StopWaiter{}
	child.Start(parent.GetContext(), &TestStruct{})
	parent.TrackChild(&child)

	slowDone := make(chan struct{})
	// This goroutine is untracked: child.StopAndWait will not wait for it.
	child.LaunchUntrackedThread(func() {
		time.Sleep(10 * time.Second)
		close(slowDone)
	})

	parent.StopOnly()
	parent.StopAndWait() // must return promptly; does not wait for the untracked goroutine

	select {
	case <-slowDone:
		t.Error("untracked goroutine finished unexpectedly fast; test is invalid")
	default:
		// correct: StopAndWait returned without waiting for the untracked goroutine
	}
}

// Before the fix, stopAndWaitImpl held an RLock across <-waitChan, so any
// goroutine that needed the write lock (StopOnly or StopAndWait) would deadlock.
func TestStopAndWaitNoDeadlockWhenGoroutineNeedsLock(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		stopFn func(*StopWaiter)
	}{
		{"StopOnly", (*StopWaiter).StopOnly},
		{"StopAndWait", (*StopWaiter).StopAndWait},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			parent := StopWaiter{}
			parent.Start(context.Background(), &TestStruct{})

			child := StopWaiter{}
			child.Start(parent.GetContext(), &TestStruct{})
			parent.TrackChild(&child)

			stopFn := tc.stopFn
			parent.LaunchThread(func(ctx context.Context) {
				time.Sleep(testStopDelayWarningTimeout + 200*time.Millisecond)
				stopFn(&child)
			})

			done := make(chan struct{})
			var stopErr error
			go func() {
				stopErr = parent.stopAndWaitImpl(testStopDelayWarningTimeout)
				close(done)
			}()

			select {
			case <-done:
				if stopErr != nil {
					t.Errorf("stopAndWaitImpl returned unexpected error: %v", stopErr)
				}
			case <-time.After(5 * time.Second):
				t.Fatalf("stopAndWaitImpl deadlocked: goroutine calling %s could not acquire lock", tc.name)
			}
		})
	}
}

func TestLaunchThreadSafePanicRecovery(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)
	sw := StopWaiter{}
	sw.Start(context.Background(), &TestStruct{})

	sw.LaunchThread(func(ctx context.Context) {
		panic("test panic message")
	})

	sw.StopAndWait()

	if !logHandler.WasLogged("Thread crashed") {
		t.Error("expected 'Thread crashed' log entry after panicking goroutine")
	}
	if !logHandler.WasLoggedWithAttr("Thread crashed", "message", "test panic message") {
		t.Error("expected panic message in 'Thread crashed' log entry")
	}
	if !logHandler.WasLoggedWithAttr("Thread crashed", "stack", "stopwaiter_test.go") {
		t.Error("expected stack trace in 'Thread crashed' log entry")
	}
}

func TestStopOnlyThenStopAndWaitWaitsForChildGoroutines(t *testing.T) {
	t.Parallel()
	parent := StopWaiter{}
	parent.Start(context.Background(), &TestStruct{})

	child := StopWaiter{}
	child.Start(parent.GetContext(), &TestStruct{})
	var childDone atomic.Bool
	child.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
		time.Sleep(100 * time.Millisecond)
		childDone.Store(true)
	})
	parent.TrackChild(&child)

	parent.StopOnly()
	parent.StopAndWait()
	if !childDone.Load() {
		t.Error("StopAndWait returned before child goroutine finished")
	}
}
