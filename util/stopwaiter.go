package util

import (
	"context"
	"sync"
)

type ThreadTracker struct {
	stopFunc func()
	wg       sync.WaitGroup
}

func NewThreadTracker(stopFunc func()) *ThreadTracker {
	return &ThreadTracker{
		stopFunc: stopFunc,
	}
}

// StopAndWait will only wait for thread launced with this function
func (s *ThreadTracker) LaunchThread(foo func()) {
	s.wg.Add(1)
	go func() {
		foo()
		s.wg.Done()
	}()
}

func (s *ThreadTracker) StopAndWait() {
	s.stopFunc()
	s.wg.Wait()
}

type StopWaiter struct {
	ThreadTracker *ThreadTracker
}

func (s *StopWaiter) Start(ctx context.Context) context.Context {
	wrapped, cancelfunc := context.WithCancel(ctx)
	s.ThreadTracker = NewThreadTracker(cancelfunc)
	return wrapped
}

// Not thread safe vs Start or itself
func (s *StopWaiter) StopAndWait() {
	if s.ThreadTracker != nil {
		s.ThreadTracker.StopAndWait()
	}
}
