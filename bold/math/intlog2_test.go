// Copyright 2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package math

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/offchainlabs/nitro/bold/testing/casttest"
)

var benchResult int

func TestUnsingedIntegerLog2Floor(t *testing.T) {
	type log2TestCase struct {
		input    uint64
		expected int
	}

	testCases := []log2TestCase{
		{input: 1, expected: 0},
		{input: 2, expected: 1},
		{input: 4, expected: 2},
		{input: 6, expected: 2},
		{input: 8, expected: 3},
		{input: 24601, expected: 14},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d", tc.input), func(t *testing.T) {
			res := Log2Floor(tc.input)
			require.Equal(t, tc.expected, res)
		})
	}
}

func TestUnsingedIntegerLog2FloorPanicsOnZero(t *testing.T) {
	require.Panics(t, func() {
		Log2Floor(0)
	})
}

func FuzzUnsingedIntegerLog2Floor(f *testing.F) {
	testcases := []uint64{0, 2, 4, 6, 8}
	for _, tc := range testcases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, input uint64) {
		if input == 0 {
			require.Panics(t, func() {
				Log2Floor(input)
			})
			t.Skip()
		}
		r := Log2Floor(input)
		fr := math.Log2(float64(input))
		require.Equal(t, int(math.Floor(fr)), r)
	})
}

func BenchmarkUnsingedIntegerLog2Floor(b *testing.B) {
	var r int
	for i := 1; i < b.N; i++ {
		r = Log2Floor(casttest.ToUint64(b, i))
	}
	benchResult = r
}

func BenchmarkMathLog2Floor(b *testing.B) {
	var r int
	for i := 1; i < b.N; i++ {
		r = int(math.Log2(float64(i)))
	}
	benchResult = r
}

func TestUnsingedIntegerLog2Ceil(t *testing.T) {
	type log2TestCase struct {
		input    uint64
		expected int
	}

	testCases := []log2TestCase{
		{input: 1, expected: 0},
		{input: 2, expected: 1},
		{input: 4, expected: 2},
		{input: 6, expected: 3},
		{input: 8, expected: 3},
		{input: 24601, expected: 15},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d", tc.input), func(t *testing.T) {
			res := Log2Ceil(tc.input)
			require.Equal(t, tc.expected, res)
		})
	}
}

func TestUnsingedIntegerLog2CeilPanicsOnZero(t *testing.T) {
	require.Panics(t, func() {
		Log2Ceil(0)
	})
}

func FuzzUnsingedIntegerLog2Ceil(f *testing.F) {
	testcases := []uint64{0, 2, 4, 6, 8}
	for _, tc := range testcases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, input uint64) {
		if input == 0 {
			require.Panics(t, func() {
				Log2Ceil(input)
			})
			t.Skip()
		}
		r := Log2Ceil(input)
		fr := math.Log2(float64(input))
		require.Equal(t, int(math.Ceil(fr)), r)
	})
}

func BenchmarkUnsingedIntegerLog2Ceil(b *testing.B) {
	var r int
	for i := 1; i < b.N; i++ {
		r = Log2Ceil(casttest.ToUint64(b, i))
	}
	benchResult = r
}

func BenchmarkMathLog2Ceil(b *testing.B) {
	var r int
	for i := 1; i < b.N; i++ {
		r = int(math.Ceil(math.Log2(float64(i))))
	}
	benchResult = r
}
