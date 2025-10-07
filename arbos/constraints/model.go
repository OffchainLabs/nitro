// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// The constraints package tracks the multi-dimensional gas usage to apply constraint-based pricing.
package constraints

import (
	"math/big"

	"github.com/offchainlabs/nitro/util/arbmath"
)

const (
	PricingInertiaFactor = 30
)

type PricingState struct {
	constraints ResourceConstraints
}

// UpdatePricingModel adjusts the basefee according to simplified constraint-based pricing.
// Formula: basefee = F_min * exp( max_i ( B_i / (30 * T_i * sqrt(Δ_i)) ) )
func (ps *PricingState) UpdatePricingModel(minBaseFee *big.Int, timePassed uint64) *big.Int {
	var maxExponentBips arbmath.Bips

	for c := range ps.constraints.All() {
		// Decay per-constraint backlog by T_i * timePassed
		c.RemoveFromBacklog(timePassed)

		if c.backlog == 0 || c.targetPerSec == 0 || c.period == 0 {
			continue
		}

		// Normalized backlog = B_i / denominator
		expBips := arbmath.NaturalToBips(arbmath.SaturatingCast[int64](c.backlog)) / arbmath.SaturatingCast[arbmath.Bips](c.denominator)

		// Pick the maximum exponent across all constraints
		if expBips > maxExponentBips {
			maxExponentBips = expBips
		}
	}

	// Apply the maximum exponent
	if maxExponentBips == 0 {
		return new(big.Int).Set(minBaseFee)
	}
	return arbmath.BigMulByBips(minBaseFee, arbmath.ApproxExpBasisPoints(maxExponentBips, 4))
}
