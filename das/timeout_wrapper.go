// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/offchainlabs/nitro/arbstate"
)

type TimeoutWrapper struct {
	t time.Duration
	DataAvailabilityService
}

func (w *TimeoutWrapper) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(w.t))
	// For Retrieve we want fast cancellation of all goroutines started by
	// the aggregator as soon as one returns.
	defer cancel()
	return w.DataAvailabilityService.GetByHash(deadlineCtx, hash)
}

func (w *TimeoutWrapper) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(w.t))
	// In the case of the aggregator, allow goroutines started by Store(...)
	// to continue until the end of the deadline even after a result
	// has been returned.
	go func() {
		<-deadlineCtx.Done()
		cancel()
	}()
	return w.DataAvailabilityService.Store(deadlineCtx, message, timeout, sig)
}

func (w *TimeoutWrapper) String() string {
	return fmt.Sprintf("TimeoutWrapper{%v}", w.DataAvailabilityService)
}
