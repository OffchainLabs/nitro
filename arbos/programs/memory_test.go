// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"math"
	"testing"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestTables(t *testing.T) {
	model := NewMemoryModel(2, 1000)
	base := math.Exp(math.Log(31_874_000) / 128)
	for p := uint16(0); p < 129; p++ {
		value := uint64(math.Pow(base, float64(p)))
		correct := model.exp(p)

		if value != correct {
			Fail(t, "wrong value for ", p, value, correct)
		}
	}
	if model.exp(129) != math.MaxUint64 || model.exp(math.MaxUint16) != math.MaxUint64 {
		Fail(t)
	}
}

func TestModel(t *testing.T) {
	model := NewMemoryModel(2, 1000)

	for jump := uint16(1); jump <= 128; jump++ {
		total := uint64(0)
		pages := uint16(0)
		for pages+jump < 128 {
			total += model.GasCost(jump, pages, pages)
			pages += jump
		}
		total += model.GasCost(128-pages, pages, pages)
		if total != 31999998 {
			Fail(t, "wrong total", total)
		}
	}
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
