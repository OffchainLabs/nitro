// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package runtime defines utilities that deal with managing lifecycles of
// functions and important behaviors at the application runtime, such as
// retrying errored functions until they succeed.
package retry

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/bold/logs/ephemeral"
)

const defaultSleepTime = time.Second * 30

var (
	retryCounter = metrics.NewRegisteredCounter("arb/validator/runtime/retry", nil)
)

type RetryConfig struct {
	sleepTime         time.Duration
	LevelWarningError string // can be extended to a list or regex if demanded in future, currently supporting for one error
	LevelInfoError    string // can be extended to a list or regex if demanded in future, currently supporting for one error
}

type Opt func(*RetryConfig)

// WithInterval specifies how often to retry an errored function.
func WithInterval(d time.Duration) Opt {
	return func(rc *RetryConfig) {
		rc.sleepTime = d
	}
}

func UntilSucceedsMultipleReturnValue[T, U any](ctx context.Context, fn func() (T, U, error), opts ...Opt) (T, U, error) {
	cfg := &RetryConfig{
		sleepTime: defaultSleepTime,
	}
	for _, o := range opts {
		o(cfg)
	}
	count := 0
	// Retry until succeeds is usually used for cases where its believed that retrying a function will most likely succeed
	// or the function has a some chance of failing even if it's not expected to fail, based on this assumption,
	// we use a commonEphemeralErrorHandler to log the errors at warn level for the first 10 minutes
	// and only after that we log the errors at error level.
	commonEphemeralErrorHandler := ephemeral.NewEphemeralErrorHandler(time.Minute*10, "", 0)
	for {
		if ctx.Err() != nil {
			return zeroVal[T](), zeroVal[U](), ctx.Err()
		}
		got, got2, err := fn()
		if err != nil {
			count++
			logLevel := log.Error
			logLevel = commonEphemeralErrorHandler.LogLevel(err, logLevel)
			if cfg.LevelWarningError != "" && strings.Contains(err.Error(), cfg.LevelWarningError) {
				logLevel = log.Warn
			}
			if cfg.LevelInfoError != "" && strings.Contains(err.Error(), cfg.LevelInfoError) {
				logLevel = log.Info
			}
			logLevel("Could not succeed function after retries",
				"retryCount", count,
				"err", err,
			)
			retryCounter.Inc(1)
			select {
			case <-ctx.Done():
				return zeroVal[T](), zeroVal[U](), ctx.Err()
			case <-time.After(cfg.sleepTime):
			}
			continue
		}
		commonEphemeralErrorHandler.Reset()
		return got, got2, nil
	}
}

// UntilSucceeds retries the given function until it succeeds or the context is cancelled.
func UntilSucceeds[T any](ctx context.Context, fn func() (T, error), opts ...Opt) (T, error) {
	result, _, err := UntilSucceedsMultipleReturnValue(ctx, func() (T, struct{}, error) {
		got, err := fn()
		return got, struct{}{}, err
	}, opts...)
	return result, err
}

func zeroVal[T any]() T {
	var result T
	return result
}
