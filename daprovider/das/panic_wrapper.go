// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
)

type WriterPanicWrapper[WriteResult any] struct {
	inner dasutil.GenericDASWriter[WriteResult]
}

func NewWriterPanicWrapper[WriteResult any](inner dasutil.GenericDASWriter[WriteResult]) dasutil.GenericDASWriter[WriteResult] {
	return &WriterPanicWrapper[WriteResult]{inner}
}
func (w *WriterPanicWrapper[WriteResult]) String() string {
	return fmt.Sprintf("WriterPanicWrapper{%v}", w.inner)
}

func (w *WriterPanicWrapper[WriteResult]) Store(ctx context.Context, message []byte, timeout uint64) (WriteResult, error) {
	cert, err := w.inner.Store(ctx, message, timeout)
	if err != nil {
		panic(fmt.Sprintf("panic wrapper Store: %v", err))
	}
	return cert, nil
}

type ReaderPanicWrapper struct {
	dasutil.DASReader
}

func NewReaderPanicWrapper(dataAvailabilityService dasutil.DASReader) dasutil.DASReader {
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
