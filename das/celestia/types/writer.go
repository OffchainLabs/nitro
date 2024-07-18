package types

import (
	"context"
	"errors"
)

func NewWriterForCelestia(celestiaWriter CelestiaWriter) *writerForCelestia {
	return &writerForCelestia{celestiaWriter: celestiaWriter}
}

type writerForCelestia struct {
	celestiaWriter CelestiaWriter
}

func (c *writerForCelestia) Store(ctx context.Context, message []byte, timeout uint64, disableFallbackStoreDataOnChain bool) ([]byte, error) {
	msg, err := c.celestiaWriter.Store(ctx, message)
	if err != nil {
		if disableFallbackStoreDataOnChain {
			return nil, errors.New("unable to batch to Celestia and fallback storing data on chain is disabled")
		}
		return nil, err
	}
	message = msg
	return message, nil
}
