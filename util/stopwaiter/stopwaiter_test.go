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

	// Launch goroutines that track children concurrently with stop
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			child := &StopWaiter{}
			child.Start(context.Background(), &TestStruct{})
			parent.TrackChild(child)
		}
		close(done)
	}()

	parent.StopAndWait()
	<-done
	// All children should be stopped (either via takeChildren or TrackChild-after-stop)
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
