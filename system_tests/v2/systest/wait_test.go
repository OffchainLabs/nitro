// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackoffUntilSucceedsImmediately(t *testing.T) {
	calls := atomic.Int64{}
	err := defaultBackoff.until(t.Context(), func() (bool, error) {
		calls.Add(1)
		return true, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 call, got %d", got)
	}
}

func TestBackoffUntilReturnsFnError(t *testing.T) {
	sentinel := errors.New("boom")
	calls := atomic.Int64{}
	err := defaultBackoff.until(t.Context(), func() (bool, error) {
		calls.Add(1)
		return false, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v, want %v", err, sentinel)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("fn must not retry after error; got %d calls", got)
	}
}

func TestBackoffUntilReturnsCtxErrOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	b := backoff{initial: time.Millisecond, max: time.Millisecond, factor: 1.5}
	err := b.until(ctx, func() (bool, error) { return false, nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}
