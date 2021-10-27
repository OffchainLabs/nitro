package arbos

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

const SpeedLimitPerSecond = 1000000
const GasPoolMax = SpeedLimitPerSecond * 10 * 60
const SmallGasPoolSeconds = 60
const SmallGasPoolMax = SpeedLimitPerSecond * SmallGasPoolSeconds

const PerBlockGasLimit uint64 = 20 * 1000000

const MinimumGasPriceWei = 1 * params.GWei
const InitialGasPriceWei = MinimumGasPriceWei

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

	secondsLeft := secondsElapsed
	for secondsLeft > 0 {
		if (gasPool == GasPoolMax) && (smallGasPool == SmallGasPoolMax) {
			if price.Cmp(minPrice) <= 0 {
				state.SetGasPool(GasPoolMax)
				state.SetSmallGasPool(SmallGasPoolMax)
				state.SetGasPriceWei(minPrice)
				return
			} else {
				if secondsLeft >= 83 {
					// price is cut in half every 83 seconds, when both gas pools are full
					price = new(big.Int).Div(price, big.NewInt(2))
					secondsLeft -= 83
				} else {
					price = new(big.Int).Div(new(big.Int).Mul(price, big.NewInt(119)), big.NewInt(120))
					secondsLeft -= 1
				}
			}
		} else {
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

			secondsLeft--
		}
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
