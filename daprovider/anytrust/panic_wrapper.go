// Copyright 2022-2026, Offchain Labs, Inc.
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
	anytrustutil.Writer
}

func NewWriterPanicWrapper(dataAvailabilityService anytrustutil.Writer) anytrustutil.Writer {
	return &WriterPanicWrapper{Writer: dataAvailabilityService}
}
func (w *WriterPanicWrapper) String() string {
	return fmt.Sprintf("WriterPanicWrapper{%v}", w.Writer)
}

func (w *WriterPanicWrapper) Store(ctx context.Context, message []byte, timeout uint64) (*anytrustutil.DataAvailabilityCertificate, error) {
	cert, err := w.Writer.Store(ctx, message, timeout)
	if err != nil {
		panic(fmt.Sprintf("panic wrapper Store: %v", err))
	}
	return cert, nil
}

type ReaderPanicWrapper struct {
	anytrustutil.Reader
}

func NewReaderPanicWrapper(dataAvailabilityService anytrustutil.Reader) anytrustutil.Reader {
	return &ReaderPanicWrapper{
		Reader: dataAvailabilityService,
	}
}
func (w *ReaderPanicWrapper) String() string {
	return fmt.Sprintf("ReaderPanicWrapper{%v}", w.Reader)
}

func (w *ReaderPanicWrapper) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	data, err := w.Reader.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Error("AnyTrust hash lookup failed from cancelled context")
			return nil, err
		}
		panic(fmt.Sprintf("panic wrapper GetByHash: %v", err))
	}
	return data, nil
}
