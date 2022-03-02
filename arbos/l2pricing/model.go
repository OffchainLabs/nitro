//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/util"
	"github.com/offchainlabs/arbstate/util/colors"
)

const InitialSpeedLimitPerSecond = 1000000
const InitialPerBlockGasLimit uint64 = 20 * 1000000
const InitialMinimumGasPriceWei = 1 * params.GWei
const InitialBaseFeeWei = InitialMinimumGasPriceWei
const InitialGasPoolSeconds = 10 * 60
const InitialRateEstimateInertia = 60
const InitialGasPoolTarget = 80 * 100 // 80% in bips
const InitialGasPoolVoice = 60 * 100  // 60% in bips

func (ps *L2PricingState) AddToGasPool(gas int64) {
	gasPool, _ := ps.GasPool()
	ps.Restrict(ps.SetGasPool(util.SaturatingAdd(gasPool, gas)))
}

// Update the pricing model with a finalized block's header
func (ps *L2PricingState) UpdatePricingModel(header *types.Header, timePassed uint64, debug bool) {

	// update the rate estimate, which is the weighted average of the past and present
	//     rate' = weighted average of the historical rate and the current
	//     rate' = (memory * rate + passed * recent) / (memory + passed)
	//     rate' = (memory * rate + used) / (memory + passed)
	//
	gasPool, _ := ps.GasPool()
	gasPoolLastBlock, _ := ps.GasPoolLastBlock()
	poolMax, _ := ps.GasPoolMax()
	gasPool = util.MinInt(gasPool, poolMax)
	gasPoolLastBlock = util.MinInt(gasPoolLastBlock, poolMax)
	gasUsed := uint64(gasPoolLastBlock - gasPool)
	rateSeconds, _ := ps.RateEstimateInertia()
	priorRate, _ := ps.RateEstimate()
	rate := util.SaturatingUAdd(util.SaturatingUMul(rateSeconds, priorRate), gasUsed) / (rateSeconds + timePassed)
	ps.SetRateEstimate(rate)

	// compute the rate ratio
	//     ratio = recent gas consumption rate / speed limit
	//
	speedLimit, _ := ps.SpeedLimitPerSecond()
	rateRatio := util.UfracToBigFloat(rate, speedLimit)

	// compute the pool fullness ratio & the updated gas pool
	//     ratio = max(0, 2 - (average fullness) / (target fullness))
	//     pool' = min(maximum, pool + speed * passed)
	//
	timeToFull := (poolMax - gasPool) / int64(speedLimit)
	bips := uint64(10000)
	var averagePool uint64
	var newGasPool int64
	if timePassed > uint64(timeToFull) {
		spaceBefore := uint64(poolMax - gasPool)
		averagePool = uint64(poolMax) - spaceBefore*spaceBefore/util.SaturatingUMul(2*speedLimit, timePassed)
		newGasPool = poolMax
	} else {
		averagePool = uint64(gasPool) + timePassed*speedLimit/2
		newGasPool = gasPool + int64(speedLimit*timePassed)
	}
	poolTarget, _ := ps.GasPoolTarget()
	poolTargetGas := poolTarget * uint64(poolMax) / bips
	poolRatio := util.UfracToBigFloat(0, 1)
	if averagePool < 2*poolTargetGas {
		poolRatio = util.UfracToBigFloat(2*poolTargetGas-averagePool, poolTargetGas)
	}

	// take the weighted average of the ratios, in basis points
	//      average = voice * pool + (1 - voice) * rate
	//
	poolVoice, _ := ps.GasPoolVoice()
	averageOfRatios, _ := util.BigAddFloat(
		util.BigMulFloatByUint(poolRatio, poolVoice),
		util.BigMulFloatByUint(rateRatio, bips-poolVoice),
	).Uint64()
	averageOfRatiosUnbounded := averageOfRatios
	if averageOfRatios > 2*bips {
		averageOfRatios = 2 * bips
	}

	// update the gas price, adjusting each second by the max allowed by EIP 1559
	//      price' = price * exp(seconds at intensity) / 2 mins
	//
	exp := int64(averageOfRatios-bips) * int64(timePassed) / 120 // limit to EIP 1559's max rate
	price := util.BigDivByUint(util.BigMulByUint(header.BaseFee, util.ApproxExpBasisPoints(exp)), bips)
	maxPrice := util.BigMulByInt(header.BaseFee, 2)
	minPrice, _ := ps.MinGasPriceWei()

	p := func(args ...interface{}) {
		if debug {
			colors.PrintGrey(args...)
		}
	}
	p("\nused\t", gasUsed, " in ", timePassed, "s = ", rate, "/s vs limit ", speedLimit, "/s for ", rateRatio)
	p("pool\t", gasPool, "/", poolMax, " ➤ ", averagePool, " ➤ ", newGasPool, " ", poolRatio)
	p("ratio\t", poolRatio, rateRatio, " ➤ ", averageOfRatiosUnbounded, "‱   bound to [0, 20000]")
	p("exp()\t", exp, " ➤ ", util.ApproxExpBasisPoints(exp), "‱  ")
	p("price\t", header.BaseFee, " ➤ ", price, " bound to [", minPrice, ", ", maxPrice, "]\n")

	if util.BigLessThan(price, minPrice) {
		price = minPrice
	}
	if util.BigGreaterThan(price, maxPrice) {
		log.Warn("ArbOS tried to 2x the price", "price", price, "bound", maxPrice)
		price = maxPrice
	}
	_ = ps.SetGasPriceWei(price)
	_ = ps.SetGasPool(newGasPool)
	ps.SetGasPoolLastBlock(newGasPool)
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
