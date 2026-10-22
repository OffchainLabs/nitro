// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"testing"
	"time"
)

// newMockEntry builds a heap entry with the given priority key; the heap never recomputes fees, so the mock needs no
// fee func.
func newMockEntry(id int, priority uint64) mockTx {
	tx := mockTx{
		PGAState:        &PGAState{},
		id:              id,
		firstAppearance: defaultArrival,
	}
	tx.SetTip(priority)
	return tx
}

func TestTxHeapPushAndPopConcrete(t *testing.T) {
	var h txHeap[mockTx]
	for i, p := range []uint64{5, 1, 9, 3} {
		h.pushConcrete(newMockEntry(i, p))
	}
	for _, want := range []uint64{9, 5, 3, 1} {
		if h.Len() == 0 {
			t.Fatalf("heap emptied before priority %d", want)
		}
		if got := h.popConcrete(); got.GetPriority() != want {
			t.Fatalf("popConcrete priority = %d, want %d", got.GetPriority(), want)
		}
	}
	if h.Len() != 0 {
		t.Fatalf("heap not empty after draining: %d", h.Len())
	}
}

func TestTxHeapPopConcreteBreaksTiesByArrival(t *testing.T) {
	var h txHeap[mockTx]
	// Equal priority; the earlier firstAppearance must pop first regardless of push order.
	mk := func(id int, offset time.Duration) mockTx {
		tx := newMockEntry(id, 7)
		tx.firstAppearance = defaultArrival.Add(offset)
		return tx
	}
	h.pushConcrete(mk(3, 2*time.Millisecond))
	h.pushConcrete(mk(1, 0))
	h.pushConcrete(mk(2, 1*time.Millisecond))
	for _, wantID := range []int{1, 2, 3} {
		if got := h.popConcrete(); got.id != wantID {
			t.Fatalf("popConcrete id = %d, want %d", got.id, wantID)
		}
	}
}

func TestTxHeapApplyRoundBoundary(t *testing.T) {
	var h txHeap[mockTx]
	// Three entries with distinct priorities; the boundary lifts every key by the same delta and leaves the order intact.
	h.pushConcrete(newMockEntry(1, 30))
	h.pushConcrete(newMockEntry(2, 10))
	h.pushConcrete(newMockEntry(3, 20))

	h.applyRoundBoundary(5)

	// Order is preserved (1 > 3 > 2) and every key rose by 5.
	for _, want := range []struct {
		id       int
		priority uint64
	}{{1, 35}, {3, 25}, {2, 15}} {
		got := h.popConcrete()
		if got.id != want.id || got.GetPriority() != want.priority {
			t.Fatalf("pop = (id %d, prio %d), want (id %d, prio %d)", got.id, got.GetPriority(), want.id, want.priority)
		}
	}
}
