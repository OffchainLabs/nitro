package arbos

import "math/big"

const SpeedLimitPerSecond = 1000000
const GasPoolMax = SpeedLimitPerSecond * 10 * 60
const SmallGasPoolMax = SpeedLimitPerSecond * 60

const PerBlockGasLimit uint64 = 20 * 1000000

const Gwei = 1000000000
const MinimumGasPriceWei = 1 * Gwei

func (state *ArbosState) notifyGasUsed(gas uint64) {
	gasInt := int64(gas)
	state.SetGasPool(state.GasPool() - gasInt)
	state.SetSmallGasPool(state.SmallGasPool() - gasInt)
}

func (state *ArbosState) notifyGasPricerThatTimeElapsed(secondsElapsed uint64) {
	gasPool := state.GasPool()
	smallGasPool := state.SmallGasPool()
	price := state.GasPriceWei()

	minPrice := big.NewInt(MinimumGasPriceWei)
	maxPoolAsBig := big.NewInt(GasPoolMax)
	maxSmallPoolAsBig := big.NewInt(SmallGasPoolMax)
	maxProd := new(big.Int).Mul(maxPoolAsBig, maxSmallPoolAsBig)
	numeratorBase := new(big.Int).Mul(big.NewInt(121), maxProd)
	denominator := new(big.Int).Mul(big.NewInt(120), maxProd)

	for i := uint64(0); i < secondsElapsed; i++ {
		if (gasPool >= GasPoolMax) && (smallGasPool >= SmallGasPoolMax) && (price.Cmp(minPrice) <= 0) {
			break
		}
		gasPool = gasPool + SpeedLimitPerSecond
		if gasPool > GasPoolMax {
			gasPool = GasPoolMax
		}
		smallGasPool = smallGasPool + SpeedLimitPerSecond
		if smallGasPool > SmallGasPoolMax {
			smallGasPool = SmallGasPoolMax
		}

		clippedGasPool := gasPool
		if clippedGasPool < 0 {
			clippedGasPool = 0
		}
		clippedSmallGasPool := smallGasPool
		if clippedSmallGasPool < 0 {
			clippedSmallGasPool = 0
		}

		numerator := new(big.Int).Sub(
			numeratorBase,
			new(big.Int).Add(
				new(big.Int).Mul(big.NewInt(clippedGasPool), maxSmallPoolAsBig),
				new(big.Int).Mul(big.NewInt(clippedSmallGasPool), maxPoolAsBig),
			),
		)

		// no need to clip the price here, because we'll do that on exit from the loop
		price = new(big.Int).Div(
			new(big.Int).Mul(price, numerator),
			denominator,
		)
	}

	if price.Cmp(minPrice) < 0 {
		price = minPrice
	}
	state.SetGasPool(gasPool)
	state.SetSmallGasPool(smallGasPool)
	state.SetGasPriceWei(price)
}

func (state *ArbosState) CurrentPerBlockGasLimit() uint64 {
	pool := state.GasPool()
	if pool < 0 {
		return 0
	} else if pool > int64(PerBlockGasLimit) {
		return PerBlockGasLimit
	} else {
		return uint64(pool)
	}
}

func MaxPerBlockGasLimit() uint64 {
	return PerBlockGasLimit
}
