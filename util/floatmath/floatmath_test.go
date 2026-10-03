// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package floatmath

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestFloatToBig(t *testing.T) {
	cases := []struct {
		input    float64
		expected *big.Int
	}{
		{0, big.NewInt(0)},
		{1.0, big.NewInt(1)},
		{-1.0, big.NewInt(-1)},
		{1.9, big.NewInt(1)},   // truncates toward zero
		{-1.9, big.NewInt(-1)}, // truncates toward zero
		{1e18, new(big.Int).SetUint64(1e18)},
		{math.NaN(), nil},
		{math.Inf(1), nil},
		{math.Inf(-1), nil},
	}

	for _, c := range cases {
		result := FloatToBig(c.input)
		if c.expected == nil {
			if result != nil {
				t.Errorf("FloatToBig(%v): expected nil, got %v", c.input, result)
			}
		} else {
			if result == nil {
				t.Errorf("FloatToBig(%v): expected %v, got nil", c.input, c.expected)
			} else if result.Cmp(c.expected) != 0 {
				t.Errorf("FloatToBig(%v): expected %v, got %v", c.input, c.expected, result)
			}
		}
	}
}

func TestBalancePerEther(t *testing.T) {
	ether := new(big.Int).SetUint64(uint64(params.Ether))
	cases := []struct {
		balance  *big.Int
		expected float64
	}{
		{big.NewInt(0), 0},
		{ether, 1.0},
		{new(big.Int).Mul(ether, big.NewInt(2)), 2.0},
		{new(big.Int).Div(ether, big.NewInt(4)), 0.25},
	}

	for _, c := range cases {
		result := BalancePerEther(c.balance)
		if result != c.expected {
			t.Errorf("BalancePerEther(%v): expected %v, got %v", c.balance, c.expected, result)
		}
	}
}

func TestWeiToGwei(t *testing.T) {
	gwei := new(big.Int).SetUint64(uint64(params.GWei))
	cases := []struct {
		value    *big.Int
		expected float64
	}{
		{big.NewInt(0), 0},
		{gwei, 1.0},
		{new(big.Int).Mul(gwei, big.NewInt(5)), 5.0},
		{new(big.Int).Div(gwei, big.NewInt(2)), 0.5},
	}

	for _, c := range cases {
		result := WeiToGwei(c.value)
		if result != c.expected {
			t.Errorf("WeiToGwei(%v): expected %v, got %v", c.value, c.expected, result)
		}
	}
}
