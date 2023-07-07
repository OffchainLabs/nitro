// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE

package retry

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

const sleepTime = time.Second * 1

var (
	retryCounter = metrics.NewRegisteredCounter("arb/validator/runtime/retry", nil)
	pkglog       = log.New("package", "retry")
)

// UntilSucceeds retries the given function until it succeeds or the context is cancelled.
func UntilSucceeds[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	count := 0
	for {
		if ctx.Err() != nil {
			return zeroVal[T](), ctx.Err()
		}
		got, err := fn()
		if err != nil {
			count++
			pkglog.Error("Failed to call function after retries", log.Ctx{
				"retryCount": count,
			})
			retryCounter.Inc(1)
			time.Sleep(sleepTime)
			continue
		}
		return got, nil
	}
}

func zeroVal[T any]() T {
	var result T
	return result
}
