// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"testing"

	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestSizeFactorBips(t *testing.T) {
	tests := []struct {
		name          string
		txSize        int
		maxTxDataSize int
		exponent      uint
		want          arbmath.UBips
	}{
		{"exponent zero is one", 500, 1000, 0, 10000},
		{"exponent zero is one for a tx filling the block", 1000, 1000, 0, 10000},
		{"exponent zero is one for a non-positive max size", 10, 0, 0, 10000},
		{"negative size is clamped to zero", -5, 1000, 1, 10000},
		{"empty tx keeps the whole boost", 0, 1000, 1, 10000},
		{"tx filling the block gets nothing", 1000, 1000, 1, 0},
		{"tx larger than the block gets nothing", 1001, 1000, 1, 0},
		{"non-positive max size gets nothing", 10, 0, 1, 0},
		{"half-size tx, linear", 500, 1000, 1, 5000},
		{"half-size tx, squared", 500, 1000, 2, 2500},
		{"quarter-size tx, cubed", 250, 1000, 3, 4218}, // 7500 -> 5625 -> 4218, truncated at each step
		{"tiny tx in a default-size block", 10, 95000, 1, 9998},
		{"tiny tx in a default-size block, cubed", 10, 95000, 3, 9994},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SizeFactorBips(tt.txSize, tt.maxTxDataSize, tt.exponent); got != tt.want {
				t.Fatalf("SizeFactorBips(%d, %d, %d) = %d, want %d", tt.txSize, tt.maxTxDataSize, tt.exponent, got, tt.want)
			}
		})
	}
}
