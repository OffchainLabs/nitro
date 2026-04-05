// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package util

import (
	"testing"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
)

func TestNormalizeL2GasForL1GasInitial(t *testing.T) {
	tests := []struct {
		name            string
		l2gas           uint64
		assumedL2Basefee uint64
		expected        uint64
	}{
		{
			name:            "exact match with initial base fee",
			l2gas:           1000000,
			assumedL2Basefee: l2pricing.InitialBaseFeeWei,
			expected:        1000000,
		},
		{
			name:            "assumed base fee higher than initial",
			l2gas:           1000000,
			assumedL2Basefee: l2pricing.InitialBaseFeeWei * 2,
			expected:        2000000,
		},
		{
			name:            "assumed base fee lower than initial",
			l2gas:           1000000,
			assumedL2Basefee: l2pricing.InitialBaseFeeWei / 2,
			expected:        500000,
		},
		{
			name:            "zero l2gas",
			l2gas:           0,
			assumedL2Basefee: l2pricing.InitialBaseFeeWei,
			expected:        0,
		},
		{
			name:            "large l2gas value",
			l2gas:           1000000000,
			assumedL2Basefee: l2pricing.InitialBaseFeeWei,
			expected:        1000000000,
		},
		{
			name:            "small assumed base fee",
			l2gas:           100,
			assumedL2Basefee: l2pricing.InitialBaseFeeWei / 100,
			expected:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeL2GasForL1GasInitial(tt.l2gas, tt.assumedL2Basefee)
			if result != tt.expected {
				t.Errorf("NormalizeL2GasForL1GasInitial(%d, %d) = %d, want %d",
					tt.l2gas, tt.assumedL2Basefee, result, tt.expected)
			}
		})
	}
}

func TestNormalizeL2GasForL1GasInitial_EdgeCases(t *testing.T) {
	t.Run("handles rounding in division", func(t *testing.T) {
		// Test that division rounds down as expected
		l2gas := uint64(1000)
		assumedL2Basefee := l2pricing.InitialBaseFeeWei + 1
		result := NormalizeL2GasForL1GasInitial(l2gas, assumedL2Basefee)

		// Since integer division rounds down, result should be slightly higher than l2gas
		// (1000 * (InitialBaseFeeWei + 1)) / InitialBaseFeeWei > 1000
		if result <= l2gas {
			t.Errorf("Expected result > %d when assumedL2Basefee > InitialBaseFeeWei, got %d",
				l2gas, result)
		}
	})

	t.Run("proportional scaling", func(t *testing.T) {
		// Verify that doubling assumed base fee doubles the result
		l2gas := uint64(500000)
		baseFee1 := l2pricing.InitialBaseFeeWei
		baseFee2 := l2pricing.InitialBaseFeeWei * 2

		result1 := NormalizeL2GasForL1GasInitial(l2gas, baseFee1)
		result2 := NormalizeL2GasForL1GasInitial(l2gas, baseFee2)

		if result2 != result1*2 {
			t.Errorf("Expected doubling base fee to double result: got %d and %d", result1, result2)
		}
	})
}
