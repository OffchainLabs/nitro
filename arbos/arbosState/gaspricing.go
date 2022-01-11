package arbosState

import (
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/util"
)

const SpeedLimitPerSecond = 1000000
const GasPoolMax = SpeedLimitPerSecond * 10 * 60
const SmallGasPoolSeconds = 60
const SmallGasPoolMax = SpeedLimitPerSecond * SmallGasPoolSeconds

const PerBlockGasLimit uint64 = 20 * 1000000

const MinimumGasPriceWei = 1 * params.GWei
const InitialGasPriceWei = MinimumGasPriceWei

func (state *ArbosState) AddToGasPools(gas int64) {
	gasPool, _ := state.GasPool()
	smallGasPool, _ := state.SmallGasPool()
	state.Restrict(state.SetGasPool(util.SaturatingAdd(gasPool, gas)))
	state.Restrict(state.SetSmallGasPool(util.SaturatingAdd(smallGasPool, gas)))
}

func (state *ArbosState) NotifyGasPricerThatTimeElapsed(secondsElapsed uint64) {
	gasPool, _ := state.GasPool()
	smallGasPool, _ := state.SmallGasPool()
	price, _ := state.GasPriceWei()
	maxPrice, err := state.MaxGasPriceWei()
	state.Restrict(err)

	minPrice := big.NewInt(MinimumGasPriceWei)
	maxPoolAsBig := big.NewInt(GasPoolMax)
	maxSmallPoolAsBig := big.NewInt(SmallGasPoolMax)
	maxProd := new(big.Int).Mul(maxPoolAsBig, maxSmallPoolAsBig)
	numeratorBase := new(big.Int).Mul(big.NewInt(121), maxProd)
	denominator := new(big.Int).Mul(big.NewInt(120), maxProd)

	secondsLeft := secondsElapsed
	for secondsLeft > 0 {
		if (gasPool == GasPoolMax) && (smallGasPool == SmallGasPoolMax) {
			// both gas pools are full, so we should multiply the price by 119/120 for each second that elapses
			if price.Cmp(minPrice) <= 0 {
				// price is already at the minimum, so no need to iterate further
				_ = state.SetGasPool(GasPoolMax)
				_ = state.SetSmallGasPool(SmallGasPoolMax)
				_ = state.SetGasPriceWei(minPrice)
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
	if price.Cmp(maxPrice) > 0 {
		log.Warn(
			"ArbOS is trying to set a price that's unsafe for geth",
			"attempted", price,
			"used", maxPrice,
		)
		price = maxPrice
	}
	state.Restrict(state.SetGasPool(gasPool))
	state.Restrict(state.SetSmallGasPool(smallGasPool))
	state.Restrict(state.SetGasPriceWei(price))
}

func (state *ArbosState) CurrentPerBlockGasLimit() (uint64, error) {
	pool, err := state.GasPool()
	if pool < 0 || err != nil {
		return 0, err
	} else if pool > int64(PerBlockGasLimit) {
		return PerBlockGasLimit, nil
	} else {
		return uint64(pool), nil
	}
}

func MaxPerBlockGasLimit() uint64 {
	return PerBlockGasLimit
}
