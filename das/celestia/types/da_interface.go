package types

import (
	"context"
)

type CelestiaWriter interface {
	Store(context.Context, []byte) ([]byte, error)
}

type CelestiaReader interface {
	Read(context.Context, *BlobPointer) ([]byte, *SquareData, error)
	GetProof(ctx context.Context, msg []byte) ([]byte, error)
}
