// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"fmt"
	"time"
)

// backoff parameterises exponential-backoff polling.
type backoff struct {
	initial time.Duration
	max     time.Duration
	factor  float64
}

var defaultBackoff = backoff{
	initial: 50 * time.Millisecond,
	max:     2 * time.Second,
	factor:  1.5,
}

// until polls fn until done=true, fn errs, or ctx cancels. fn's error is
// treated as permanent and returned immediately. done=false + err=nil means
// "not ready yet; keep polling".
func (b backoff) until(ctx context.Context, fn func() (done bool, err error)) error {
	delay := b.initial
	for {
		done, err := fn()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay = time.Duration(float64(delay) * b.factor)
		if delay > b.max {
			delay = b.max
		}
	}
}

// waitFor polls fn until true or ctx cancels. desc names the condition for
// the failure message.
func waitFor(ctx context.Context, desc string, fn func() bool) error {
	err := defaultBackoff.until(ctx, func() (bool, error) {
		return fn(), nil
	})
	if err != nil {
		return fmt.Errorf("waiting for %s: %w", desc, err)
	}
	return nil
}
