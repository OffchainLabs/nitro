// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"context"
	"testing"
	"time"
)

// PollConfig controls the polling behavior of WaitFor and related helpers.
type PollConfig struct {
	InitialInterval time.Duration // default 50ms
	MaxInterval     time.Duration // default 2s
	Multiplier      float64       // default 1.5
	Timeout         time.Duration // default 30s
}

var defaultPollConfig = PollConfig{
	InitialInterval: 50 * time.Millisecond,
	MaxInterval:     2 * time.Second,
	Multiplier:      1.5,
	Timeout:         30 * time.Second,
}

func mergePollConfig(opts []PollConfig) PollConfig {
	cfg := defaultPollConfig
	if len(opts) > 0 {
		o := opts[0]
		if o.InitialInterval > 0 {
			cfg.InitialInterval = o.InitialInterval
		}
		if o.MaxInterval > 0 {
			cfg.MaxInterval = o.MaxInterval
		}
		if o.Multiplier > 0 {
			cfg.Multiplier = o.Multiplier
		}
		if o.Timeout > 0 {
			cfg.Timeout = o.Timeout
		}
	}
	return cfg
}

// WaitFor polls check() with exponential backoff until it returns true.
// Fails the test if the context is cancelled or the timeout expires.
// desc is included in the failure message for debuggability.
func WaitFor(ctx context.Context, t testing.TB, desc string, check func() bool, opts ...PollConfig) {
	t.Helper()
	cfg := mergePollConfig(opts)
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	interval := cfg.InitialInterval
	for {
		if check() {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("WaitFor %q timed out after %s", desc, cfg.Timeout)
		case <-time.After(interval):
		}
		interval = min(time.Duration(float64(interval)*cfg.Multiplier), cfg.MaxInterval)
	}
}

// WaitForE polls check() with exponential backoff until it returns (true, nil).
// If check returns a non-nil error, the test fails immediately (no retry).
// If check returns (false, nil), it retries with backoff.
func WaitForE(ctx context.Context, t testing.TB, desc string, check func() (bool, error), opts ...PollConfig) {
	t.Helper()
	cfg := mergePollConfig(opts)
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	interval := cfg.InitialInterval
	for {
		done, err := check()
		if err != nil {
			t.Fatalf("WaitForE %q failed: %v", desc, err)
		}
		if done {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("WaitForE %q timed out after %s", desc, cfg.Timeout)
		case <-time.After(interval):
		}
		interval = min(time.Duration(float64(interval)*cfg.Multiplier), cfg.MaxInterval)
	}
}

// WaitForValue polls get() with exponential backoff until it returns a non-zero value.
// Returns the first non-zero value.
func WaitForValue[T comparable](ctx context.Context, t testing.TB, desc string, get func() T, opts ...PollConfig) T {
	t.Helper()
	cfg := mergePollConfig(opts)
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	var zero T
	interval := cfg.InitialInterval
	for {
		v := get()
		if v != zero {
			return v
		}
		select {
		case <-ctx.Done():
			t.Fatalf("WaitForValue %q timed out after %s", desc, cfg.Timeout)
		case <-time.After(interval):
		}
		interval = min(time.Duration(float64(interval)*cfg.Multiplier), cfg.MaxInterval)
	}
}
