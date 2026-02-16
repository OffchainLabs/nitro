// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"

	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
)

type ReaderTimeoutWrapper struct {
	t time.Duration
	anytrustutil.Reader
}

type TimeoutWrapper struct {
	ReaderTimeoutWrapper
}

func NewReaderTimeoutWrapper(dataAvailabilityServiceReader anytrustutil.Reader, t time.Duration) anytrustutil.Reader {
	return &ReaderTimeoutWrapper{
		t:      t,
		Reader: dataAvailabilityServiceReader,
	}
}

func (w *ReaderTimeoutWrapper) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(w.t))
	// For GetByHash we want fast cancellation of all goroutines started by
	// the aggregator as soon as one returns.
	defer cancel()
	return w.Reader.GetByHash(deadlineCtx, hash)
}

func (w *ReaderTimeoutWrapper) String() string {
	return fmt.Sprintf("ReaderTimeoutWrapper{%v}", w.Reader)
}
