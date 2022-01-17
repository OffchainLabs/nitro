package arbos

import (
	"testing"

	"github.com/offchainlabs/arbstate/arbos/arbosState"
)

func TestGasPricingGasPool(t *testing.T) {
	state := arbosState.OpenArbosStateForTesting(t)
	expectedSmallGasPool := int64(arbosState.SmallGasPoolMax)
	expectedGasPool := int64(arbosState.GasPoolMax)

	checkGasPools := func() {
		t.Helper()
		if smallGasPool(t, state) != expectedSmallGasPool {
			Fail(t, "wrong small gas pool, expected", expectedSmallGasPool, "but got", smallGasPool(t, state))
		}

		if gasPool(t, state) != expectedGasPool {
			Fail(t, "wrong gas pool, expected", expectedGasPool, "but got", gasPool(t, state))
		}
	}

	checkGasPools()

	initialSub := int64(arbosState.SmallGasPoolMax / 2)
	state.AddToGasPools(-initialSub)

	expectedSmallGasPool -= initialSub
	expectedGasPool -= initialSub

	checkGasPools()

	elapseTimesToCheck := []int64{1, 2, 4, 10}
	totalTime := int64(0)
	for _, t := range elapseTimesToCheck {
		totalTime += t
	}
	if totalTime > (arbosState.SmallGasPoolMax-expectedSmallGasPool)/arbosState.SpeedLimitPerSecond {
		Fail(t, "should only test within small gas pool size")
	}

	for _, t := range elapseTimesToCheck {
		state.NotifyGasPricerThatTimeElapsed(uint64(t))
		expectedSmallGasPool += arbosState.SpeedLimitPerSecond * t
		expectedGasPool += arbosState.SpeedLimitPerSecond * t

		checkGasPools()
	}

	state.NotifyGasPricerThatTimeElapsed(10000000)

	expectedSmallGasPool = int64(arbosState.SmallGasPoolMax)
	expectedGasPool = int64(arbosState.GasPoolMax)

	checkGasPools()
}

func TestGasPricingPoolPrice(t *testing.T) {
	state := arbosState.OpenArbosStateForTesting(t)

	if gasPriceWei(t, state) != arbosState.MinimumGasPriceWei {
		Fail(t, "wrong initial gas price")
	}

	initialSub := int64(arbosState.SmallGasPoolMax * 4)
	state.AddToGasPools(-initialSub)

	if gasPriceWei(t, state) != arbosState.MinimumGasPriceWei {
		Fail(t, "price should not be changed")
	}

	state.NotifyGasPricerThatTimeElapsed(20)

	if gasPriceWei(t, state) <= arbosState.MinimumGasPriceWei {
		Fail(t, "price should be above minimum")
	}

	state.NotifyGasPricerThatTimeElapsed(500)

	if gasPriceWei(t, state) != arbosState.MinimumGasPriceWei {
		Fail(t, "price should return to minimum")
	}
}

func gasPriceWei(t *testing.T, state *arbosState.ArbosState) uint64 {
	t.Helper()
	price, err := state.GasPriceWei()
	Require(t, err)
	return price.Uint64()
}

func gasPool(t *testing.T, state *arbosState.ArbosState) int64 {
	t.Helper()
	pool, err := state.GasPool()
	Require(t, err)
	return pool
}

func smallGasPool(t *testing.T, state *arbosState.ArbosState) int64 {
	t.Helper()
	pool, err := state.SmallGasPool()
	Require(t, err)
	return pool
}
