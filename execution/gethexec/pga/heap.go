// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"container/heap"
	"fmt"
	"math/big"
)

// txHeap implements heap.Interface as a max-heap on priority, ties broken by earliest GetFirstAppearance. It owns the
// container/heap bookkeeping so Mempool never has to reach into the heap's internals.
type txHeap[T Tx] []prioritizedTx[T]

func (h txHeap[T]) Len() int {
	return len(h)
}

func (h txHeap[T]) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority > h[j].priority // max-heap: higher priority first
	}
	return h[i].tx.GetFirstAppearance().Before(h[j].tx.GetFirstAppearance()) // tie-break: earlier arrival
}

func (h txHeap[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push implements heap.Interface and is only meant to be called by container/heap (via heap.Push and heap.Init).
// Callers should use pushConcrete or pushBatch instead.
func (h *txHeap[T]) Push(x any) {
	item, ok := x.(prioritizedTx[T])
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
	old[n-1] = prioritizedTx[T]{} // zero the slot so it doesn't pin the popped item's references
	*h = old[:n-1]
	return item
}

// pushConcrete adds entry and restores the heap invariant in O(log n). It is the typed entry point callers should use
// instead of the heap.Interface Push.
func (h *txHeap[T]) pushConcrete(entry prioritizedTx[T]) {
	heap.Push(h, entry)
}

// popConcrete removes and returns the highest-priority entry, recovering the concrete type from heap.Pop. It is the
// typed entry point callers should use instead of the heap.Interface Pop. The caller must ensure the heap is non-empty.
func (h *txHeap[T]) popConcrete() prioritizedTx[T] {
	entry, ok := heap.Pop(h).(prioritizedTx[T])
	if !ok {
		panic("txHeap.popConcrete: unexpected heap element type") // impossible
	}
	return entry
}

// rekey recomputes every entry's priority against baseFee, dropping those whose fee cap fell below it, then
// re-establishes the heap invariant.
func (h *txHeap[T]) rekey(baseFee *big.Int) {
	kept := 0
	for _, entry := range *h {
		if !entry.setPriority(baseFee) {
			continue
		}
		(*h)[kept] = entry
		kept++
	}
	// Clear the vacated tail so dropped or moved entries aren't pinned.
	for i := kept; i < len(*h); i++ {
		(*h)[i] = prioritizedTx[T]{}
	}
	*h = (*h)[:kept]
	heap.Init(h)
}

// pushBatch appends entries and re-establishes the heap invariant in a single O(n) pass, cheaper than pushing one at a
// time when promoting a whole round. It always re-heapifies, so an empty batch still leaves a valid heap.
func (h *txHeap[T]) pushBatch(entries []prioritizedTx[T]) {
	*h = append(*h, entries...)
	heap.Init(h)
}
