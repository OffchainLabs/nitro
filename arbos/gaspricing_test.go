package arbos

import (
	"math/big"
	"testing"
)

func TestGasPricingGasPool(t *testing.T) {
	st := OpenArbosStateForTest(t)
	expectedSmallGasPool := int64(SmallGasPoolMax)
	expectedGasPool := int64(GasPoolMax)

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

	initialSub := int64(SmallGasPoolMax / 2)
	st.notifyGasUsed(uint64(initialSub))

	expectedSmallGasPool -= initialSub
	expectedGasPool -= initialSub

	checkGasPools()

	elapseTimesToCheck := []int64{1, 2, 4, 10}
	totalTime := int64(0)
	for _, t := range elapseTimesToCheck {
		totalTime += t
	}
	if totalTime > (SmallGasPoolMax-expectedSmallGasPool)/SpeedLimitPerSecond {
		t.Fatal("should only test within small gas pool size")
	}

	for _, t := range elapseTimesToCheck {
		st.notifyGasPricerThatTimeElapsed(uint64(t))
		expectedSmallGasPool += SpeedLimitPerSecond * t
		expectedGasPool += SpeedLimitPerSecond * t

		checkGasPools()
	}

	st.notifyGasPricerThatTimeElapsed(10000000)

	expectedSmallGasPool = int64(SmallGasPoolMax)
	expectedGasPool = int64(GasPoolMax)

	checkGasPools()
}

func TestGasPricingPoolPrice(t *testing.T) {
	st := OpenArbosStateForTest(t)

	if st.GasPriceWei().Cmp(big.NewInt(MinimumGasPriceWei)) != 0 {
		t.Fatal("wrong initial gas price")
	}

	initialSub := int64(SmallGasPoolMax * 4)
	st.notifyGasUsed(uint64(initialSub))

	if st.GasPriceWei().Cmp(big.NewInt(MinimumGasPriceWei)) != 0 {
		t.Fatal("price should not be changed")
	}

	st.notifyGasPricerThatTimeElapsed(20)

	if st.GasPriceWei().Cmp(big.NewInt(MinimumGasPriceWei)) <= 0 {
		t.Fatal("price should be above minimum")
	}

	st.notifyGasPricerThatTimeElapsed(500)

	if st.GasPriceWei().Cmp(big.NewInt(MinimumGasPriceWei)) != 0 {
		t.Fatal("price should return to minimum")
	}
}
