// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/offchainlabs/nitro/arbstate"
)

type RetryWrapper struct {
	backoffPolicy backoff.BackOff
	DataAvailabilityService
}

func NewRetryWrapper(dataAvailabilityService DataAvailabilityService) DataAvailabilityService {
	backoffPolicy := backoff.NewExponentialBackOff()
	backoffPolicy.InitialInterval = time.Millisecond * 100
	return &RetryWrapper{
		backoffPolicy:           backoffPolicy,
		DataAvailabilityService: dataAvailabilityService,
	}
}

func (w *RetryWrapper) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	var res []byte
	err := backoff.Retry(func() error {
		if ctx.Err() != nil {
			return backoff.Permanent(ctx.Err())
		}
		data, err := w.DataAvailabilityService.GetByHash(ctx, hash)
		if err != nil {
			return err
		}
		res = data
		return nil
	}, w.backoffPolicy)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (w *RetryWrapper) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	var res *arbstate.DataAvailabilityCertificate
	err := backoff.Retry(func() error {
		if ctx.Err() != nil {
			return backoff.Permanent(ctx.Err())
		}
		data, err := w.DataAvailabilityService.Store(ctx, message, timeout, sig)
		if err != nil {
			return err
		}
		res = data
		return nil
	}, w.backoffPolicy)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (w *RetryWrapper) String() string {
	return fmt.Sprintf("RetryWrapper{%v}", w.DataAvailabilityService)
}
