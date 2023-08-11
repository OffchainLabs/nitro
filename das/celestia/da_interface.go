package celestia

import (
	"context"
)

type DataAvailabilityWriter interface {
	Store(context.Context, []byte) ([]byte, error)
}

type DataAvailabilityReader interface {
	Read(context.Context, *BlobPointer) ([]byte, *SquareData, error)
}
