package types

import (
	"context"
)

type CelestiaWriter interface {
	Store(context.Context, []byte) ([]byte, error)
}

type SquareData struct {
	RowRoots    [][]byte   `json:"row_roots"`
	ColumnRoots [][]byte   `json:"column_roots"`
	Rows        [][][]byte `json:"rows"`
	SquareSize  uint64     `json:"square_size"` // Refers to original data square size
	StartRow    uint64     `json:"start_row"`
	EndRow      uint64     `json:"end_row"`
}

type CelestiaReader interface {
	Read(context.Context, *BlobPointer) ([]byte, *SquareData, error)
	GetProof(ctx context.Context, msg []byte) ([]byte, error)
}
