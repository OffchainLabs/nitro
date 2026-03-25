// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package floatmath

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

// FloatToBig casts a float to a huge
// Returns nil when passed NaN or Infinity
func FloatToBig(value float64) *big.Int {
	if math.IsNaN(value) {
		return nil
	}
	result, _ := new(big.Float).SetFloat64(value).Int(nil)
	return result
}

// UintToBigFloat casts a uint to a big float
func UintToBigFloat(value uint64) *big.Float {
	return new(big.Float).SetPrec(53).SetUint64(value)
}

// UfracToBigFloat casts a rational to a big float
func UfracToBigFloat(numerator, denominator uint64) *big.Float {
	float := new(big.Float)
	float.Quo(UintToBigFloat(numerator), UintToBigFloat(denominator))
	return float
}

// BigAddFloat add two big floats together
func BigAddFloat(augend, addend *big.Float) *big.Float {
	return new(big.Float).Add(augend, addend)
}

// BigMulFloat multiply a big float by another
func BigMulFloat(multiplicand, multiplier *big.Float) *big.Float {
	return new(big.Float).Mul(multiplicand, multiplier)
}

// BigFloatMulByUint multiply a big float by an unsigned integer
func BigFloatMulByUint(multiplicand *big.Float, multiplier uint64) *big.Float {
	return new(big.Float).Mul(multiplicand, UintToBigFloat(multiplier))
}

// SquareFloat returns square of float
func SquareFloat(value float64) float64 {
	return value * value
}

// BalancePerEther returns balance per ether.
func BalancePerEther(balance *big.Int) float64 {
	balancePerEther, _ := new(big.Float).Quo(new(big.Float).SetInt(balance), new(big.Float).SetFloat64(params.Ether)).Float64()
	return balancePerEther
}
