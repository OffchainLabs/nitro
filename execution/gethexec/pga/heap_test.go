// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"math/big"
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
		boost:           new(uint64),
	}
	return PrioritizedTx[mockTx]{tx: tx, priority: priority}, resultChan
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
		if got := h.popConcrete(); got.priority != want {
			t.Fatalf("popConcrete priority = %d, want %d", got.priority, want)
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
			tx:       mockTx{id: id, ctx: context.Background(), firstAppearance: defaultArrival.Add(offset)},
			priority: 7,
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

func TestTxHeapPushBatch(t *testing.T) {
	var h txHeap[mockTx]
	seed, _ := newMockEntry(0, nil, 4)
	h.pushConcrete(seed)

	batch := make([]PrioritizedTx[mockTx], 0, 3)
	for i, p := range []uint64{8, 2, 6} {
		entry, _ := newMockEntry(i+1, nil, p)
		batch = append(batch, entry)
	}
	h.pushBatch(batch)

	if h.Len() != 4 {
		t.Fatalf("len = %d, want 4", h.Len())
	}
	for _, want := range []uint64{8, 6, 4, 2} {
		if got := h.popConcrete(); got.priority != want {
			t.Fatalf("popConcrete priority = %d, want %d", got.priority, want)
		}
	}
}

func TestTxHeapPushBatchEmptyKeepsHeapValid(t *testing.T) {
	var h txHeap[mockTx]
	seed, _ := newMockEntry(0, nil, 5)
	h.pushConcrete(seed)

	h.pushBatch(nil) // empty round must still leave a valid heap

	if h.Len() != 1 {
		t.Fatalf("len = %d, want 1", h.Len())
	}
	if got := h.popConcrete(); got.priority != 5 {
		t.Fatalf("priority = %d, want 5", got.priority)
	}
}

func TestTxHeapRekeyRecomputesPriority(t *testing.T) {
	var h txHeap[mockTx]

	// A is tip-bound (constant 10); B is cap-bound: 50 at base 10, 5 otherwise. Their order flips with the basefee.
	entryA, _ := newMockEntry(1, constFee(10), 10)
	entryB, _ := newMockEntry(2, func(baseFee *big.Int) (uint64, error) {
		if baseFee.Cmp(big.NewInt(10)) == 0 {
			return 50, nil
		}
		return 5, nil
	}, 50) // seeded with the stale base-10 priority
	h.pushConcrete(entryA)
	h.pushConcrete(entryB)

	h.rekey(big.NewInt(55)) // base 55: A=10, B=5 -> A on top

	if h.Len() != 2 {
		t.Fatalf("len = %d, want 2", h.Len())
	}
	if got := h.popConcrete(); got.tx.id != entryA.tx.id || got.priority != 10 {
		t.Fatalf("top = (id %d, prio %d), want A with prio 10", got.tx.id, got.priority)
	}
	if got := h.popConcrete(); got.tx.id != entryB.tx.id || got.priority != 5 {
		t.Fatalf("second = (id %d, prio %d), want B with prio 5", got.tx.id, got.priority)
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

	// Order is preserved (a > c > b), every key rose by 5, and each tx's accumulated boost rose to 5.
	for _, want := range []struct {
		id       int
		priority uint64
	}{{1, 35}, {3, 25}, {2, 15}} {
		got := h.popConcrete()
		if got.tx.id != want.id || got.priority != want.priority {
			t.Fatalf("pop = (id %d, prio %d), want (id %d, prio %d)", got.tx.id, got.priority, want.id, want.priority)
		}
		if *got.tx.boost != 5 {
			t.Fatalf("tx %d boost = %d, want 5", got.tx.id, *got.tx.boost)
		}
	}
}

func TestTxHeapRekeyDropsFeeCapTooLow(t *testing.T) {
	var h txHeap[mockTx]

	good, goodResult := newMockEntry(1, constFee(10), 10)
	bad, badResult := newMockEntry(2, failFee(errFeeCapTooLow), 20)
	h.pushConcrete(good)
	h.pushConcrete(bad)

	h.rekey(big.NewInt(40))

	expectResult(t, badResult, errFeeCapTooLow)
	expectNoResult(t, goodResult)
	if h.Len() != 1 {
		t.Fatalf("len = %d, want 1", h.Len())
	}
	if got := h.popConcrete(); got.tx.id != good.tx.id {
		t.Fatalf("survivor = %d, want good", got.tx.id)
	}
}
