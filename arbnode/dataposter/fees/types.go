// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package fees

import (
	"math/big"
)

// Split between blob and non-blob for calculations
type BlobSplit[T any] struct {
	Blob    T
	NonBlob T
}

func (s *BlobSplit[T]) SelectIfBlobs(hasBlobs bool) T {
	if hasBlobs {
		return s.Blob
	} else {
		return s.NonBlob
	}
}

// Calculated fee and tip caps
type Caps struct {
	Fee BlobSplit[*big.Int]
	Tip *big.Int
}
