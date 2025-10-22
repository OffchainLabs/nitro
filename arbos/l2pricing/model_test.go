// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"testing"
)

func TestComputeConstraintDivisor(t *testing.T) {
	cases := []struct {
		target uint64
		period uint64
		want   uint64
	}{
		{
			target: 7_000_000,
			period: 12,
			want:   630_000_000,
		},
		{
			target: 15_000_000,
			period: 86_400, // one day
			want:   131_850_000_000,
		},
	}
	for _, test := range cases {
		got := computeConstraintDivisor(test.target, test.period)
		if got != test.want {
			t.Errorf("wrong result for target=%v period=%v: got %v, want %v",
				test.target, test.period, got, test.want)
		}
	}
}

func TestCompareLegacyPricingModelWithMultiConstraints(t *testing.T) {
	pricing := PricingForTest(t)

	// In this test, we don't check for storage set errors because they won't happen and they
	// are not the focus of the test.

	// Set the innertia to a value that is divisible by 30 to negate the rounding error
	_ = pricing.SetPricingInertia(120)

	// Set the tolerance to zero because this doesn't exist in the new model
	_ = pricing.SetBacklogTolerance(0)

	// Initialize with a single constraint based on the legacy model
	_ = pricing.SetConstraintsFromLegacy()

	// Compare the basefee for both models with different backlogs
	for backlogShift := range uint64(32) {
		for timePassed := range uint64(5) {
			backlog := uint64(1 << backlogShift)

			_ = pricing.gasBacklog.Set(backlog)
			pricing.updatePricingModelLegacy(timePassed)
			legacyPrice, _ := pricing.baseFeeWei.Get()

			constraint := pricing.OpenConstraintAt(0)
			_ = constraint.backlog.Set(backlog)
			pricing.updatePricingModelMultiConstraints(timePassed)
			multiPrice, _ := pricing.baseFeeWei.Get()

			if multiPrice.Cmp(legacyPrice) != 0 {
				t.Errorf("wrong result: backlog=%v, timePassed=%v, multiPrice=%v, legacyPrice=%v",
					backlog, timePassed, multiPrice, legacyPrice)
			}
		}
	}
}
