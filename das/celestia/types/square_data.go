package types

type SquareData struct {
	RowRoots    [][]byte
	ColumnRoots [][]byte
	Rows        [][][]byte
	SquareSize  uint64 // Refers to original data square size
	StartRow    uint64
	EndRow      uint64
}
