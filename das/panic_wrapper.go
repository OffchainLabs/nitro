// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
)

type WriterPanicWrapper struct {
	DataAvailabilityServiceWriter
}

func NewWriterPanicWrapper(dataAvailabilityService DataAvailabilityServiceWriter) DataAvailabilityServiceWriter {
	return &WriterPanicWrapper{
		DataAvailabilityServiceWriter: dataAvailabilityService,
	}
}
func (w *WriterPanicWrapper) String() string {
	return fmt.Sprintf("WriterPanicWrapper{%v}", w.DataAvailabilityServiceWriter)
}

func (w *WriterPanicWrapper) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*daprovider.DataAvailabilityCertificate, error) {
	cert, err := w.DataAvailabilityServiceWriter.Store(ctx, message, timeout, sig)
	if err != nil {
		panic(fmt.Sprintf("panic wrapper Store: %v", err))
	}
	return cert, nil
}

type ReaderPanicWrapper struct {
	DataAvailabilityServiceReader
}

func NewReaderPanicWrapper(dataAvailabilityService DataAvailabilityServiceReader) DataAvailabilityServiceReader {
	return &ReaderPanicWrapper{
		DataAvailabilityServiceReader: dataAvailabilityService,
	}
}
func (w *ReaderPanicWrapper) String() string {
	return fmt.Sprintf("ReaderPanicWrapper{%v}", w.DataAvailabilityServiceReader)
}

func (w *ReaderPanicWrapper) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	data, err := w.DataAvailabilityServiceReader.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Error("DAS hash lookup failed from cancelled context")
			return nil, err
		}
		panic(fmt.Sprintf("panic wrapper GetByHash: %v", err))
	}
	return data, nil
}
