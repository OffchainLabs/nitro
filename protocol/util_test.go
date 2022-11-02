package protocol

import (
	"errors"
	"testing"
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
		_, err := bisectionPoint(testCase.pre, testCase.post)
		if !errors.Is(err, ErrInvalid) {
			t.Fatal(testCase, err)
		}
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
		res, err := bisectionPoint(testCase.pre, testCase.post)
		if err != nil {
			t.Fatal(err, testCase)
		}
		if res != testCase.expected {
			t.Fatal(testCase, res)
		}
	}
}
