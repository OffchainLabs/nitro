// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package arbmath

import "math/big"

type Bips int64

const OneInBips Bips = 10000

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
	return Bips(SaturatingCast(value))
}
