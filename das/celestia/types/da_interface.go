package types

import (
	"context"
)

type DataAvailabilityWriter interface {
	Store(context.Context, []byte) ([]byte, error)
}

type DataAvailabilityReader interface {
	Read(context.Context, *BlobPointer) ([]byte, *SquareData, error)
	GetProof(ctx context.Context, msg []byte) ([]byte, error)
}
