// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package pga implements the priority mempool that backs the sequencer's Priority Gas Auction.
package pga

import (
	"container/heap"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/txpool"
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
// transactions against the new basefee, dropping those whose context expired, whose size exceeds the (hot-reloadable)
// max transaction size, or whose fee cap fell below the basefee. It finishes by calling StartNewPGARound, which
// promotes the waiting list and re-establishes the heap, so it doubles as the block's first PGA round. Call it once
// per block; use StartNewPGARound directly for any further rounds.
func (m *Mempool[T]) StartNewBlock(baseFee *big.Int, maxTxDataSize int) {
	m.baseFee = baseFee
	m.maxTxDataSize = maxTxDataSize

	// Re-key the queued transactions, compacting in place.
	kept := 0
	for _, entry := range m.heap {
		if err := entry.tx.GetContext().Err(); err != nil {
			entry.tx.ReportError(err)
			continue
		}
		// maxTxDataSize is hot-reloadable, so a tx accepted under a larger limit can
		// linger in the queue; drop it if it no longer fits the current block.
		if entry.tx.GetSize() > m.maxTxDataSize {
			entry.tx.ReportError(txpool.ErrOversizedData)
			continue
		}
		if err := entry.setPriority(m.baseFee); err != nil {
			entry.tx.ReportError(err)
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
// Intake drops expired contexts and fee caps below the basefee, and rejects oversized transactions. StartNewBlock
// runs a block's first round; call this directly to run further rounds within the same block.
func (m *Mempool[T]) StartNewPGARound() {
	// n (the waiting-list length) is captured once; we are the sole consumer, so these receivers never block, and
	// arrivals after the snapshot stay buffered for the next round.
	n := len(m.txQueue)
	for range n {
		item := <-m.txQueue
		if err := item.GetContext().Err(); err != nil {
			item.ReportError(err)
			continue
		}
		if item.GetSize() > m.maxTxDataSize {
			item.ReportError(txpool.ErrOversizedData)
			continue
		}
		entry := prioritizedTx[T]{tx: item}
		if err := entry.setPriority(m.baseFee); err != nil {
			item.ReportError(err)
			continue
		}
		m.heap = append(m.heap, entry)
	}
	// Always re-establish the heap, even for an empty round: StartNewBlock re-keys the queued transactions and then
	// relies on this call to restore the heap invariant.
	heap.Init(&m.heap)
}

// Peek returns the highest-priority transaction without removing it. The caller must check Len() > 0 first.
func (m *Mempool[T]) Peek() T {
	return m.heap[0].tx
}

// Pop removes and returns the highest-priority transaction. The caller must check Len() > 0 first; popping an empty
// queue panics, like container/heap.
func (m *Mempool[T]) Pop() T {
	entry, ok := heap.Pop(&m.heap).(prioritizedTx[T])
	if !ok {
		panic("Mempool.Pop: unexpected heap element type") // impossible
	}
	return entry.tx
}

// Push re-inserts a transaction popped from the queue, re-keying it against the block's basefee.
func (m *Mempool[T]) Push(item T) {
	entry := prioritizedTx[T]{tx: item}
	if err := entry.setPriority(m.baseFee); err != nil {
		item.ReportError(err)
		return
	}
	heap.Push(&m.heap, entry)
}
