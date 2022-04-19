// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"time"

	"github.com/offchainlabs/nitro/arbstate"
)

type DeadlineWrapper struct {
	t time.Duration
	DataAvailabilityService
}

func (w *DeadlineWrapper) Retrieve(ctx context.Context, cert []byte) ([]byte, error) {
	deadlineCtx, _ := context.WithDeadline(ctx, time.Now().Add(w.t))
	return w.DataAvailabilityService.Retrieve(deadlineCtx, cert)

}

func (w *DeadlineWrapper) Store(ctx context.Context, message []byte, timeout uint64) (*arbstate.DataAvailabilityCertificate, error) {
	deadlineCtx, _ := context.WithDeadline(ctx, time.Now().Add(w.t))
	return w.DataAvailabilityService.Store(deadlineCtx, message, timeout)
}
