package eigenda

import (
	"context"
)

type DataAvailabilityWriter interface {
	Store(context.Context, []byte) ([]byte, error)
}

type DataAvailabilityReader interface {
	Read(BlobRef) ([]byte, error)
}
