// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"container/heap"
	"fmt"
)

// txHeap implements heap.Interface as a max-heap on priority, ties broken by earliest GetFirstAppearance. It owns the
// container/heap bookkeeping so Mempool never has to reach into the heap's internals.
type txHeap[T Tx] []T

func (h txHeap[T]) Len() int {
	return len(h)
}

func (h txHeap[T]) Less(i, j int) bool {
	if h[i].GetPriority() != h[j].GetPriority() {
		return h[i].GetPriority() > h[j].GetPriority() // max-heap: higher priority first
	}
	return h[i].GetFirstAppearance().Before(h[j].GetFirstAppearance()) // tie-break: earlier arrival
}

func (h txHeap[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push implements heap.Interface and is only meant to be called by container/heap (via heap.Push and heap.Init).
// Callers should use pushConcrete instead.
func (h *txHeap[T]) Push(x any) {
	item, ok := x.(T)
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
	var empty T
	old[n-1] = empty // zero the slot so it doesn't pin the popped item's references
	*h = old[:n-1]
	return item
}

// pushConcrete adds entry and restores the heap invariant in O(log n). It is the typed entry point callers should use
// instead of the heap.Interface Push.
func (h *txHeap[T]) pushConcrete(entry T) {
	heap.Push(h, entry)
}

// popConcrete removes and returns the highest-priority entry, recovering the concrete type from heap.Pop. It is the
// typed entry point callers should use instead of the heap.Interface Pop. The caller must ensure the heap is non-empty.
func (h *txHeap[T]) popConcrete() T {
	entry, ok := heap.Pop(h).(T)
	if !ok {
		panic("txHeap.popConcrete: unexpected heap element type") // impossible
	}
	return entry
}

// pushBatch appends entries and re-establishes the heap invariant in a single O(n) pass, cheaper than pushing one at a
// time when promoting a whole round. It always re-heapifies, so an empty batch still leaves a valid heap.
func (h *txHeap[T]) pushBatch(entries []T) {
	*h = append(*h, entries...)
	heap.Init(h)
}

// addBoost adds delta to every entry's accumulated boost and priority key, applying the anti-starvation boost to the
// whole queue. Adding the same delta to every key preserves the relative order, so the heap invariant holds without a
// re-heapify.
func (h txHeap[T]) addBoost(delta uint64) {
	for i := range h {
		h[i].AddBoost(delta)
	}
}

// takeRemaining returns all remaining transactions in the heap.
func (h txHeap[T]) takeRemaining() []T {
	return h
}
