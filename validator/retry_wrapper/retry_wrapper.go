// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package retry_wrapper

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_common"
)

type ValidationSpawnerRetryWrapper struct {
	stopwaiter.StopWaiter
	validator.ValidationSpawner
}

func NewValidationSpawnerRetryWrapper(spawner validator.ValidationSpawner) *ValidationSpawnerRetryWrapper {
	return &ValidationSpawnerRetryWrapper{
		ValidationSpawner: spawner,
	}
}

// LaunchWithNAllowedAttempts launches the validation with a specified number of
// allowed attempts to retry in case of failure. Timeout errors are retried
// indefinitely and do not count against the retry limit, since they represent
// transient conditions (e.g., CPU starvation, slow validation) rather than
// genuine validation failures.
func (v *ValidationSpawnerRetryWrapper) LaunchWithNAllowedAttempts(entry *validator.ValidationInput, moduleRoot common.Hash, allowedAttempts uint64) validator.ValidationRun {
	promise := stopwaiter.LaunchPromiseThread(v, func(ctx context.Context) (validator.GoGlobalState, error) {
		nonTimeoutAttempts := uint64(0)
		for {
			res, err := v.ValidationSpawner.Launch(entry, moduleRoot).Await(ctx)
			// If the attempt is successful, return immediately
			if err == nil {
				return res, nil
			}
			// If the context is done, return the error
			if ctx.Err() != nil {
				return validator.GoGlobalState{}, ctx.Err()
			}
			// Timeout errors are retried indefinitely without counting against the retry limit.
			if validator.IsTimeoutError(err) {
				log.Warn("validation attempt timed out, retrying",
					"err", err,
					"moduleRoot", moduleRoot,
				)
				continue
			}
			// Non-timeout error: count against retry limit
			nonTimeoutAttempts++
			if nonTimeoutAttempts >= allowedAttempts {
				return validator.GoGlobalState{}, err
			}
		}
	})
	return server_common.NewValRun(promise, moduleRoot)
}
