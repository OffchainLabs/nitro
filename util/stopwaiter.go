package util

import (
	"context"
	"errors"
	"sync"
	"time"
)

type StopWaiterSafe struct {
	mutex    sync.Mutex // protects started, stopped, ctx, stopFunc
	started  bool
	stopped  bool
	ctx      context.Context
	stopFunc func()

	wg sync.WaitGroup
}

func (s *StopWaiterSafe) Started() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.started
}

func (s *StopWaiterSafe) Stopped() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.stopped
}

func (s *StopWaiterSafe) GetContext() (context.Context, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started {
		return s.ctx, nil
	}
	return nil, errors.New("not started")
}

// start-after-start will error, start-after-stop will immediately cancel
func (s *StopWaiterSafe) Start(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started {
		return errors.New("start after start")
	}
	s.started = true
	s.ctx, s.stopFunc = context.WithCancel(ctx)
	if s.stopped {
		s.stopFunc()
	}
	return nil
}

func (s *StopWaiterSafe) StopOnly() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started && !s.stopped {
		s.stopFunc()
	}
	s.stopped = true
}

// Stopping multiple times, even before start, will work
func (s *StopWaiterSafe) StopAndWait() {
	s.StopOnly()
	s.wg.Wait()
}

// If stop was already called, thread might silently not be launched
func (s *StopWaiterSafe) LaunchThread(foo func(context.Context)) error {
	ctx, err := s.GetContext()
	if err != nil {
		return err
	}
	if s.Stopped() {
		return nil
	}
	s.wg.Add(1)
	go func() {
		foo(ctx)
		s.wg.Done()
	}()
	return nil
}

// This calls go foo() directly, with the benefit of being easily searchable
func (s *StopWaiterSafe) LaunchUntrackedThread(foo func()) {
	go foo()
}

// call function iteratively in a thread.
// input param return value is how long to wait before next invocation
func (s *StopWaiterSafe) CallIteratively(foo func(context.Context) time.Duration) error {
	return s.LaunchThread(func(ctx context.Context) {
		for {
			interval := foo(ctx)
			WaitForContextOrTimeout(ctx, interval)
			if ctx.Err() != nil {
				return
			}
		}
	})
}

// May panic on race conditions instead of returning errors
type StopWaiter struct {
	StopWaiterSafe
}

func (s *StopWaiter) Start(ctx context.Context) {
	if err := s.StopWaiterSafe.Start(ctx); err != nil {
		panic(err)
	}
}

func (s *StopWaiter) LaunchThread(foo func(context.Context)) {
	if err := s.StopWaiterSafe.LaunchThread(foo); err != nil {
		panic(err)
	}
}

func (s *StopWaiter) CallIteratively(foo func(context.Context) time.Duration) {
	if err := s.StopWaiterSafe.CallIteratively(foo); err != nil {
		panic(err)
	}
}

func (s *StopWaiter) GetContext() context.Context {
	ctx, err := s.StopWaiterSafe.GetContext()
	if err != nil {
		panic(err)
	}
	return ctx
}

func WaitForContextOrTimeout(ctx context.Context, timeout time.Duration) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	<-timeoutCtx.Done()
	cancel()
}
