//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package util

import (
	"testing"

	"github.com/offchainlabs/arbstate/util/testhelpers"
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

}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
