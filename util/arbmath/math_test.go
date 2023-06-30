// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package arbmath

import (
	"math"
	"math/rand"
	"testing"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestMath(t *testing.T) {
	cases := []uint64{0, 1, 2, 3, 4, 7, 13, 28, 64}
	correctPower := []uint64{1, 2, 4, 4, 8, 8, 16, 32, 128}
	correctLog := []uint64{0, 1, 2, 2, 3, 3, 4, 5, 7}

	for i := 0; i < len(cases); i++ {
		calculated := NextPowerOf2(cases[i])
		if calculated != correctPower[i] {
			Fail(t, "expected power", correctPower[i], "but got", calculated)
		}
		calculated = Log2ceil(cases[i])
		if calculated != correctLog[i] {
			Fail(t, "expected log", correctLog[i], "but got", calculated)
		}
	}

	// try large random sqrts
	for i := 0; i < 100000; i++ {
		input := rand.Uint64() / 256
		approx := ApproxSquareRoot(input)
		correct := math.Sqrt(float64(input))
		diff := int(approx) - int(correct)
		if diff < -1 || diff > 1 {
			Fail(t, "sqrt approximation off by too much", diff, input, approx, correct)
		}
	}

	// try the first million sqrts
	for i := 0; i < 1000000; i++ {
		input := uint64(i)
		approx := ApproxSquareRoot(input)
		correct := math.Sqrt(float64(input))
		diff := int(approx) - int(correct)
		if diff < 0 || diff > 1 {
			Fail(t, "sqrt approximation off by too much", diff, input, approx, correct)
		}
	}

	// try powers of 2
	for i := 0; i < 63; i++ {
		input := uint64(1 << i)
		approx := ApproxSquareRoot(input)
		correct := math.Sqrt(float64(input))
		diff := int(approx) - int(correct)
		if diff != 0 {
			Fail(t, "incorrect", "2^", i, diff, approx, correct)
		}
	}
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
