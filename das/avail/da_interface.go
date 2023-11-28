package avail

import (
	"context"
)

type DataAvailabilityWriter interface {
	Store(context.Context, []byte) ([]byte, error)
}

type DataAvailabilityReader interface {
	Read(BlobPointer) ([]byte, error)
}
