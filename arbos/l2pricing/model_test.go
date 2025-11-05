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

func TestCompareLegacyPricingModelWithMultiConstraints(t *testing.T) {
	pricing := PricingForTest(t)

	toGwei := func(wei *big.Int) string {
		gweiDivisor := big.NewInt(params.GWei)
		weiRat := new(big.Rat).SetInt(wei)
		gweiDivisorRat := new(big.Rat).SetInt(gweiDivisor)
		gweiRat := new(big.Rat).Quo(weiRat, gweiDivisorRat)
		return gweiRat.FloatString(3)
	}

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
			_ = pricing.setConstraintsFromLegacy()

			pricing.updatePricingModelLegacy(timePassed)
			legacyPrice, _ := pricing.baseFeeWei.Get()

			pricing.updatePricingModelMultiConstraints(timePassed)
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
