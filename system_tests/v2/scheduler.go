// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// Scheduler manages weighted concurrency for the test runner.
// It provides context-aware acquisition and diagnostic logging
// when tests are blocked waiting for capacity.
type Scheduler struct {
	mu       sync.Mutex
	cond     *sync.Cond
	capacity int
	used     int
	running  map[string]int // test name -> weight
	waiting  map[string]int // test name -> weight
	t        testing.TB
}

// NewScheduler creates a scheduler with the given total capacity.
func NewScheduler(capacity int, t testing.TB) *Scheduler {
	s := &Scheduler{
		capacity: capacity,
		running:  make(map[string]int),
		waiting:  make(map[string]int),
		t:        t,
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

// Acquire blocks until n slots of capacity are available.
// Returns an error if the context is cancelled while waiting.
// If blocked for more than 30s, logs diagnostic info about what's running.
func (s *Scheduler) Acquire(ctx context.Context, n int, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fast path: capacity available immediately.
	if s.used+n <= s.capacity {
		s.used += n
		s.running[name] = n
		return nil
	}

	// Record that we're waiting.
	s.waiting[name] = n

	// Wake on context cancellation.
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			s.cond.Broadcast()
		case <-done:
		}
	}()
	defer close(done)

	// Diagnostic: log if blocked for too long.
	warnTimer := time.AfterFunc(30*time.Second, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.logStatus(name, n)
	})
	defer warnTimer.Stop()

	for s.used+n > s.capacity {
		if ctx.Err() != nil {
			delete(s.waiting, name)
			return ctx.Err()
		}
		s.cond.Wait()
	}

	delete(s.waiting, name)
	s.used += n
	s.running[name] = n
	return nil
}

// Release frees n slots of capacity.
func (s *Scheduler) Release(n int, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.used -= n
	delete(s.running, name)
	s.cond.Broadcast()
}

// logStatus logs what's currently running and waiting (must be called with mu held).
func (s *Scheduler) logStatus(blockedName string, blockedWeight int) {
	var running []string
	for name, w := range s.running {
		running = append(running, fmt.Sprintf("%s(w=%d)", name, w))
	}
	var waiting []string
	for name, w := range s.waiting {
		waiting = append(waiting, fmt.Sprintf("%s(w=%d)", name, w))
	}
	s.t.Logf("SCHEDULER: %q (w=%d) blocked >30s. capacity=%d, used=%d\n  running: [%s]\n  waiting: [%s]",
		blockedName, blockedWeight, s.capacity, s.used,
		strings.Join(running, ", "),
		strings.Join(waiting, ", "),
	)
}
