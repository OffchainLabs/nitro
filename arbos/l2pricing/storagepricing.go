package l2pricing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/math"
	"math/big"
)

const (
	UnitsPerStorage      = 1000000 // make storage tracing fixed-point, to reduce roundoff error
	FastAvgSeconds       = 5 * 60
	SlowAvgSeconds       = 12 * 60 * 60
	TargetAllocRateCells = 10
)

func (pricingState *L2PricingState) NotifyStorageUsageChange(delta int64) {
	pricingState.fastStorageAvg.Set(clippedAdd(pricingState.fastStorageAvg.Get(), delta*UnitsPerStorage))
	pricingState.slowStorageAvg.Set(clippedAdd(pricingState.slowStorageAvg.Get(), delta*UnitsPerStorage))
}

func (pricingState *L2PricingState) updateStorageComponentForElapsedTime(secondsElapsed uint64, price *big.Int) *big.Int {
	fastAvg := pricingState.fastStorageAvg.Get()
	slowAvg := pricingState.slowStorageAvg.Get()
	minPrice := big.NewInt(MinimumGasPriceWei)
	numeratorBase := big.NewInt(119 * UnitsPerStorage)
	denominator := big.NewInt(120 * UnitsPerStorage)

	for secondsElapsed > 0 {
		if fastAvg == 0 && slowAvg == 0 && price.Cmp(minPrice) <= 0 {
			pricingState.fastStorageAvg.Set(0)
			pricingState.slowStorageAvg.Set(0)
			return minPrice
		}

		fastAvg = fastAvg * (FastAvgSeconds - 1) / FastAvgSeconds
		slowAvg = slowAvg * (SlowAvgSeconds - 1) / SlowAvgSeconds

		fastRatio := fastAvg / (TargetAllocRateCells * FastAvgSeconds)
		slowRatio := slowAvg / (TargetAllocRateCells * SlowAvgSeconds)
		fmt.Println("fastRatio=", fastRatio, ", slowRatio=", slowRatio)
		ratio := (fastRatio + slowRatio) / 2
		if ratio > 2*UnitsPerStorage {
			ratio = 2 * UnitsPerStorage
		}
		// ratio == 0 means min storage usage; ratio == 2 * UnitsPerStorage means max storage usage

		price = new(big.Int).Div(
			new(big.Int).Mul(
				price,
				new(big.Int).Add(numeratorBase, new(big.Int).SetUint64(ratio)),
			),
			denominator,
		)
		// price might now be less than min, but if so it will stay below min and we'll fix it up before returning

		secondsElapsed--
	}

	pricingState.fastStorageAvg.Set(fastAvg)
	pricingState.slowStorageAvg.Set(slowAvg)
	if price.Cmp(minPrice) < 0 {
		price = minPrice
	}
	return price
}

func clippedAdd(u uint64, i int64) uint64 {
	if i >= 0 {
		sum := u + uint64(i)
		if sum < u {
			return math.MaxUint64
		}
		return sum
	} else {
		if u < uint64(-i) {
			return 0
		}
		return u - uint64(i)
	}
}
