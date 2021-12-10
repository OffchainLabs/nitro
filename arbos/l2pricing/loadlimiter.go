package l2pricing

import (
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"
)

type LoadLimiter struct {
	sto         *storage.Storage
	fastAvg     *storage.StorageBackedUint64
	slowAvg     *storage.StorageBackedUint64
	constParams *loadLimiterConstParams // these must be constants (because they're not stored in the statedb)
}

type loadLimiterConstParams struct {
	fixedPointMul uint64
	FastSeconds   uint64
	SlowSeconds   uint64
	TargetRate    uint64
}

var StorageLimiterParams *loadLimiterConstParams

func init() {
	StorageLimiterParams = &loadLimiterConstParams{
		fixedPointMul: 1000000,
		FastSeconds:   5 * 60,
		SlowSeconds:   12 * 60 * 60,
		TargetRate:    10,
	}
}

const (
	fastOffset uint64 = iota
	slowOffset
)

func OpenLoadLimiter(sto *storage.Storage, constParams *loadLimiterConstParams) *LoadLimiter {
	return &LoadLimiter{
		sto:         sto,
		fastAvg:     sto.OpenStorageBackedUint64(util.UintToHash(fastOffset)),
		slowAvg:     sto.OpenStorageBackedUint64(util.UintToHash(slowOffset)),
		constParams: constParams,
	}
}

func (limiter *LoadLimiter) NotifyUsageChange(delta int64) {
	fpmul := limiter.constParams.fixedPointMul
	limiter.fastAvg.Set(clippedAdd(limiter.fastAvg.Get(), delta*int64(fpmul)))
	limiter.slowAvg.Set(clippedAdd(limiter.slowAvg.Get(), delta*int64(fpmul)))
}

func (limiter *LoadLimiter) updateStorageComponentForElapsedTime(secondsElapsed uint64, price *big.Int, minPrice *big.Int) *big.Int {
	fpmul := limiter.constParams.fixedPointMul
	fastAvg := limiter.fastAvg.Get()
	slowAvg := limiter.slowAvg.Get()
	numeratorBase := big.NewInt(int64(119 * fpmul))
	denominator := big.NewInt(int64(120 * fpmul))

	for secondsElapsed > 0 {
		if fastAvg == 0 && slowAvg == 0 && price.Cmp(minPrice) <= 0 {
			limiter.fastAvg.Set(0)
			limiter.slowAvg.Set(0)
			return minPrice
		}

		fastAvg = fastAvg * (limiter.constParams.FastSeconds - 1) / limiter.constParams.FastSeconds
		slowAvg = slowAvg * (limiter.constParams.SlowSeconds - 1) / limiter.constParams.SlowSeconds

		fastRatio := fastAvg / (limiter.constParams.TargetRate * limiter.constParams.FastSeconds)
		slowRatio := slowAvg / (limiter.constParams.TargetRate * limiter.constParams.SlowSeconds)
		ratio := (fastRatio + slowRatio) / 2
		if ratio > 2*fpmul {
			ratio = 2 * fpmul
		}
		// ratio == 0 means min usage; ratio == 2 * fpmul means max usage

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

	limiter.fastAvg.Set(fastAvg)
	limiter.slowAvg.Set(slowAvg)
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
