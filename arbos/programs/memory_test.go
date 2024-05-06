// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"math"
	"testing"

	"github.com/offchainlabs/nitro/util/arbmath"
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
		for pages < 128 {
			jump := arbmath.MinInt(jump, 128-pages)
			total += model.GasCost(jump, pages, pages)
			pages += jump
		}
		AssertEq(t, total, 31999998)
	}

	for jump := uint16(1); jump <= 128; jump++ {
		total := uint64(0)
		open := uint16(0)
		ever := uint16(0)
		adds := uint64(0)
		for ever < 128 {
			jump := arbmath.MinInt(jump, 128-open)
			total += model.GasCost(jump, open, ever)
			open += jump
			ever = arbmath.MaxInt(ever, open)

			if ever > model.freePages {
				adds += uint64(arbmath.MinInt(jump, ever-model.freePages))
			}

			// pretend we've deallocated some pages
			open -= jump / 2
		}
		expected := 31873998 + adds*uint64(model.pageGas)
		AssertEq(t, total, expected)
	}

	// check saturation
	AssertEq(t, math.MaxUint64, model.GasCost(129, 0, 0))
	AssertEq(t, math.MaxUint64, model.GasCost(math.MaxUint16, 0, 0))

	// check free pages
	model = NewMemoryModel(128, 1000)
	AssertEq(t, 0, model.GasCost(128, 0, 0))
	AssertEq(t, 0, model.GasCost(128, 0, 128))
	AssertEq(t, math.MaxUint64, model.GasCost(129, 0, 0))
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}

func AssertEq[T comparable](t *testing.T, a T, b T) {
	t.Helper()
	if a != b {
		Fail(t, a, "!=", b)
	}
}
