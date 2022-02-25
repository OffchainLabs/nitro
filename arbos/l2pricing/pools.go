//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/util"
)

const InitialSpeedLimitPerSecond = 1000000
const InitialPerBlockGasLimit uint64 = 20 * 1000000
const InitialMinimumGasPriceWei = 1 * params.GWei
const InitialBaseFeeWei = InitialMinimumGasPriceWei
const InitialGasPoolSeconds = 10 * 60
const InitialRateEstimateSeconds = 60
const InitialGasPoolTarget = 50
const InitialGasPoolVoice = 60 * 1000

func (ps *L2PricingState) AddToGasPools(gas int64) {
	gasPool, _ := ps.GasPool()
	ps.Restrict(ps.SetGasPool(util.SaturatingAdd(gasPool, gas)))
}

//
func (ps *L2PricingState) UpdatePricingModel(header *types.Header, timePassed uint64) {

	// update the rate estimate, which is the weighted average of the past and present
	//     rate' = weighted average of the historical rate and the current
	//     rate' = (memory * rate + passed * recent) / (memory + passed)
	//     rate' = (memory * rate + used) / (memory + passed)
	//
	memory, _ := ps.RateEstimateSeconds()
	priorRate, _ := ps.RateEstimate()
	rate := util.SaturatingUAdd(util.SaturatingUMul(memory, priorRate), header.GasUsed) / (memory + timePassed)
	ps.SetRateEstimate(rate)

	// compute the rate ratio
	//     ratio = recent gas consumption rate / speed limit
	//
	speedLimit, _ := ps.SpeedLimitPerSecond()
	rateRatio := util.UfracToBigFloat(rate, speedLimit)

	// compute the pool fullness ratio & update the gas pool
	//     ratio = max(0, 2 - (average fullness) / (target fullness))
	//
	gasPool, _ := ps.GasPool()
	poolMax, _ := ps.GasPoolMax()
	timeToFull := (poolMax - gasPool) / int64(speedLimit)
	var averagePool uint64
	if timePassed > uint64(timeToFull) {
		spaceBefore := uint64(poolMax - gasPool)
		averagePool = uint64(poolMax) - spaceBefore*spaceBefore/util.SaturatingUMul(2*speedLimit, timePassed)
		_ = ps.SetGasPool(poolMax)
	} else {
		averagePool = uint64(gasPool) + timePassed*speedLimit/2
		_ = ps.SetGasPool(gasPool + int64(speedLimit*timePassed))
	}
	poolTarget, _ := ps.GasPoolTarget()
	poolRatio := util.UfracToBigFloat(0, 1)
	if averagePool < 2*poolTarget*uint64(poolMax) {
		poolRatio = util.UfracToBigFloat(2*poolTarget*uint64(poolMax)-averagePool, poolTarget)
	}

	// take the weighted average of the ratios, in basis points
	//      average = voice * pool + (1 - voice) * rate
	//
	poolVoice, _ := ps.GasPoolVoice()
	averageOfRatios, _ := util.BigAddFloat(
		util.BigMulFloatByUint(poolRatio, poolVoice),
		util.BigMulFloatByUint(rateRatio, 10000-poolVoice),
	).Int64()
	if averageOfRatios > 20000 {
		averageOfRatios = 20000
	}

	// update the gas price, adjusting each second by the max allowed by EIP 1559
	//      price' = price * exp(seconds at intensity) / 2 mins
	//
	exp := (averageOfRatios - 10000) * int64(timePassed) / 120 // limit to EIP 1559's max rate
	price := util.BigDivByInt(util.BigMulByUint(header.BaseFee, util.ApproxExpBasisPoints(exp)), 10000)
	maxPrice := util.BigMulByInt(header.BaseFee, 2)
	minPrice, _ := ps.MinGasPriceWei()

	if util.BigLessThan(price, minPrice) {
		price = minPrice
	}
	if util.BigGreaterThan(price, maxPrice) {
		log.Warn("ArbOS tried to 2x the price", "price", price, "bound", maxPrice)
		price = maxPrice
	}
	_ = ps.SetGasPriceWei(price)
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
