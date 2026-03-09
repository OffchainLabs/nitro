// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package ctxhelper

import (
	"context"
	"time"
)

// WithTimeout is like context.WithTimeout except a timeout of 0 means unlimited instead of instantly expired.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == time.Duration(0) {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}
