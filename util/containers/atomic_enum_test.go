// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package containers

import "testing"

type testEnum int32

const (
	testEnumZero testEnum = iota
	testEnumOne
	testEnumTwo
)

func TestAtomicEnum(t *testing.T) {
	var e AtomicEnum[testEnum]
	if got := e.Load(); got != testEnumZero {
		t.Fatalf("Load() = %v, want %v", got, testEnumZero)
	}
	e.Store(testEnumOne)
	if got := e.Load(); got != testEnumOne {
		t.Fatalf("Load() after Store = %v, want %v", got, testEnumOne)
	}
	if got := e.Swap(testEnumTwo); got != testEnumOne {
		t.Fatalf("Swap() = %v, want %v", got, testEnumOne)
	}
	if got := e.Load(); got != testEnumTwo {
		t.Fatalf("Load() after Swap = %v, want %v", got, testEnumTwo)
	}
}
