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

// BalancePerEther returns balance per ether.
func BalancePerEther(balance *big.Int) float64 {
	balancePerEther, _ := new(big.Float).Quo(new(big.Float).SetInt(balance), new(big.Float).SetFloat64(params.Ether)).Float64()
	return balancePerEther
}

// WeiToGwei returns the gwei representation of the value.
func WeiToGwei(value *big.Int) float64 {
	gwei, _ := new(big.Float).Quo(new(big.Float).SetInt(value), new(big.Float).SetFloat64(params.GWei)).Float64()
	return gwei
}
