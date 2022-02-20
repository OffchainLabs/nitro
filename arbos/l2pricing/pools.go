//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/util"
)

const InitialSpeedLimitPerSecond = 1000000
const InitialPerBlockGasLimit uint64 = 20 * 1000000
const InitialMinimumGasPriceWei = 1 * params.GWei
const InitialBaseFeeWei = InitialMinimumGasPriceWei
const InitialGasPoolSeconds = 10 * 60
const InitialSmallGasPoolSeconds = 60

func (ps *L2PricingState) AddToGasPools(gas int64) {
	gasPool, _ := ps.GasPool()
	smallGasPool, _ := ps.SmallGasPool()
	ps.Restrict(ps.SetGasPool(util.SaturatingAdd(gasPool, gas)))
	ps.Restrict(ps.SetSmallGasPool(util.SaturatingAdd(smallGasPool, gas)))
}

func (ps *L2PricingState) NotifyGasPricerThatTimeElapsed(secondsElapsed uint64) {
	gasPool, _ := ps.GasPool()
	smallGasPool, _ := ps.SmallGasPool()
	gasPoolMax, _ := ps.GasPoolMax()
	smallGasPoolMax, _ := ps.SmallGasPoolMax()
	speedLimit, _ := ps.SpeedLimitPerSecond()
	price, _ := ps.GasPriceWei()
	minPrice, _ := ps.MinGasPriceWei()
	maxPrice, err := ps.MaxGasPriceWei()
	ps.Restrict(err)

	maxPoolAsBig := big.NewInt(gasPoolMax)
	maxSmallPoolAsBig := big.NewInt(smallGasPoolMax)
	maxPoolProduct := util.BigMul(maxPoolAsBig, maxSmallPoolAsBig)

	secondsLeft := secondsElapsed
	for secondsLeft > 0 {
		if (gasPool == gasPoolMax) && (smallGasPool == smallGasPoolMax) {
			// both gas pools are full, so we should multiply the price by 119/120 for each second that elapses
			if price.Cmp(minPrice) <= 0 {
				// price is already at the minimum, so no need to iterate further
				_ = ps.SetGasPool(gasPoolMax)
				_ = ps.SetSmallGasPool(smallGasPoolMax)
				_ = ps.SetGasPriceWei(minPrice)
				return
			} else {
				if secondsLeft >= 83 {
					// price is cut in half every 83 seconds, when both gas pools are full
					price = util.BigMulByFrac(price, 1, 2)
					secondsLeft -= 83
				} else {
					price = util.BigMulByFrac(price, 119, 120)
					secondsLeft -= 1
				}
			}
		} else {
			gasPool = util.UpperBoundInt(gasPool+int64(speedLimit), gasPoolMax)
			smallGasPool = util.UpperBoundInt(smallGasPool+int64(speedLimit), smallGasPoolMax)
			clippedGasPool := util.LowerBoundInt(gasPool, 0)
			clippedSmallGasPool := util.LowerBoundInt(smallGasPool, 0)

			cross := util.BigAdd(
				util.BigMulByInt(maxSmallPoolAsBig, clippedGasPool),
				util.BigMulByInt(maxPoolAsBig, clippedSmallGasPool),
			)
			ratio := util.BigDiv(util.BigSub(maxPoolProduct, cross), maxPoolProduct)

			// no need to clip the price here, because we'll do that on exit from the loop
			price = util.BigMulByFrac(ratio, 121, 120)
			secondsLeft--
		}
	}

	if util.BigLessThan(price, minPrice) {
		price = minPrice
	}
	if util.BigGreaterThan(price, maxPrice) {
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

func (ps *L2PricingState) PerBlockGasLimit() (uint64, error) {
	pool, _ := ps.GasPool()
	maxLimit, err := ps.MaxPerBlockGasLimit()
	if pool < 0 || err != nil {
		return 0, err
	} else if uint64(pool) > maxLimit {
		return maxLimit, nil
	} else {
		return uint64(pool), nil
	}
}
