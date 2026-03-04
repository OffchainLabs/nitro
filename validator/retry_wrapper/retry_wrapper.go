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
// allowed attempts to retry in case of failure. Timeout errors have their own
// separate counter (allowedTimeouts) since they typically represent transient
// conditions (e.g., CPU starvation, slow validation) rather than genuine
// validation failures.
func (v *ValidationSpawnerRetryWrapper) LaunchWithNAllowedAttempts(entry *validator.ValidationInput, moduleRoot common.Hash, allowedAttempts uint64, allowedTimeouts uint64) validator.ValidationRun {
	promise := stopwaiter.LaunchPromiseThread(v, func(ctx context.Context) (validator.GoGlobalState, error) {
		nonTimeoutAttempts := uint64(0)
		timeoutAttempts := uint64(0)
		for {
			res, err := v.ValidationSpawner.Launch(entry, moduleRoot).Await(ctx)
			if err == nil {
				return res, nil
			}
			if ctx.Err() != nil {
				return validator.GoGlobalState{}, ctx.Err()
			}
			if validator.IsTimeoutError(err) {
				timeoutAttempts++
				if timeoutAttempts >= allowedTimeouts {
					return validator.GoGlobalState{}, err
				}
				log.Warn("validation attempt timed out, retrying",
					"err", err,
					"moduleRoot", moduleRoot,
					"timeoutAttempt", timeoutAttempts,
					"allowedTimeouts", allowedTimeouts,
				)
				continue
			}
			nonTimeoutAttempts++
			if nonTimeoutAttempts >= allowedAttempts {
				return validator.GoGlobalState{}, err
			}
		}
	})
	return server_common.NewValRun(promise, moduleRoot)
}
