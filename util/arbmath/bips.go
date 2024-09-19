// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbmath

import "math/big"

type Bips int64
type UBips uint64

const OneInBips Bips = 10000
const OneInUBips UBips = 10000

func NaturalToBips(natural int64) Bips {
	return Bips(SaturatingMul(natural, int64(OneInBips)))
}

func PercentToBips(percentage int64) Bips {
	return Bips(SaturatingMul(percentage, 100))
}

func BigToBips(natural *big.Int) Bips {
	return Bips(natural.Int64())
}

func BigMulByBips(value *big.Int, bips Bips) *big.Int {
	return BigMulByFrac(value, int64(bips), int64(OneInBips))
}

func BigMulByUBips(value *big.Int, bips UBips) *big.Int {
	return BigMulByUFrac(value, uint64(bips), uint64(OneInUBips))
}

func IntMulByBips(value int64, bips Bips) int64 {
	return value * int64(bips) / int64(OneInBips)
}

// UintMulByBips multiplies a uint value by a bips value
// bips must be positive and not cause an overflow
func UintMulByBips(value uint64, bips Bips) uint64 {
	// #nosec G115
	return value * uint64(bips) / uint64(OneInBips)
}

// UintSaturatingMulByBips multiplies a uint value by a bips value,
// saturating at the maximum bips value (not the maximum uint64 result),
// then rounding down and returning a uint64.
// Returns 0 if bips is less than or equal to zero
func UintSaturatingMulByBips(value uint64, bips Bips) uint64 {
	if bips <= 0 {
		return 0
	}
	// #nosec G115
	return SaturatingUMul(value, uint64(bips)) / uint64(OneInBips)
}

func SaturatingCastToBips(value uint64) Bips {
	return Bips(SaturatingCast[int64](value))
}

// BigDivToBips returns dividend/divisor as bips, saturating if out of bounds
func BigDivToBips(dividend, divisor *big.Int) Bips {
	value := BigMulByInt(dividend, int64(OneInBips))
	value.Div(value, divisor)
	return Bips(BigToIntSaturating(value))
}
