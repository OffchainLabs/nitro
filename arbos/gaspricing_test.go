package arbos

import (
	"github.com/offchainlabs/arbstate/arbos/l2pricing"
	"math/big"
	"testing"
)

func TestGasPricingGasPool(t *testing.T) {
	arbosState := OpenArbosStateForTest(t)
	st := arbosState.L2PricingState()
	expectedSmallGasPool := int64(l2pricing.SmallGasPoolMax)
	expectedGasPool := int64(l2pricing.GasPoolMax)

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

	initialSub := int64(l2pricing.SmallGasPoolMax / 2)
	st.NotifyGasUsed(uint64(initialSub))

	expectedSmallGasPool -= initialSub
	expectedGasPool -= initialSub

	checkGasPools()

	elapseTimesToCheck := []int64{1, 2, 4, 10}
	totalTime := int64(0)
	for _, t := range elapseTimesToCheck {
		totalTime += t
	}
	if totalTime > (l2pricing.SmallGasPoolMax-expectedSmallGasPool)/l2pricing.SpeedLimitPerSecond {
		t.Fatal("should only test within small gas pool size")
	}

	for _, t := range elapseTimesToCheck {
		st.NotifyGasPricerThatTimeElapsed(uint64(t))
		expectedSmallGasPool += l2pricing.SpeedLimitPerSecond * t
		expectedGasPool += l2pricing.SpeedLimitPerSecond * t

		checkGasPools()
	}

	st.NotifyGasPricerThatTimeElapsed(10000000)

	expectedSmallGasPool = int64(l2pricing.SmallGasPoolMax)
	expectedGasPool = int64(l2pricing.GasPoolMax)

	checkGasPools()
}

func TestGasPricingPoolPrice(t *testing.T) {
	arbosState := OpenArbosStateForTest(t)
	st := arbosState.L2PricingState()

	if st.GasPriceWei().Cmp(big.NewInt(l2pricing.MinimumGasPriceWei)) != 0 {
		t.Fatal("wrong initial gas price")
	}

	initialSub := int64(l2pricing.SmallGasPoolMax * 4)
	st.NotifyGasUsed(uint64(initialSub))

	if st.GasPriceWei().Cmp(big.NewInt(l2pricing.MinimumGasPriceWei)) != 0 {
		t.Fatal("price should not be changed")
	}

	st.NotifyGasPricerThatTimeElapsed(20)

	if st.GasPriceWei().Cmp(big.NewInt(l2pricing.MinimumGasPriceWei)) <= 0 {
		t.Fatal("price should be above minimum")
	}

	st.NotifyGasPricerThatTimeElapsed(500)

	if st.GasPriceWei().Cmp(big.NewInt(l2pricing.MinimumGasPriceWei)) != 0 {
		t.Fatal("price should return to minimum")
	}
}
