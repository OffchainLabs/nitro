// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbstate"
)

type ReaderTimeoutWrapper struct {
	t time.Duration
	DataAvailabilityServiceReader
}
type WriterTimeoutWrapper struct {
	t time.Duration
	DataAvailabilityServiceWriter
}

type TimeoutWrapper struct {
	ReaderTimeoutWrapper
	WriterTimeoutWrapper
}

func NewReaderTimeoutWrapper(dataAvailabilityServiceReader DataAvailabilityServiceReader, t time.Duration) DataAvailabilityServiceReader {
	return &ReaderTimeoutWrapper{
		t:                             t,
		DataAvailabilityServiceReader: dataAvailabilityServiceReader,
	}
}
func NewWriterTimeoutWrapper(dataAvailabilityServiceWriter DataAvailabilityServiceWriter, t time.Duration) DataAvailabilityServiceWriter {
	return &WriterTimeoutWrapper{
		t:                             t,
		DataAvailabilityServiceWriter: dataAvailabilityServiceWriter,
	}
}

func (w *ReaderTimeoutWrapper) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(w.t))
	// For Retrieve we want fast cancellation of all goroutines started by
	// the aggregator as soon as one returns.
	defer cancel()
	return w.DataAvailabilityServiceReader.GetByHash(deadlineCtx, hash)
}

func (w *WriterTimeoutWrapper) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(w.t))
	// In the case of the aggregator, allow goroutines started by Store(...)
	// to continue until the end of the deadline even after a result
	// has been returned.
	go func() {
		<-deadlineCtx.Done()
		cancel()
	}()
	return w.DataAvailabilityServiceWriter.Store(deadlineCtx, message, timeout, sig)
}

func (w *ReaderTimeoutWrapper) String() string {
	return fmt.Sprintf("ReaderTimeoutWrapper{%v}", w.DataAvailabilityServiceReader)
}

func (w *WriterTimeoutWrapper) String() string {
	return fmt.Sprintf("WriterTimeoutWrapper{%v}", w.DataAvailabilityServiceWriter)
}
