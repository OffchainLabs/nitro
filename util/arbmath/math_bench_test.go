// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbmath

import (
	"math/rand"
	"testing"
)

// Benchmark tests for Log2Floor and Log2Ceil functions
func BenchmarkLog2Floor(b *testing.B) {
	// Test with various input sizes
	testCases := []struct {
		name string
		gen  func() uint64
	}{
		{"Small", func() uint64 { return uint64(rand.Intn(1000)) }},
		{"Medium", func() uint64 { return uint64(rand.Intn(1000000)) }},
		{"Large", func() uint64 { return uint64(rand.Intn(1000000000)) }},
		{"VeryLarge", func() uint64 { return rand.Uint64() }},
		{"PowersOfTwo", func() uint64 { return 1 << uint(rand.Intn(64)) }},
		{"Zero", func() uint64 { return 0 }},
		{"One", func() uint64 { return 1 }},
		{"MaxUint64", func() uint64 { return ^uint64(0) }},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Pre-generate test values to avoid timing the random generation
			values := make([]uint64, b.N)
			for i := range values {
				values[i] = tc.gen()
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Log2Floor(values[i])
			}
		})
	}
}

func BenchmarkLog2Ceil(b *testing.B) {
	// Test with various input sizes
	testCases := []struct {
		name string
		gen  func() uint64
	}{
		{"Small", func() uint64 { return uint64(rand.Intn(1000)) }},
		{"Medium", func() uint64 { return uint64(rand.Intn(1000000)) }},
		{"Large", func() uint64 { return uint64(rand.Intn(1000000000)) }},
		{"VeryLarge", func() uint64 { return rand.Uint64() }},
		{"PowersOfTwo", func() uint64 { return 1 << uint(rand.Intn(64)) }},
		{"Zero", func() uint64 { return 0 }},
		{"One", func() uint64 { return 1 }},
		{"MaxUint64", func() uint64 { return ^uint64(0) }},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Pre-generate test values to avoid timing the random generation
			values := make([]uint64, b.N)
			for i := range values {
				values[i] = tc.gen()
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Log2Ceil(values[i])
			}
		})
	}
}

// Benchmark comparison between Log2Floor and Log2Ceil
func BenchmarkLog2FloorVsCeil(b *testing.B) {
	values := make([]uint64, b.N)
	for i := range values {
		values[i] = rand.Uint64()
	}

	b.Run("Log2Floor", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Log2Floor(values[i])
		}
	})

	b.Run("Log2Ceil", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Log2Ceil(values[i])
		}
	})
}

// Benchmark the relationship between Log2Floor and Log2Ceil
func BenchmarkLog2FloorAndCeil(b *testing.B) {
	values := make([]uint64, b.N)
	for i := range values {
		values[i] = rand.Uint64()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		floor := Log2Floor(values[i])
		ceil := Log2Ceil(values[i])
		_ = floor
		_ = ceil
	}
}
