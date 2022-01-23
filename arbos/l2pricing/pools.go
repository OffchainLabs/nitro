//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/util"
)

const SpeedLimitPerSecond = 1000000
const GasPoolMax = SpeedLimitPerSecond * 10 * 60
const SmallGasPoolSeconds = 60
const SmallGasPoolMax = SpeedLimitPerSecond * SmallGasPoolSeconds

const PerBlockGasLimit uint64 = math.MaxInt64 // so we can cast it to int

const MinimumGasPriceWei = 1 * params.GWei
const InitialGasPriceWei = MinimumGasPriceWei

func (ps *L2PricingState) AddToGasPools(gas int64) {
	gasPool, _ := ps.GasPool()
	smallGasPool, _ := ps.SmallGasPool()
	ps.Restrict(ps.SetGasPool(util.SaturatingAdd(gasPool, gas)))
	ps.Restrict(ps.SetSmallGasPool(util.SaturatingAdd(smallGasPool, gas)))
}

func (ps *L2PricingState) NotifyGasPricerThatTimeElapsed(secondsElapsed uint64) {
	gasPool, _ := ps.GasPool()
	smallGasPool, _ := ps.SmallGasPool()
	price, _ := ps.GasPriceWei()
	maxPrice, err := ps.MaxGasPriceWei()
	ps.Restrict(err)

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
				_ = ps.SetGasPool(GasPoolMax)
				_ = ps.SetSmallGasPool(SmallGasPoolMax)
				_ = ps.SetGasPriceWei(minPrice)
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
	ps.Restrict(ps.SetGasPool(gasPool))
	ps.Restrict(ps.SetSmallGasPool(smallGasPool))
	ps.Restrict(ps.SetGasPriceWei(price))
}

func (ps *L2PricingState) CurrentPerBlockGasLimit() (uint64, error) {
	pool, err := ps.GasPool()
	if pool < 0 || err != nil {
		return 0, err
	} else {
		return uint64(pool), nil
	}
}

func MaxPerBlockGasLimit() uint64 {
	return PerBlockGasLimit
}
