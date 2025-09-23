// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package otherpkg

// ExportedLarge has more than 5 fields
type ExportedLarge struct {
	Field1 int
	Field2 string
	Field3 bool
	Field4 float64
	Field5 int64
	Field6 []byte
}

type ExportedSmall struct {
	A int
	B string
	C bool
}
