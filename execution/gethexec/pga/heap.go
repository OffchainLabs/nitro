// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"container/heap"
	"fmt"
)

// txHeap implements heap.Interface as a max-heap on priority, ties broken by earliest GetFirstAppearance. It owns the
// container/heap bookkeeping so Mempool never has to reach into the heap's internals.
type txHeap[T Tx] []PrioritizedTx[T]

func (h txHeap[T]) Len() int {
	return len(h)
}

func (h txHeap[T]) Less(i, j int) bool {
	if h[i].cachedPriority != h[j].cachedPriority {
		return h[i].cachedPriority > h[j].cachedPriority // max-heap: higher priority first
	}
	return h[i].tx.GetFirstAppearance().Before(h[j].tx.GetFirstAppearance()) // tie-break: earlier arrival
}

func (h txHeap[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push implements heap.Interface and is only meant to be called by container/heap (via heap.Push and heap.Init).
// Callers should use pushConcrete instead.
func (h *txHeap[T]) Push(x any) {
	item, ok := x.(PrioritizedTx[T])
	if !ok {
		panic(fmt.Sprintf("txHeap.Push: unexpected element type %T", x)) // impossible
	}
	*h = append(*h, item)
}

// Pop implements heap.Interface and is only meant to be called by container/heap (via heap.Pop). Callers should use
// popConcrete instead.
func (h *txHeap[T]) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = PrioritizedTx[T]{} // zero the slot so it doesn't pin the popped item's references
	*h = old[:n-1]
	return item
}

// pushConcrete adds entry and restores the heap invariant in O(log n). It is the typed entry point callers should use
// instead of the heap.Interface Push.
func (h *txHeap[T]) pushConcrete(entry PrioritizedTx[T]) {
	heap.Push(h, entry)
}

// popConcrete removes and returns the highest-priority entry, recovering the concrete type from heap.Pop. It is the
// typed entry point callers should use instead of the heap.Interface Pop. The caller must ensure the heap is non-empty.
func (h *txHeap[T]) popConcrete() PrioritizedTx[T] {
	entry, ok := heap.Pop(h).(PrioritizedTx[T])
	if !ok {
		panic("txHeap.popConcrete: unexpected heap element type") // impossible
	}
	return entry
}

// pushBatch appends entries and re-establishes the heap invariant in a single O(n) pass, cheaper than pushing one at a
// time when promoting a whole round. It always re-heapifies, so an empty batch still leaves a valid heap.
func (h *txHeap[T]) pushBatch(entries []PrioritizedTx[T]) {
	*h = append(*h, entries...)
	heap.Init(h)
}

// addBoost adds delta to every entry's accumulated boost and priority key, applying the anti-starvation boost to the
// whole queue. Adding the same delta to every key preserves the relative order, so the heap invariant holds without a
// re-heapify.
func (h txHeap[T]) addBoost(delta uint64) {
	for i := range h {
		h[i].addBoost(delta)
	}
}
