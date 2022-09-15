// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package stopwaiter

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

const stopDelayWarningTimeout = 30 * time.Second

type StopWaiterSafe struct {
	mutex    sync.Mutex // protects started, stopped, ctx, stopFunc
	started  bool
	stopped  bool
	ctx      context.Context
	stopFunc func()
	name     string
	waitChan <-chan interface{}

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
	return s.getContext()
}

// Only call this internally with the mutex held.
func (s *StopWaiterSafe) getContext() (context.Context, error) {
	if s.started {
		return s.ctx, nil
	}
	return nil, errors.New("not started")

}

func getParentName(parent any) string {
	// remove asterisk in case the type is a pointer
	return strings.Replace(reflect.TypeOf(parent).String(), "*", "", 1)
}

// start-after-start will error, start-after-stop will immediately cancel
func (s *StopWaiterSafe) Start(ctx context.Context, parent any) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started {
		return errors.New("start after start")
	}
	s.started = true
	s.name = getParentName(parent)
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
	s.StopOnly()
	timer := time.NewTimer(warningTimeout)
	waitChan, err := s.GetWaitChannel()
	if err != nil {
		return err
	}

	select {
	case <-timer.C:
		traces := getAllStackTraces()
		log.Warn("taking too long to stop", "name", s.name, "delay[s]", warningTimeout.Seconds())
		log.Warn(traces)
	case <-waitChan:
		return nil
	}
	<-waitChan
	return nil
}

func (s *StopWaiterSafe) GetWaitChannel() (<-chan interface{}, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.waitChan == nil {
		ctx, err := s.getContext()
		if err != nil {
			return nil, err
		}
		waitChan := make(chan interface{})
		go func() {
			<-ctx.Done()
			s.wg.Wait()
			close(waitChan)
		}()
		s.waitChan = waitChan
	}
	return s.waitChan, nil
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
			if ctx.Err() != nil {
				return
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

// May panic on race conditions instead of returning errors
type StopWaiter struct {
	StopWaiterSafe
}

func (s *StopWaiter) Start(ctx context.Context, parent any) {
	if err := s.StopWaiterSafe.Start(ctx, parent); err != nil {
		panic(err)
	}
}

func (s *StopWaiter) StopAndWait() {
	if err := s.StopWaiterSafe.StopAndWait(); err != nil {
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
