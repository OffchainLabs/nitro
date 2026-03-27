// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package stopwaiter

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter/state"
	"github.com/offchainlabs/nitro/util/stopwaiter/stoppable"
)

// Re-exported for callers' convenience: use stopwaiter.Stoppable / stopwaiter.StoppableChild
// instead of importing the internal stoppable sub-package directly.
type Stoppable = stoppable.Stoppable
type StoppableChild = stoppable.StoppableChild

const stopDelayWarningTimeout = 30 * time.Second

type StopWaiterSafe struct {
	state.InternalState
	wg sync.WaitGroup
}

func (s *StopWaiterSafe) Started() bool {
	st := s.RLock()
	defer s.RUnlock()
	return st.Started
}

func (s *StopWaiterSafe) Stopped() bool {
	st := s.RLock()
	defer s.RUnlock()
	return st.Stopped
}

func (s *StopWaiterSafe) GetContextSafe() (context.Context, error) {
	st := s.RLock()
	defer s.RUnlock()
	return st.GetContext()
}

// this context is not cancelled even after someone calls Stop
func (s *StopWaiterSafe) GetParentContextSafe() (context.Context, error) {
	st := s.RLock()
	defer s.RUnlock()
	return st.GetParentContext()
}

// TrackChild registers a child Stoppable to be automatically stopped
// when this StopWaiter is stopped, in LIFO (reverse) order.
// If children have already been taken for shutdown, the child is stopped immediately.
// A nil child is silently ignored.
func (s *StopWaiterSafe) TrackChild(child Stoppable) {
	if child == nil {
		return
	}
	st := s.Lock()
	if st.IsChildrenTaken() {
		s.Unlock()
		child.StopAndWait()
		return
	}
	st.AppendChild(child)
	s.Unlock()
}

func getParentName(parent any) string {
	// remove asterisk in case the type is a pointer
	return strings.Replace(reflect.TypeOf(parent).String(), "*", "", 1)
}

// start-after-start will error, start-after-stop will immediately cancel
func (s *StopWaiterSafe) Start(ctx context.Context, parent any) error {
	st := s.Lock()
	defer s.Unlock()
	if st.Started {
		return errors.New("start after start")
	}
	st.Started = true
	st.Name = getParentName(parent)

	var childCtx context.Context
	childCtx, st.StopFunc = context.WithCancel(ctx)

	st.SetCtx(childCtx)
	st.SetParentCtx(ctx)

	if st.Stopped {
		st.StopFunc()
	}
	return nil
}

// takeChildren atomically takes children from the state so that
// concurrent StopOnly/StopAndWait calls don't double-stop them.
// Returns nil on subsequent calls.
// The children are also stored in TakenChildren so that stopAndWaitImpl
// can call StopAndWait on them even after StopOnly has already taken them.
func (s *StopWaiterSafe) takeChildren() []Stoppable {
	st := s.Lock()
	defer s.Unlock()
	if st.IsChildrenTaken() {
		return nil
	}
	return st.TakeChildren()
}

// StopOnly cancels the context and stops all tracked children (non-blocking).
// A subsequent StopAndWait will still wait for children's goroutines to finish.
func (s *StopWaiterSafe) StopOnly() {
	children := s.takeChildren()
	for i := len(children) - 1; i >= 0; i-- {
		children[i].StopOnly()
	}
	st := s.Lock()
	defer s.Unlock()
	if st.Started && !st.Stopped {
		st.StopFunc()
	}
	st.Stopped = true
}

// StopAndWait may be called multiple times, even before start.
func (s *StopWaiterSafe) StopAndWait() error {
	return s.stopAndWaitImpl(stopDelayWarningTimeout)
}

func getAllStackTraces() string {
	buf := make([]byte, 64*1024*1024)
	size := runtime.Stack(buf, true)
	builder := strings.Builder{}
	builder.Write(buf[0:size])
	return builder.String()
}

func (s *StopWaiterSafe) stopAndWaitImpl(warningTimeout time.Duration) error {
	children := s.takeChildren()
	if children == nil {
		// StopOnly was already called and took the children; retrieve them for waiting.
		st := s.RLock()
		children = st.GetTakenChildren()
		s.RUnlock()
	}
	for i := len(children) - 1; i >= 0; i-- {
		children[i].StopAndWait()
	}
	s.StopOnly()
	if !s.Started() {
		// No need to wait, because nothing can be started if it's already stopped.
		return nil
	}
	// Even if StopOnly has been previously called, make sure we wait for everything to shut down.
	// Otherwise, a StopOnly call followed by StopAndWait might return early without waiting.
	// At this point started must be true (because it was true above and cannot go back to false),
	// so GetWaitChannel won't return an error.
	waitChan, err := s.GetWaitChannel()
	if err != nil {
		return err
	}
	timer := time.NewTimer(warningTimeout)

	select {
	case <-timer.C:
		traces := getAllStackTraces()
		st := s.RLock()
		defer s.RUnlock()
		log.Warn("taking too long to stop", "name", st.Name, "delay[s]", warningTimeout.Seconds())
		log.Warn(traces)
	case <-waitChan:
		timer.Stop()
		return nil
	}
	<-waitChan
	return nil
}

func (s *StopWaiterSafe) GetWaitChannel() (<-chan interface{}, error) {
	st := s.Lock()
	defer s.Unlock()
	if st.WaitChan == nil {
		ctx, err := st.GetContext()
		if err != nil {
			return nil, err
		}
		waitChan := make(chan interface{})
		go func() {
			<-ctx.Done()
			s.wg.Wait()
			close(waitChan)
		}()
		st.WaitChan = waitChan
	}
	return st.WaitChan, nil
}

// If stop was already called, thread might silently not be launched
func (s *StopWaiterSafe) LaunchThreadSafe(foo func(context.Context)) error {
	ctx, err := s.GetContextSafe()
	if err != nil {
		return err
	}
	if s.Stopped() {
		return nil
	}
	st := s.RLock()
	name := st.Name
	s.RUnlock()
	s.wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("Thread crashed", "name", name, "message", r, "stack", string(debug.Stack()))
			}
		}()
		foo(ctx)
	})
	return nil
}

// This calls go foo() directly, with the benefit of being easily searchable.
// Callers may rely on the assumption that foo runs even if this is stopped.
func (s *StopWaiterSafe) LaunchUntrackedThread(foo func()) {
	go foo()
}

// CallIteratively calls function iteratively in a thread.
// input param return value is how long to wait before next invocation
func (s *StopWaiterSafe) CallIterativelySafe(foo func(context.Context) time.Duration) error {
	return s.LaunchThreadSafe(func(ctx context.Context) {
		for {
			interval := foo(ctx)
			if ctx.Err() != nil {
				return
			}
			if interval == time.Duration(0) {
				continue
			}
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	})
}

type ThreadLauncher interface {
	GetContextSafe() (context.Context, error)
	LaunchThreadSafe(foo func(context.Context)) error
	LaunchUntrackedThread(foo func())
	Stopped() bool
}

// CallIterativelyWith calls function iteratively in a thread.
// The return value of foo is how long to wait before next invocation
// Anything sent to triggerChan parameter triggers call to happen immediately
func CallIterativelyWith[T any](
	s ThreadLauncher,
	foo func(context.Context, T) time.Duration,
	triggerChan <-chan T,
) error {
	return s.LaunchThreadSafe(func(ctx context.Context) {
		var defaultVal T
		var val T
		var ok bool
		for {
			interval := foo(ctx, val)
			if ctx.Err() != nil {
				return
			}
			val = defaultVal
			if interval == time.Duration(0) {
				continue
			}
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			case val, ok = <-triggerChan:
				if !ok {
					return
				}
			}
		}
	})
}

func CallWhenTriggeredWith[T any](
	s ThreadLauncher,
	foo func(context.Context, T),
	triggerChan <-chan T,
) error {
	return s.LaunchThreadSafe(func(ctx context.Context) {
		for {
			if ctx.Err() != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case val := <-triggerChan:
				foo(ctx, val)
			}
		}
	})
}

func LaunchPromiseThread[T any](
	s ThreadLauncher,
	foo func(context.Context) (T, error),
) containers.PromiseInterface[T] {
	ctx, err := s.GetContextSafe()
	if err != nil {
		promise := containers.NewPromise[T](nil)
		promise.ProduceError(err)
		return &promise
	}
	if s.Stopped() {
		promise := containers.NewPromise[T](nil)
		promise.ProduceError(errors.New("stopped"))
		return &promise
	}
	innerCtx, cancel := context.WithCancel(ctx)
	promise := containers.NewPromise[T](cancel)
	err = s.LaunchThreadSafe(func(context.Context) { // we don't use the param's context
		defer func() {
			if r := recover(); r != nil {
				// Fulfill the promise with the panic details before re-panicking.
				// LaunchThreadSafe's outer recovery handles logging with stack trace.
				_ = promise.ProduceErrorSafe(fmt.Errorf("promise thread panicked: %v", r))
				cancel()
				panic(r)
			}
			if !promise.Ready() {
				_ = promise.ProduceErrorSafe(errors.New("promise thread exited without producing a value"))
			}
			cancel()
		}()
		val, err := foo(innerCtx)
		if err != nil {
			promise.ProduceError(err)
		} else {
			promise.Produce(val)
		}
	})
	if err != nil {
		promise.ProduceError(err)
	}
	return &promise
}

// StopWaiter may panic on race conditions instead of returning errors
type StopWaiter struct {
	StopWaiterSafe
}

func (s *StopWaiter) Start(ctx context.Context, parent any) {
	if err := s.StopWaiterSafe.Start(ctx, parent); err != nil {
		panic(err)
	}
}

// StartAndTrackChild starts a child with the parent's managed context
// and registers it for automatic shutdown in LIFO order.
// A nil child is silently ignored.
func (s *StopWaiter) StartAndTrackChild(child StoppableChild) {
	if child == nil {
		return
	}
	child.Start(s.GetContext())
	s.TrackChild(child)
}

func (s *StopWaiter) StopAndWait() {
	if err := s.StopWaiterSafe.StopAndWait(); err != nil {
		panic(err)
	}
}

// If stop was already called, thread might silently not be launched
func (s *StopWaiter) LaunchThread(foo func(context.Context)) {
	if err := s.StopWaiterSafe.LaunchThreadSafe(foo); err != nil {
		panic(err)
	}
}

func (s *StopWaiter) CallIteratively(foo func(context.Context) time.Duration) {
	if err := s.StopWaiterSafe.CallIterativelySafe(foo); err != nil {
		panic(err)
	}
}

func (s *StopWaiter) GetContext() context.Context {
	ctx, err := s.StopWaiterSafe.GetContextSafe()
	if err != nil {
		panic(err)
	}
	return ctx
}

func (s *StopWaiter) GetParentContext() context.Context {
	ctx, err := s.StopWaiterSafe.GetParentContextSafe()
	if err != nil {
		panic(err)
	}
	return ctx
}
