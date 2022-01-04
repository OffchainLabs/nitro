package arbos

import (
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"math/big"
	"testing"
)

func TestGasPricingGasPool(t *testing.T) {
	st := arbosState.OpenArbosStateForTesting(t)
	expectedSmallGasPool := int64(arbosState.SmallGasPoolMax)
	expectedGasPool := int64(arbosState.GasPoolMax)

	checkGasPools := func() {
		t.Helper()
		if st.SmallGasPool() != expectedSmallGasPool {
			t.Fatal("wrong small gas pool, expected", expectedSmallGasPool, "but got", st.SmallGasPool())
		}

		if st.GasPool() != expectedGasPool {
			t.Fatal("wrong gas pool, expected", expectedGasPool, "but got", st.GasPool())
		}
	}

	checkGasPools()

	initialSub := int64(arbosState.SmallGasPoolMax / 2)
	st.AddToGasPools(-initialSub)

	expectedSmallGasPool -= initialSub
	expectedGasPool -= initialSub

	checkGasPools()

	elapseTimesToCheck := []int64{1, 2, 4, 10}
	totalTime := int64(0)
	for _, t := range elapseTimesToCheck {
		totalTime += t
	}
	if totalTime > (arbosState.SmallGasPoolMax-expectedSmallGasPool)/arbosState.SpeedLimitPerSecond {
		t.Fatal("should only test within small gas pool size")
	}

	for _, t := range elapseTimesToCheck {
		st.NotifyGasPricerThatTimeElapsed(uint64(t))
		expectedSmallGasPool += arbosState.SpeedLimitPerSecond * t
		expectedGasPool += arbosState.SpeedLimitPerSecond * t

		checkGasPools()
	}

	st.NotifyGasPricerThatTimeElapsed(10000000)

	expectedSmallGasPool = int64(arbosState.SmallGasPoolMax)
	expectedGasPool = int64(arbosState.GasPoolMax)

	checkGasPools()
}

func TestGasPricingPoolPrice(t *testing.T) {
	st := arbosState.OpenArbosStateForTesting(t)

	if st.GasPriceWei().Get().Cmp(big.NewInt(arbosState.MinimumGasPriceWei)) != 0 {
		t.Fatal("wrong initial gas price")
	}

	initialSub := int64(arbosState.SmallGasPoolMax * 4)
	st.AddToGasPools(-initialSub)

	if st.GasPriceWei().Get().Cmp(big.NewInt(arbosState.MinimumGasPriceWei)) != 0 {
		t.Fatal("price should not be changed")
	}

	st.NotifyGasPricerThatTimeElapsed(20)

	if st.GasPriceWei().Get().Cmp(big.NewInt(arbosState.MinimumGasPriceWei)) <= 0 {
		t.Fatal("price should be above minimum")
	}

	st.NotifyGasPricerThatTimeElapsed(500)

	if st.GasPriceWei().Get().Cmp(big.NewInt(arbosState.MinimumGasPriceWei)) != 0 {
		t.Fatal("price should return to minimum")
	}
}
