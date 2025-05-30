package retry_wrapper

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

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

// LaunchWithNAllowedAttempts launches the validation with a specified number of allowed attempts to retry in case of failure.
func (v *ValidationSpawnerRetryWrapper) LaunchWithNAllowedAttempts(entry *validator.ValidationInput, moduleRoot common.Hash, allowedAttempts uint64) validator.ValidationRun {
	promise := stopwaiter.LaunchPromiseThread(v, func(ctx context.Context) (validator.GoGlobalState, error) {
		totalAttempts := uint64(0)
		for {
			res, err := v.ValidationSpawner.Launch(entry, moduleRoot).Await(ctx)
			totalAttempts++
			// If the attempt is successful, return immediately
			if err == nil {
				return res, nil
			}
			// If we have reached the retry limit, return the error
			if totalAttempts >= allowedAttempts {
				return validator.GoGlobalState{}, err
			}
			// If the context is done, return the error
			if ctx.Err() != nil {
				return validator.GoGlobalState{}, ctx.Err()
			}
		}
	})
	return server_common.NewValRun(promise, moduleRoot)
}
