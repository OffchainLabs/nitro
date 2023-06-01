package math

import (
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/stretchr/testify/require"
)

func TestBisectionPoint(t *testing.T) {
	type bpTestCase struct {
		pre      uint64
		post     uint64
		expected uint64
	}

	errorTestCases := []bpTestCase{
		{12, 13, 0},
		{13, 9, 0},
	}
	for _, testCase := range errorTestCases {
		_, err := Bisect(testCase.pre, testCase.post)
		require.ErrorIs(t, err, ErrUnableToBisect, testCase)
	}
	testCases := []bpTestCase{
		{0, 2, 1},
		{1, 3, 2},
		{31, 33, 32},
		{32, 34, 33},
		{13, 15, 14},
		{0, 9, 8},
		{0, 13, 8},
		{0, 15, 8},
		{13, 17, 16},
		{13, 31, 16},
		{15, 31, 16},
	}
	for _, testCase := range testCases {
		res, err := Bisect(testCase.pre, testCase.post)
		require.NoError(t, err, testCase)
		require.Equal(t, testCase.expected, res)
	}
}

func TestMin(t *testing.T) {
	type testCase[T Unsigned] struct {
		items      []T
		wanted     T
		wantedNone bool
	}
	testCases := []testCase[uint64]{
		{
			items:      []uint64{},
			wantedNone: true,
		},
		{
			items:  []uint64{1},
			wanted: 1,
		},
		{
			items:  []uint64{1, 2, 3},
			wanted: 1,
		},
		{
			items:  []uint64{32, 333, 202, 11, 3, 5, 1000},
			wanted: 3,
		},
	}
	for _, tt := range testCases {
		res := Min(tt.items)
		if tt.wantedNone {
			require.Equal(t, option.None[uint64](), res)
		} else {
			require.Equal(t, tt.wanted, res.Unwrap())
		}
	}
}

func TestMax(t *testing.T) {
	type testCase[T Unsigned] struct {
		items      []T
		wanted     T
		wantedNone bool
	}
	testCases := []testCase[uint64]{
		{
			items:      []uint64{},
			wantedNone: true,
		},
		{
			items:  []uint64{1},
			wanted: 1,
		},
		{
			items:  []uint64{1, 2, 3},
			wanted: 3,
		},
		{
			items:  []uint64{32, 333, 202, 11, 3, 5, 1000},
			wanted: 1000,
		},
	}
	for _, tt := range testCases {
		res := Max(tt.items)
		if tt.wantedNone {
			require.Equal(t, option.None[uint64](), res)
		} else {
			require.Equal(t, tt.wanted, res.Unwrap())
		}
	}
}
