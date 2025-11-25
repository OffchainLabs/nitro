// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func PricingForTest(t *testing.T) *L2PricingState {
	storage := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	err := InitializeL2PricingState(storage)
	Require(t, err)
	return OpenL2PricingState(storage, params.MaxDebugArbosVersionSupported)
}

func fakeBlockUpdate(t *testing.T, pricing *L2PricingState, gasUsed int64, timePassed uint64) {
	t.Helper()

	pricing.storage.Burner().Restrict(pricing.addToGasPoolLegacy(-gasUsed))
	pricing.updatePricingModelLegacy(timePassed)
}

func TestPricingModelExp(t *testing.T) {
	pricing := PricingForTest(t)
	minPrice := getMinPrice(t, pricing)
	price := getPrice(t, pricing)
	limit := getSpeedLimit(t, pricing)

	if price != minPrice {
		Fail(t, "price not minimal", price, minPrice)
	}

	// show that running at the speed limit with a full pool is a steady-state
	colors.PrintBlue("full pool & speed limit")
	for seconds := 0; seconds < 4; seconds++ {
		// #nosec G115
		fakeBlockUpdate(t, pricing, int64(seconds)*int64(limit), uint64(seconds))
		if getPrice(t, pricing) != minPrice {
			Fail(t, "price changed when it shouldn't have")
		}
	}

	// show that running at the speed limit with a target pool is close to a steady-state
	// note that for large enough spans of time the price will rise a minuscule amount due to the pool's avg
	colors.PrintBlue("pool target & speed limit")
	for seconds := 0; seconds < 4; seconds++ {
		// #nosec G115
		fakeBlockUpdate(t, pricing, int64(seconds)*int64(limit), uint64(seconds))
		if getPrice(t, pricing) != minPrice {
			Fail(t, "price changed when it shouldn't have")
		}
	}

	// show that running over the speed limit escalates the price before the pool drains
	colors.PrintBlue("exceeding the speed limit")
	for {
		// #nosec G115
		fakeBlockUpdate(t, pricing, 8*int64(limit), 1)
		newPrice := getPrice(t, pricing)
		if newPrice < price {
			Fail(t, "the price shouldn't have fallen")
		}
		if newPrice > price {
			break
		}
		price = newPrice
	}

	// empty the pool
	price = getPrice(t, pricing)
	Require(t, pricing.SetGasBacklog(100000000))

	// show that nothing happens when no time has passed and no gas has been burnt
	colors.PrintBlue("nothing should happen")
	fakeBlockUpdate(t, pricing, 0, 0)

	// show that the pool will escalate the price
	colors.PrintBlue("gas pool is empty")
	fakeBlockUpdate(t, pricing, 0, 1)
	if getPrice(t, pricing) <= price {
		fmt.Println(price, getPrice(t, pricing))
		Fail(t, "price should have risen")
	}
}

func getPrice(t *testing.T, pricing *L2PricingState) uint64 {
	value, err := pricing.BaseFeeWei()
	Require(t, err)
	return arbmath.BigToUintOrPanic(value)
}

func getMinPrice(t *testing.T, pricing *L2PricingState) uint64 {
	value, err := pricing.MinBaseFeeWei()
	Require(t, err)
	return arbmath.BigToUintOrPanic(value)
}

func getSpeedLimit(t *testing.T, pricing *L2PricingState) uint64 {
	value, err := pricing.SpeedLimitPerSecond()
	Require(t, err)
	return value
}

func getConstraintsLength(t *testing.T, pricing *L2PricingState) uint64 {
	length, err := pricing.ConstraintsLength()
	Require(t, err)
	return length
}

func TestGasConstraints(t *testing.T) {
	pricing := PricingForTest(t)
	if got := getConstraintsLength(t, pricing); got != 0 {
		t.Fatalf("wrong number of constraints: got %v want 0", got)
	}
	const n uint64 = 10
	for i := range n {
		Require(t, pricing.AddConstraint(100*i+1, 100*i+2, 100*i+3))
	}
	if got := getConstraintsLength(t, pricing); got != n {
		t.Fatalf("wrong number of constraints: got %v want %v", got, n)
	}
	for i := range n {
		constraint := pricing.OpenConstraintAt(i)
		target, err := constraint.target.Get()
		Require(t, err)
		if want := 100*i + 1; target != want {
			t.Errorf("wrong target: got %v, want %v", target, want)
		}
		inertia, err := constraint.adjustmentWindow.Get()
		Require(t, err)
		if want := 100*i + 2; inertia != want {
			t.Errorf("wrong inertia: got %v, want %v", inertia, want)
		}
		backlog, err := constraint.backlog.Get()
		Require(t, err)
		if want := 100*i + 3; backlog != want {
			t.Errorf("wrong backlog: got %v, want %v", backlog, want)
		}
	}
	Require(t, pricing.ClearConstraints())
	if got := getConstraintsLength(t, pricing); got != 0 {
		t.Fatalf("wrong number of constraints: got %v want 0", got)
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
