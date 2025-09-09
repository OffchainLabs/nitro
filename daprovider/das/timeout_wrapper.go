// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
)

type ReaderTimeoutWrapper struct {
	t time.Duration
	dasutil.DASReader
}

type TimeoutWrapper struct {
	ReaderTimeoutWrapper
}

func NewReaderTimeoutWrapper(dataAvailabilityServiceReader dasutil.DASReader, t time.Duration) dasutil.DASReader {
	return &ReaderTimeoutWrapper{
		t:         t,
		DASReader: dataAvailabilityServiceReader,
	}
}

func (w *ReaderTimeoutWrapper) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(w.t))
	// For GetByHash we want fast cancellation of all goroutines started by
	// the aggregator as soon as one returns.
	defer cancel()
	return w.DASReader.GetByHash(deadlineCtx, hash)
}

func (w *ReaderTimeoutWrapper) String() string {
	return fmt.Sprintf("ReaderTimeoutWrapper{%v}", w.DASReader)
}
