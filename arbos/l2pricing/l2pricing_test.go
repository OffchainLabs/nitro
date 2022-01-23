//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"testing"

	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util/testhelpers"
)

func PricingForTest(t *testing.T) *L2PricingState {
	storage := storage.NewMemoryBacked(burn.NewSystemBurner(true))
	err := InitializeL2PricingState(storage)
	Require(t, err)
	return OpenL2PricingState(storage)
}

func TestGasPricingGasPool(t *testing.T) {
	pricing := PricingForTest(t)
	expectedSmallGasPool, err := pricing.SmallGasPoolMax()
	Require(t, err)
	expectedGasPool, err := pricing.GasPoolMax()
	Require(t, err)

	checkGasPools := func() {
		t.Helper()
		smallGasPool := smallGasPool(t, pricing)
		if smallGasPool != expectedSmallGasPool {
			Fail(t, "wrong small gas pool, expected", expectedSmallGasPool, "but got", smallGasPool)
		}
		gasPool := gasPool(t, pricing)
		if gasPool != expectedGasPool {
			Fail(t, "wrong gas pool, expected", expectedGasPool, "but got", gasPool)
		}
	}

	checkGasPools()

	gasPoolMax, err := pricing.GasPoolMax()
	Require(t, err)
	smallGasPoolMax, err := pricing.SmallGasPoolMax()
	Require(t, err)

	initialSub := int64(smallGasPoolMax / 2)
	pricing.AddToGasPools(-initialSub)

	expectedSmallGasPool -= initialSub
	expectedGasPool -= initialSub

	checkGasPools()

	elapseTimesToCheck := []int64{1, 2, 4, 10}
	totalTime := int64(0)
	for _, t := range elapseTimesToCheck {
		totalTime += t
	}
	if totalTime > (smallGasPoolMax-expectedSmallGasPool)/InitialSpeedLimitPerSecond {
		Fail(t, "should only test within small gas pool size")
	}

	for _, t := range elapseTimesToCheck {
		pricing.NotifyGasPricerThatTimeElapsed(uint64(t))
		expectedSmallGasPool += InitialSpeedLimitPerSecond * t
		expectedGasPool += InitialSpeedLimitPerSecond * t

		checkGasPools()
	}

	pricing.NotifyGasPricerThatTimeElapsed(10000000)

	expectedSmallGasPool = smallGasPoolMax
	expectedGasPool = gasPoolMax

	checkGasPools()
}

func TestGasPricingPoolPrice(t *testing.T) {
	pricing := PricingForTest(t)
	smallGasPoolMax, err := pricing.SmallGasPoolMax()
	Require(t, err)

	if gasPriceWei(t, pricing) != InitialMinimumGasPriceWei {
		Fail(t, "wrong initial gas price")
	}

	pricing.AddToGasPools(-smallGasPoolMax * 4)

	if gasPriceWei(t, pricing) != InitialMinimumGasPriceWei {
		Fail(t, "price should not be changed")
	}

	pricing.NotifyGasPricerThatTimeElapsed(20)

	if gasPriceWei(t, pricing) <= InitialMinimumGasPriceWei {
		Fail(t, "price should be above minimum")
	}

	pricing.NotifyGasPricerThatTimeElapsed(500)

	if gasPriceWei(t, pricing) != InitialMinimumGasPriceWei {
		Fail(t, "price should return to minimum")
	}
}

func gasPriceWei(t *testing.T, state *L2PricingState) uint64 {
	t.Helper()
	price, err := state.GasPriceWei()
	Require(t, err)
	return price.Uint64()
}

func gasPool(t *testing.T, state *L2PricingState) int64 {
	t.Helper()
	pool, err := state.GasPool()
	Require(t, err)
	return pool
}

func smallGasPool(t *testing.T, state *L2PricingState) int64 {
	t.Helper()
	pool, err := state.SmallGasPool()
	Require(t, err)
	return pool
}

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
