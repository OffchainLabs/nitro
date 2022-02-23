package util

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type StopWaiterSafe struct {
	started  uint32
	stopped  uint32
	stopFunc func()
	ctx      context.Context
	wg       sync.WaitGroup
}

func (s *StopWaiterSafe) Started() bool {
	return atomic.LoadUint32(&s.started) != 0
}

func (s *StopWaiterSafe) Stopped() bool {
	return atomic.LoadUint32(&s.stopped) != 0
}

func (s *StopWaiterSafe) GetContext() context.Context {
	return s.ctx
}

// start-after-start will error, start-after-stop will immediately cancel
func (s *StopWaiterSafe) Start(ctx context.Context) error {
	alreadyStarted := atomic.SwapUint32(&s.started, 1)
	if alreadyStarted != 0 {
		return errors.New("start after start")
	}
	s.ctx, s.stopFunc = context.WithCancel(ctx)
	if s.Stopped() {
		s.stopFunc()
	}
	return nil
}

// Stopping multiple times, even before start, will work
func (s *StopWaiterSafe) StopAndWait() {
	atomic.StoreUint32(&s.stopped, 1)
	s.stopFunc()
	s.wg.Wait()
}

// If stop was already called, thread might silently not be launched
func (s *StopWaiterSafe) LaunchThread(foo func(context.Context)) error {
	if !s.Started() {
		return errors.New("launch thread before start")
	}
	if s.Stopped() {
		return nil
	}
	s.wg.Add(1)
	go func() {
		foo(s.ctx)
		s.wg.Done()
	}()
	return nil
}

// This calls go foo() directly, with the benefit of being easily searchable
func (s *StopWaiterSafe) LaunchUntrackedThread(foo func()) {
	go foo()
}

func (s *StopWaiterSafe) LaunchWithInterval(foo func(context.Context), interval time.Duration) error {
	return s.LaunchThread(func(ctx context.Context) {
	ThreadMainLoop:
		for {
			foo(ctx)
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				break ThreadMainLoop
			case <-timer.C:
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

func (s *StopWaiter) LaunchWithInterval(foo func(context.Context), interval time.Duration) {
	if err := s.StopWaiterSafe.LaunchWithInterval(foo, interval); err != nil {
		panic(err)
	}
}
