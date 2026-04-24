// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
)

type CircuitBreakerConfig struct {
	Enabled         bool          `koanf:"enabled"`
	WindowDuration  time.Duration `koanf:"window-duration"`
	MinSamples      uint          `koanf:"min-samples"`
	OpenThreshold   float64       `koanf:"open-threshold"`
	OpenCooldown    time.Duration `koanf:"open-cooldown"`
	HalfOpenTimeout time.Duration `koanf:"half-open-timeout"`
}

var DefaultCircuitBreakerConfig = CircuitBreakerConfig{
	Enabled:         true,
	WindowDuration:  1 * time.Minute,
	MinSamples:      5,
	OpenThreshold:   0.5,
	OpenCooldown:    30 * time.Second,
	HalfOpenTimeout: 2 * time.Minute,
}

func (c *CircuitBreakerConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.WindowDuration <= 0 {
		return fmt.Errorf("circuit-breaker.window-duration must be positive, got %s", c.WindowDuration)
	}
	if c.OpenCooldown <= 0 {
		return fmt.Errorf("circuit-breaker.open-cooldown must be positive, got %s", c.OpenCooldown)
	}
	if c.HalfOpenTimeout <= 0 {
		return fmt.Errorf("circuit-breaker.half-open-timeout must be positive, got %s", c.HalfOpenTimeout)
	}
	if c.OpenThreshold <= 0 || c.OpenThreshold > 1 {
		return fmt.Errorf("circuit-breaker.open-threshold must be in (0, 1], got %f", c.OpenThreshold)
	}
	if c.MinSamples == 0 {
		return fmt.Errorf("circuit-breaker.min-samples must be >= 1 (0 would trip on the first failure)")
	}
	return nil
}

func CircuitBreakerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enabled", DefaultCircuitBreakerConfig.Enabled, "enable the per-forwarder circuit breaker around external-endpoint calls")
	f.Duration(prefix+".window-duration", DefaultCircuitBreakerConfig.WindowDuration, "sliding window used to compute the failure rate in the closed state")
	f.Uint(prefix+".min-samples", DefaultCircuitBreakerConfig.MinSamples, "minimum samples in the window before the breaker can trip")
	f.Float64(prefix+".open-threshold", DefaultCircuitBreakerConfig.OpenThreshold, "failure rate in (0,1] that trips the breaker from closed to open")
	f.Duration(prefix+".open-cooldown", DefaultCircuitBreakerConfig.OpenCooldown, "time to stay in open before a single probe is admitted (half-open)")
	f.Duration(prefix+".half-open-timeout", DefaultCircuitBreakerConfig.HalfOpenTimeout, "safety timeout: if the half-open probe never records a result within this, force the breaker back to open")
}

type breakerState uint32

const (
	stateClosed breakerState = iota
	stateOpen
	stateHalfOpen
)

// atomicBreakerState is a tiny wrapper over atomic.Uint32 that speaks the
// breakerState enum directly, so call sites stay free of uint32 casts.
type atomicBreakerState struct{ v atomic.Uint32 }

func (a *atomicBreakerState) Load() breakerState   { return breakerState(a.v.Load()) }
func (a *atomicBreakerState) Store(s breakerState) { a.v.Store(uint32(s)) }

type breakerSample struct {
	at      time.Time
	success bool
}

type Breaker struct {
	cfg *CircuitBreakerConfig
	now func() time.Time

	// state is the authoritative breaker state. Writers update it under mu;
	// the Closed-state fast path in Allow loads it lock-free so healthy
	// workers never contest the mutex.
	state atomicBreakerState

	mu                sync.Mutex
	samples           []breakerSample
	openedAt          time.Time
	halfOpenTaken     bool
	halfOpenEnteredAt time.Time
}

func NewBreaker(cfg *CircuitBreakerConfig, now func() time.Time) *Breaker {
	if now == nil {
		now = time.Now
	}
	b := &Breaker{cfg: cfg, now: now}
	b.state.Store(stateClosed)
	return b
}

// Allow reports whether a worker may call the endpoint. If it returns true
// in HalfOpen, the caller owns the single probe slot and must call Record.
//
// The Closed-state fast path is a single atomic load. The tradeoff is that
// an Allow racing with a Closed->Open transition may return true for one
// extra call after the transition commits; that call's Record then lands in
// the defensive Open branch and is dropped. No HalfOpen admission invariant
// is at risk because the slow path (which owns HalfOpen) is still fully
// mutex-protected.
func (b *Breaker) Allow() bool {
	if b.state.Load() == stateClosed {
		return true
	}
	return b.allowSlow()
}

func (b *Breaker) allowSlow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	switch b.state.Load() {
	case stateClosed:
		return true
	case stateOpen:
		if now.Sub(b.openedAt) < b.cfg.OpenCooldown {
			return false
		}
		b.transitionToHalfOpenLocked(now)
		b.halfOpenTaken = true
		return true
	case stateHalfOpen:
		if b.halfOpenTaken {
			if now.Sub(b.halfOpenEnteredAt) >= b.cfg.HalfOpenTimeout {
				log.Warn("circuit breaker reopened: half-open probe timed out without reporting", "timeout", b.cfg.HalfOpenTimeout)
				b.transitionToOpenLocked(now)
			}
			return false
		}
		b.halfOpenTaken = true
		return true
	}
	return false
}

// Record feeds the outcome of a call that Allow permitted.
func (b *Breaker) Record(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	switch b.state.Load() {
	case stateClosed:
		b.samples = append(b.samples, breakerSample{at: now, success: success})
		b.pruneLocked(now)
		if uint(len(b.samples)) >= b.cfg.MinSamples && b.failureRateLocked() >= b.cfg.OpenThreshold {
			log.Warn(
				"circuit breaker tripped",
				"failureRate", b.failureRateLocked(),
				"samples", len(b.samples),
				"cooldown", b.cfg.OpenCooldown,
			)
			b.transitionToOpenLocked(now)
		}
	case stateHalfOpen:
		b.halfOpenTaken = false
		if now.Sub(b.halfOpenEnteredAt) >= b.cfg.HalfOpenTimeout {
			// Probe came back after HalfOpenTimeout — the endpoint was too
			// slow. Treat as a failure regardless of what it reported.
			log.Warn("circuit breaker reopened: probe exceeded half-open timeout", "timeout", b.cfg.HalfOpenTimeout)
			b.transitionToOpenLocked(now)
			return
		}
		if success {
			b.transitionToClosedLocked()
		} else {
			log.Warn("circuit breaker reopened: half-open probe failed")
			b.transitionToOpenLocked(now)
		}
	case stateOpen:
		// Possible when Allow raced a Closed->Open transition and returned
		// true on stale state; the race is documented on Allow.
	}
}

func (b *Breaker) pruneLocked(now time.Time) {
	cutoff := now.Add(-b.cfg.WindowDuration)
	i := 0
	for i < len(b.samples) && b.samples[i].at.Before(cutoff) {
		i++
	}
	if i > 0 {
		b.samples = b.samples[i:]
	}
}

func (b *Breaker) failureRateLocked() float64 {
	if len(b.samples) == 0 {
		return 0
	}
	var failures int
	for _, s := range b.samples {
		if !s.success {
			failures++
		}
	}
	return float64(failures) / float64(len(b.samples))
}

func (b *Breaker) transitionToOpenLocked(now time.Time) {
	b.state.Store(stateOpen)
	b.openedAt = now
	b.samples = nil
	b.halfOpenTaken = false
}

func (b *Breaker) transitionToHalfOpenLocked(now time.Time) {
	log.Info("circuit breaker half-open, admitting single probe")
	b.state.Store(stateHalfOpen)
	b.halfOpenEnteredAt = now
}

func (b *Breaker) transitionToClosedLocked() {
	log.Info("circuit breaker closed")
	b.state.Store(stateClosed)
	b.samples = nil
	b.halfOpenTaken = false
}
