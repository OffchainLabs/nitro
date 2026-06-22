// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package pga implements the priority mempool that backs the sequencer's Priority Gas Auction.
package pga

import (
	"container/heap"
	"fmt"
	"math/big"
)

// txHeap implements heap.Interface as a max-heap on priority, ties broken by earliest GetFirstAppearance.
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

func (h *txHeap[T]) Push(x any) {
	item, ok := x.(prioritizedTx[T])
	if !ok {
		panic(fmt.Sprintf("txHeap.Push: unexpected element type %T", x)) // impossible
	}
	*h = append(*h, item)
}

func (h *txHeap[T]) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = prioritizedTx[T]{} // zero the slot so it doesn't pin the popped item's references
	*h = old[:n-1]
	return item
}

// Mempool is a two-stage mempool for PGA. The first stage is a channel with the waiting list and the second stage is a
// priority queue. It is not safe for concurrent use; every method runs on the block-production goroutine.
type Mempool[T Tx] struct {
	txQueue       <-chan T  // stage one: the waiting list
	heap          txHeap[T] // stage two: the priority queue
	baseFee       *big.Int  // basefee of the block under construction
	maxTxDataSize int       // max promoted-transaction size, for the block under construction
}

func NewMempool[T Tx](txQueue <-chan T) *Mempool[T] {
	return &Mempool[T]{txQueue: txQueue}
}

func (m *Mempool[T]) Len() int {
	return len(m.heap)
}

// AreThereTxsForNextRound reports whether the next PGA round would have anything to work with.
func (m *Mempool[T]) AreThereTxsForNextRound() bool {
	return len(m.heap) != 0 || len(m.txQueue) != 0
}

// StartNewBlock begins a block: it records the block's basefee and max transaction size, then re-keys the queued
// transactions against the new basefee, dropping any whose fee cap fell below it. It finishes by calling
// StartNewPGARound, which promotes the waiting list and re-establishes the heap, so it doubles as the block's first PGA
// round.
func (m *Mempool[T]) StartNewBlock(baseFee *big.Int, maxTxDataSize int) {
	m.baseFee = baseFee
	m.maxTxDataSize = maxTxDataSize

	// Re-key the queued transactions against the new basefee, compacting in place.
	kept := 0
	for _, entry := range m.heap {
		if !entry.setPriority(m.baseFee) {
			continue
		}
		m.heap[kept] = entry
		kept++
	}
	// Clear the vacated tail so dropped or moved entries aren't pinned.
	for i := kept; i < len(m.heap); i++ {
		m.heap[i] = prioritizedTx[T]{}
	}
	m.heap = m.heap[:kept]
	// Run the block's first PGA round, which promotes the waiting list and re-establishes the heap after the re-key.
	m.StartNewPGARound()
}

// StartNewPGARound promotes a snapshot of the waiting list into the priority queue, then re-establishes the heap.
func (m *Mempool[T]) StartNewPGARound() {
	// n (the waiting-list length) is captured once; we are the sole consumer, so these receivers never block, and
	// arrivals after the snapshot stay buffered for the next round.
	n := len(m.txQueue)
	for range n {
		entry := prioritizedTx[T]{tx: <-m.txQueue}
		if !entry.setPriority(m.baseFee) {
			continue
		}
		m.heap = append(m.heap, entry)
	}
	// Always re-establish the heap, even for an empty round: StartNewBlock re-keys the queued transactions and then
	// relies on this call to restore the heap invariant.
	heap.Init(&m.heap)
}

// Pop removes and returns the highest-priority valid transaction. It validates each candidate against the block's max
// transaction size and context, dropping those that fail.
func (m *Mempool[T]) Pop() (tx T, ok bool) {
	for m.heap.Len() > 0 {
		entry, valid := heap.Pop(&m.heap).(prioritizedTx[T])
		if !valid {
			panic("Mempool.Pop: unexpected heap element type") // impossible
		}
		if entry.validate(m.maxTxDataSize) {
			return entry.tx, true
		}
	}
	return tx, false
}

// Push re-inserts a transaction popped from the queue, re-keying it against the block's basefee.
func (m *Mempool[T]) Push(item T) {
	entry := prioritizedTx[T]{tx: item}
	if !entry.setPriority(m.baseFee) {
		return
	}
	heap.Push(&m.heap, entry)
}
