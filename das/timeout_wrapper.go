// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type ReaderTimeoutWrapper struct {
	t time.Duration
	DataAvailabilityServiceReader
}

type TimeoutWrapper struct {
	ReaderTimeoutWrapper
}

func NewReaderTimeoutWrapper(dataAvailabilityServiceReader DataAvailabilityServiceReader, t time.Duration) DataAvailabilityServiceReader {
	return &ReaderTimeoutWrapper{
		t:                             t,
		DataAvailabilityServiceReader: dataAvailabilityServiceReader,
	}
}

func (w *ReaderTimeoutWrapper) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(w.t))
	// For GetByHash we want fast cancellation of all goroutines started by
	// the aggregator as soon as one returns.
	defer cancel()
	return w.DataAvailabilityServiceReader.GetByHash(deadlineCtx, hash)
}

func (w *ReaderTimeoutWrapper) String() string {
	return fmt.Sprintf("ReaderTimeoutWrapper{%v}", w.DataAvailabilityServiceReader)
}
