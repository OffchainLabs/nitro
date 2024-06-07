package types

import (
	"context"
)

func NewWriterForCelestia(celestiaWriter CelestiaWriter) *writerForCelestia {
	return &writerForCelestia{celestiaWriter: celestiaWriter}
}

type writerForCelestia struct {
	celestiaWriter CelestiaWriter
}

func (c *writerForCelestia) Store(ctx context.Context, message []byte, timeout uint64, sig []byte, disableFallbackStoreDataOnChain bool) ([]byte, error) {
	return c.celestiaWriter.Store(ctx, message)
}
