// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"github.com/offchainlabs/nitro/util/arbmath"
)

// SizeFactorBips returns ((maxTxDataSize - txSize) / maxTxDataSize)^exponent in basis points, truncating after each
// multiplication.
func SizeFactorBips(txSize, maxTxDataSize int, exponent uint) arbmath.UBips {
	if exponent == 0 {
		return arbmath.OneInUBips
	}
	if maxTxDataSize <= 0 || txSize >= maxTxDataSize {
		return 0
	}
	txSize = max(txSize, 0)
	// After the guards both operands are non-negative, and the sequencer config caps maxTxDataSize below the L2
	// message size, so neither the conversions nor the product can overflow.
	base := arbmath.UBips(uint64(maxTxDataSize-txSize) * uint64(arbmath.OneInUBips) / uint64(maxTxDataSize)) // #nosec G115
	factor := arbmath.OneInUBips
	for range exponent {
		factor = factor * base / arbmath.OneInUBips
		if factor == 0 {
			break
		}
	}
	return factor
}
