// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// newMockEntry builds a heap entry around a mock tx with a working result channel, returning the entry and that
// channel so tests can assert what the heap reported. fee may be nil for tests that never recompute the priority.
func newMockEntry(id int, fee priorityFeeFunc, priority uint64) (PrioritizedTx[mockTx], chan error) {
	resultChan := make(chan error, 1)
	tx := mockTx{
		id:              id,
		fee:             fee,
		ctx:             context.Background(),
		firstAppearance: defaultArrival,
		resultChan:      resultChan,
		returnedResult:  &atomic.Bool{},
	}
	return PrioritizedTx[mockTx]{tx: tx, cachedPriority: priority}, resultChan
}

func TestTxHeapPushAndPopConcrete(t *testing.T) {
	var h txHeap[mockTx]
	for i, p := range []uint64{5, 1, 9, 3} {
		entry, _ := newMockEntry(i, nil, p)
		h.pushConcrete(entry)
	}
	for _, want := range []uint64{9, 5, 3, 1} {
		if h.Len() == 0 {
			t.Fatalf("heap emptied before priority %d", want)
		}
		if got := h.popConcrete(); got.cachedPriority != want {
			t.Fatalf("popConcrete priority = %d, want %d", got.cachedPriority, want)
		}
	}
	if h.Len() != 0 {
		t.Fatalf("heap not empty after draining: %d", h.Len())
	}
}

func TestTxHeapPopConcreteBreaksTiesByArrival(t *testing.T) {
	var h txHeap[mockTx]
	// Equal priority; the earlier firstAppearance must pop first regardless of push order.
	mk := func(id int, offset time.Duration) PrioritizedTx[mockTx] {
		return PrioritizedTx[mockTx]{
			tx:             mockTx{id: id, ctx: context.Background(), firstAppearance: defaultArrival.Add(offset)},
			cachedPriority: 7,
		}
	}
	h.pushConcrete(mk(3, 2*time.Millisecond))
	h.pushConcrete(mk(1, 0))
	h.pushConcrete(mk(2, 1*time.Millisecond))
	for _, wantID := range []int{1, 2, 3} {
		if got := h.popConcrete(); got.tx.id != wantID {
			t.Fatalf("popConcrete id = %d, want %d", got.tx.id, wantID)
		}
	}
}

func TestTxHeapAddBoost(t *testing.T) {
	var h txHeap[mockTx]
	// Three entries with distinct priorities; addBoost lifts every key by the same delta and leaves the order intact.
	a, _ := newMockEntry(1, constFee(30), 30)
	b, _ := newMockEntry(2, constFee(10), 10)
	c, _ := newMockEntry(3, constFee(20), 20)
	h.pushConcrete(a)
	h.pushConcrete(b)
	h.pushConcrete(c)

	h.addBoost(5)

	// Order is preserved (a > c > b), every key rose by 5, and each entry's accumulated boost rose to 5.
	for _, want := range []struct {
		id       int
		priority uint64
	}{{1, 35}, {3, 25}, {2, 15}} {
		got := h.popConcrete()
		if got.tx.id != want.id || got.cachedPriority != want.priority {
			t.Fatalf("pop = (id %d, prio %d), want (id %d, prio %d)", got.tx.id, got.cachedPriority, want.id, want.priority)
		}
		if got.boost != 5 {
			t.Fatalf("tx %d boost = %d, want 5", got.tx.id, got.boost)
		}
	}
}
