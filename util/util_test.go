package util

import (
	"testing"

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
		_, err := BisectionPoint(testCase.pre, testCase.post)
		require.ErrorIs(t, err, ErrUnableToBisect, testCase)
	}
	testCases := []bpTestCase{
		{0, 2, 1},
		{1, 3, 2},
		{31, 33, 32},
		{32, 34, 33},
		{13, 15, 14},
		{0, 9, 7},
		{0, 13, 7},
		{0, 15, 7},
		{13, 17, 15},
		{13, 31, 15},
		{15, 31, 15},
	}
	for _, testCase := range testCases {
		res, err := BisectionPoint(testCase.pre, testCase.post)
		require.NoError(t, err, testCase)
		require.Equal(t, testCase.expected, res)
	}
}
