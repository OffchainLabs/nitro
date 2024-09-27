// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE.md

package math

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnsingedIntegerLog2(t *testing.T) {
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
			res := Log2(tc.input)
			require.Equal(t, tc.expected, res)
		})
	}
}

func TestUnsingedIntegerLog2PanicsOnZero(t *testing.T) {
	require.Panics(t, func() {
		Log2(0)
	})
}

func FuzzUnsingedIntegerLog2(f *testing.F) {
	testcases := []uint64{0, 2, 4, 6, 8}
	for _, tc := range testcases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, input uint64) {
		if input == 0 {
			require.Panics(t, func() {
				Log2(input)
			})
			t.Skip()
		}
		r := Log2(input)
		fr := math.Log2(float64(input))
		require.Equal(t, int(math.Floor(fr)), r)
	})
}

var benchResult int

func BenchmarkUnsingedIntegerLog2(b *testing.B) {
	var r int
	for i := 1; i < b.N; i++ {
		r = Log2(uint64(i))
	}
	benchResult = r
}

func BenchmarkMathLog2(b *testing.B) {
	var r int
	for i := 1; i < b.N; i++ {
		r = int(math.Log2(float64(i)))
	}
	benchResult = r
}
