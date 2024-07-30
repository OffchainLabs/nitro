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
	return Bips(natural.Uint64())
}

func BigMulByBips(value *big.Int, bips Bips) *big.Int {
	return BigMulByFrac(value, int64(bips), int64(OneInBips))
}

func IntMulByBips(value int64, bips Bips) int64 {
	return value * int64(bips) / int64(OneInBips)
}

func UintMulByBips(value uint64, bips Bips) uint64 {
	return value * uint64(bips) / uint64(OneInBips)
}

func SaturatingCastToBips(value uint64) Bips {
	return Bips(SaturatingCast[int64](value))
}

func (bips UBips) Uint64() uint64 {
	return uint64(bips)
}

func (bips Bips) Uint64() uint64 {
	return uint64(bips)
}

// BigDivToBips returns dividend/divisor as bips, saturating if out of bounds
func BigDivToBips(dividend, divisor *big.Int) Bips {
	value := BigMulByInt(dividend, int64(OneInBips))
	value.Div(value, divisor)
	return Bips(BigToUintSaturating(value))
}
