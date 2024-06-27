// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package daprovider

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/log"
)

type Writer interface {
	// Store posts the batch data to the invoking DA provider
	// And returns sequencerMsg which is later used to retrieve the batch data
	Store(
		ctx context.Context,
		message []byte,
		timeout uint64,
		disableFallbackStoreDataOnChain bool,
	) ([]byte, error)
}

// DAProviderWriterForDAS is generally meant to be only used by nitro.
// DA Providers should implement methods in the DAProviderWriter interface independently
func NewWriterForDAS(dasWriter DASWriter) *writerForDAS {
	return &writerForDAS{dasWriter: dasWriter}
}

type writerForDAS struct {
	dasWriter DASWriter
}

func (d *writerForDAS) Store(ctx context.Context, message []byte, timeout uint64, disableFallbackStoreDataOnChain bool) ([]byte, error) {
	cert, err := d.dasWriter.Store(ctx, message, timeout)
	if errors.Is(err, ErrBatchToDasFailed) {
		if disableFallbackStoreDataOnChain {
			return nil, errors.New("unable to batch to DAS and fallback storing data on chain is disabled")
		}
		log.Warn("Falling back to storing data on chain", "err", err)
		return message, nil
	} else if err != nil {
		return nil, err
	} else {
		return Serialize(cert), nil
	}
}
