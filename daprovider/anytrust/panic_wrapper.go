// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
)

type WriterPanicWrapper struct {
	anytrustutil.DASWriter
}

func NewWriterPanicWrapper(dataAvailabilityService anytrustutil.DASWriter) anytrustutil.DASWriter {
	return &WriterPanicWrapper{DASWriter: dataAvailabilityService}
}
func (w *WriterPanicWrapper) String() string {
	return fmt.Sprintf("WriterPanicWrapper{%v}", w.DASWriter)
}

func (w *WriterPanicWrapper) Store(ctx context.Context, message []byte, timeout uint64) (*anytrustutil.DataAvailabilityCertificate, error) {
	cert, err := w.DASWriter.Store(ctx, message, timeout)
	if err != nil {
		panic(fmt.Sprintf("panic wrapper Store: %v", err))
	}
	return cert, nil
}

type ReaderPanicWrapper struct {
	anytrustutil.DASReader
}

func NewReaderPanicWrapper(dataAvailabilityService anytrustutil.DASReader) anytrustutil.DASReader {
	return &ReaderPanicWrapper{
		DASReader: dataAvailabilityService,
	}
}
func (w *ReaderPanicWrapper) String() string {
	return fmt.Sprintf("ReaderPanicWrapper{%v}", w.DASReader)
}

func (w *ReaderPanicWrapper) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	data, err := w.DASReader.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Error("DAS hash lookup failed from cancelled context")
			return nil, err
		}
		panic(fmt.Sprintf("panic wrapper GetByHash: %v", err))
	}
	return data, nil
}
