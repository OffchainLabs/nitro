// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func toGwei(wei *big.Int) string {
	gweiDivisor := big.NewInt(params.GWei)
	weiRat := new(big.Rat).SetInt(wei)
	gweiDivisorRat := new(big.Rat).SetInt(gweiDivisor)
	gweiRat := new(big.Rat).Quo(weiRat, gweiDivisorRat)
	return gweiRat.FloatString(3)
}

func TestCompareLegacyPricingModelWithSingleGasConstraints(t *testing.T) {
	pricing := PricingForTest(t)

	// In this test, we don't check for storage set errors because they won't happen and they
	// are not the focus of the test.

	// Set the speed limit
	_ = pricing.SetSpeedLimitPerSecond(InitialSpeedLimitPerSecondV6)

	// Compare the basefee for both models with different backlogs
	var backlogs = []uint64{0}
	for i := range uint64(9) {
		backlogs = append(backlogs, 1_000_000*(1+i))
		backlogs = append(backlogs, 10_000_000*(1+i))
		backlogs = append(backlogs, 100_000_000*(1+i))
		backlogs = append(backlogs, 1_000_000_000*(1+i))
		backlogs = append(backlogs, 10_000_000_000*(1+i))
	}

	slices.Sort(backlogs)
	for timePassed := range uint64(100) {
		for _, backlog := range backlogs {
			_ = pricing.gasBacklog.Set(backlog)

			// Initialize with a single constraint based on the legacy model
			_ = pricing.setGasConstraintsFromLegacy()

			pricing.updatePricingModelLegacy(timePassed)
			legacyPrice, _ := pricing.baseFeeWei.Get()

			pricing.updatePricingModelSingleConstraints(timePassed)
			multiPrice, _ := pricing.baseFeeWei.Get()

			if timePassed == 0 {
				fmt.Printf("backlog=%vM\tlegacy=%v gwei\tmultiConstraints=%v gwei\ttimePassed=%v\n",
					backlog/1_000_000, toGwei(legacyPrice), toGwei(multiPrice), timePassed)
			}

			if multiPrice.Cmp(legacyPrice) != 0 {
				t.Errorf("wrong result: backlog=%v, timePassed=%v, multiPrice=%v, legacyPrice=%v",
					backlog, timePassed, multiPrice, legacyPrice)
			}
		}
	}
}

func TestCompareSingleGasConstraintsPricingModelWithMultiGasConstraints(t *testing.T) {
	pricing := PricingForTest(t)

	// Configure base parameters (same as single-constraint test)
	_ = pricing.SetSpeedLimitPerSecond(InitialSpeedLimitPerSecondV6)
	inertia, _ := pricing.PricingInertia()
	target, _ := pricing.SpeedLimitPerSecond()

	// Build a set of backlogs to test
	var backlogs = []uint64{0}
	for i := range uint64(9) {
		backlogs = append(backlogs, 1_000_000*(1+i))
		backlogs = append(backlogs, 10_000_000*(1+i))
		backlogs = append(backlogs, 100_000_000*(1+i))
		backlogs = append(backlogs, 1_000_000_000*(1+i))
		backlogs = append(backlogs, 10_000_000_000*(1+i))
	}

	fmt.Printf("target = %v, inertia = %v\n", target, inertia)

	slices.Sort(backlogs)
	for timePassed := range uint64(1) {
		for _, backlog := range backlogs {
			// Clear any existing constraints
			Require(t, pricing.ClearGasConstraints())
			Require(t, pricing.ClearMultiGasConstraints())

			// Manually create a single-gas constraint:
			// this is the "single-dimensional" model: one constraint with (target, inertia, backlog).
			Require(t, pricing.AddGasConstraint(target, inertia, backlog))

			// Transfer single-gas constraint to multi-gas constraint
			Require(t, pricing.setMultiGasConstraintsFromSingleGasConstraints())

			// Trigger single-constraint pricing update
			pricing.updatePricingModelSingleConstraints(timePassed)
			singlePrice, err := pricing.baseFeeWei.Get()
			Require(t, err)

			// Trigger multi-gas pricing update
			pricing.updatePricingModelMultiConstraints(timePassed)
			multiPrice, err := pricing.baseFeeWei.Get()
			Require(t, err)

			if timePassed == 0 {
				fmt.Printf("backlog=%vM\tlegacy=%v gwei\tmultiConstraints=%v gwei\ttimePassed=%v\n",
					backlog/1_000_000, toGwei(singlePrice), toGwei(multiPrice), timePassed)
			}

			if multiPrice.Cmp(singlePrice) != 0 {
				t.Errorf(
					"mismatch: backlog=%v timePassed=%v single=%v multi=%v",
					backlog, timePassed, singlePrice, multiPrice,
				)
			}
		}
	}
}
