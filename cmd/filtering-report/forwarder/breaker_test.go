// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Unix(1_700_000_000, 0)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestBreaker(cfg CircuitBreakerConfig) (*Breaker, *fakeClock) {
	clock := newFakeClock()
	return NewBreaker(&cfg, clock.Now), clock
}

func defaultTestBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Enabled:         true,
		WindowDuration:  1 * time.Minute,
		MinSamples:      4,
		OpenThreshold:   0.5,
		OpenCooldown:    30 * time.Second,
		HalfOpenTimeout: 2 * time.Minute,
	}
}

func TestBreaker_TripsOnFailureRate(t *testing.T) {
	b, _ := newTestBreaker(defaultTestBreakerConfig())
	for i := 0; i < 2; i++ {
		if !b.Allow() {
			t.Fatalf("allow should be true while Closed")
		}
		b.Record(true)
	}
	for i := 0; i < 2; i++ {
		if !b.Allow() {
			t.Fatalf("allow should still be true before trip")
		}
		b.Record(false)
	}
	// 2 failures out of 4 samples = 0.5 failure rate, >= OpenThreshold.
	if b.Allow() {
		t.Fatalf("breaker should be Open after failure rate met")
	}
}

func TestBreaker_StaysClosedBelowMinSamples(t *testing.T) {
	cfg := defaultTestBreakerConfig()
	cfg.MinSamples = 10
	b, _ := newTestBreaker(cfg)
	for i := 0; i < 5; i++ {
		if !b.Allow() {
			t.Fatalf("allow should be true below MinSamples")
		}
		b.Record(false)
	}
	if !b.Allow() {
		t.Fatalf("breaker should still be Closed below MinSamples")
	}
}

func TestBreaker_PrunesOldSamples(t *testing.T) {
	cfg := defaultTestBreakerConfig()
	cfg.MinSamples = 4
	cfg.WindowDuration = 10 * time.Second
	b, clock := newTestBreaker(cfg)

	// Record 3 failures one second apart, then age them out of the window.
	for i := 0; i < 3; i++ {
		b.Record(false)
		clock.Advance(time.Second)
	}
	clock.Advance(20 * time.Second) // all three are now outside WindowDuration

	// Three fresh failures: without pruning we'd be at 6 samples and trip,
	// with pruning we have 3, still below MinSamples=4.
	for i := 0; i < 3; i++ {
		b.Record(false)
		clock.Advance(time.Second)
	}
	if !b.Allow() {
		t.Fatalf("expected Closed: pruned samples should keep us under MinSamples")
	}
	// Fourth in-window failure puts us at the threshold and trips.
	b.Record(false)
	if b.Allow() {
		t.Fatalf("expected Open after 4th in-window failure")
	}
}

func TestBreaker_OpenCooldownGatesHalfOpen(t *testing.T) {
	b, clock := newTestBreaker(defaultTestBreakerConfig())
	for i := 0; i < 4; i++ {
		b.Record(false)
	}
	if b.Allow() {
		t.Fatalf("breaker should be Open immediately after tripping")
	}
	clock.Advance(10 * time.Second)
	if b.Allow() {
		t.Fatalf("breaker should still be Open before cooldown elapses")
	}
	clock.Advance(30 * time.Second) // past cooldown
	if !b.Allow() {
		t.Fatalf("expected HalfOpen admission after cooldown")
	}
}

func TestBreaker_HalfOpenAdmitsOnlyOne(t *testing.T) {
	cfg := defaultTestBreakerConfig()
	b, clock := newTestBreaker(cfg)
	for i := 0; i < 4; i++ {
		b.Record(false)
	}
	clock.Advance(cfg.OpenCooldown)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		admitted int
	)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if b.Allow() {
				mu.Lock()
				admitted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if admitted != 1 {
		t.Fatalf("expected exactly one HalfOpen admission, got %d", admitted)
	}
}

func TestBreaker_HalfOpenSuccessCloses(t *testing.T) {
	cfg := defaultTestBreakerConfig()
	b, clock := newTestBreaker(cfg)
	for i := 0; i < 4; i++ {
		b.Record(false)
	}
	clock.Advance(cfg.OpenCooldown)
	if !b.Allow() {
		t.Fatalf("expected HalfOpen admission")
	}
	b.Record(true)
	// Back to Closed: every subsequent Allow should succeed.
	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatalf("expected Closed after HalfOpen success, iteration %d", i)
		}
		b.Record(true)
	}
}

func TestBreaker_HalfOpenFailureReopens(t *testing.T) {
	cfg := defaultTestBreakerConfig()
	b, clock := newTestBreaker(cfg)
	for i := 0; i < 4; i++ {
		b.Record(false)
	}
	clock.Advance(cfg.OpenCooldown)
	if !b.Allow() {
		t.Fatalf("expected HalfOpen admission")
	}
	b.Record(false)
	if b.Allow() {
		t.Fatalf("breaker should be Open again after HalfOpen failure")
	}
	clock.Advance(cfg.OpenCooldown)
	if !b.Allow() {
		t.Fatalf("expected HalfOpen admission after second cooldown")
	}
}

func TestBreaker_HalfOpenTimeoutForcesReopen(t *testing.T) {
	cfg := defaultTestBreakerConfig()
	b, clock := newTestBreaker(cfg)
	for i := 0; i < 4; i++ {
		b.Record(false)
	}
	clock.Advance(cfg.OpenCooldown)
	if !b.Allow() {
		t.Fatalf("expected HalfOpen admission")
	}
	// The probing worker never records. After HalfOpenTimeout, another
	// Allow call should force the breaker back to Open and return false.
	clock.Advance(cfg.HalfOpenTimeout)
	if b.Allow() {
		t.Fatalf("should not admit while forcing reopen")
	}
	// Still in Open, waiting out a fresh cooldown.
	if b.Allow() {
		t.Fatalf("breaker should be Open after stuck-HalfOpen timeout")
	}
	clock.Advance(cfg.OpenCooldown)
	if !b.Allow() {
		t.Fatalf("expected HalfOpen admission after fresh cooldown")
	}
}

// A late HalfOpen probe that finally records after HalfOpenTimeout must
// force Open even when it's reporting success — the endpoint was too slow
// to trust.
func TestBreaker_RecordHalfOpenTimeoutReopens(t *testing.T) {
	cfg := defaultTestBreakerConfig()
	b, clock := newTestBreaker(cfg)
	for i := 0; i < 4; i++ {
		b.Record(false)
	}
	clock.Advance(cfg.OpenCooldown)
	if !b.Allow() {
		t.Fatalf("expected HalfOpen admission")
	}

	clock.Advance(cfg.HalfOpenTimeout + time.Second)
	b.Record(true) // "success", but after the timeout window

	if b.Allow() {
		t.Fatalf("late HalfOpen record should force Open, not Closed")
	}
}

// Pins the documented Allow-vs-Record race: if a worker wins Allow on the
// Closed fast path and records after a concurrent transition to Open, the
// sample is dropped instead of polluting Open state.
func TestBreaker_RecordInStateOpenIsDropped(t *testing.T) {
	b, clock := newTestBreaker(defaultTestBreakerConfig())
	// Simulate the race outcome directly: state already flipped to Open.
	b.state.Store(stateOpen)
	b.openedAt = clock.Now()

	b.Record(true)
	b.Record(false)

	if b.Allow() {
		t.Fatalf("Record in Open should not clear the Open state")
	}
}

func TestCircuitBreakerConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(c *CircuitBreakerConfig)
		wantErr string
	}{
		{"disabled short-circuits", func(c *CircuitBreakerConfig) { c.Enabled = false; c.WindowDuration = 0 }, ""},
		{"zero window", func(c *CircuitBreakerConfig) { c.WindowDuration = 0 }, "window-duration"},
		{"zero cooldown", func(c *CircuitBreakerConfig) { c.OpenCooldown = 0 }, "open-cooldown"},
		{"zero half-open timeout", func(c *CircuitBreakerConfig) { c.HalfOpenTimeout = 0 }, "half-open-timeout"},
		{"zero open threshold", func(c *CircuitBreakerConfig) { c.OpenThreshold = 0 }, "open-threshold"},
		{"too-large open threshold", func(c *CircuitBreakerConfig) { c.OpenThreshold = 1.1 }, "open-threshold"},
		{"zero min samples", func(c *CircuitBreakerConfig) { c.MinSamples = 0 }, "min-samples"},
		{"valid", func(c *CircuitBreakerConfig) {}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultCircuitBreakerConfig
			tc.mutate(&cfg)
			err := cfg.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected valid config, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}
