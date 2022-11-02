package protocol

import (
	"errors"
	"testing"
)

func TestOldBisectionPoint(t *testing.T) {
	_, err := oldBisectionPointAlgorithm(13, 9)
	if err == nil {
		t.Fatal()
	}
	_, err = oldBisectionPointAlgorithm(13, 14)
	if err == nil {
		t.Fatal()
	}

	lo := uint64(26)
	hi := uint64(45)
	expected := []uint64{44, 40, 32, 28, 27}
	for len(expected) > 0 {
		bp, err := oldBisectionPointAlgorithm(lo, hi)
		if err != nil {
			t.Fatal(lo, hi, err)
		}
		if bp != expected[0] {
			t.Fatal(lo, hi, bp, expected[0])
		}
		hi = bp
		expected = expected[1:]
	}
}

type bpTestCase struct {
	pre      uint64
	post     uint64
	expected uint64
}

func TestBisectionPoint(t *testing.T) {
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
