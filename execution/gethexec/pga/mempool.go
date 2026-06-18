// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package pga implements the priority mempool that backs the sequencer's Priority Gas Auction.
package pga

import (
	"container/heap"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/txpool"
)

// Tx is a transaction managed by the priority mempool. The sequencer's txQueueItem implements it; the mempool depends
// only on this interface so it stays decoupled from that concrete type.
type Tx interface {
	// ComputePriorityFee returns the transaction's priority fee per gas against the given basefee, saturated to a
	// uint64. It errors when the fee cap is below the basefee, matching the sequencer's gather-time fee-cap check.
	ComputePriorityFee(baseFee *big.Int) (uint64, error)
	// ReturnResult resolves the submitting client's result channel.
	ReturnResult(err error)
	// GetContext returns the submission context, used to drop expired entries.
	GetContext() context.Context
	// GetSize returns the size in bytes of the marshalled transaction.
	GetSize() int
	// GetFirstAppearance returns when the transaction first reached the sequencer; it breaks ties between
	// equal-priority entries.
	GetFirstAppearance() time.Time
}

// txItem pairs a queued transaction with its priority key.
type txItem[T Tx] struct {
	tx       T
	priority uint64
}

// setPriority sets the item's priority from ComputePriorityFee.
func (item *txItem[T]) setPriority(baseFee *big.Int) error {
	fee, err := item.tx.ComputePriorityFee(baseFee)
	if err != nil {
		return err
	}
	item.priority = fee
	return nil
}

// txHeap implements heap.Interface as a max-heap on priority, ties broken by earliest GetFirstAppearance.
type txHeap[T Tx] []txItem[T]

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
	item, ok := x.(txItem[T])
	if !ok {
		panic(fmt.Sprintf("txHeap.Push: unexpected element type %T", x)) // impossible
	}
	*h = append(*h, item)
}

func (h *txHeap[T]) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = txItem[T]{} // zero the slot so it doesn't pin the popped item's references
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
// max transaction size, or whose fee cap fell below the basefee, and re-establishes the heap. Call it once per block,
// before its PGA rounds.
func (m *Mempool[T]) StartNewBlock(baseFee *big.Int, maxTxDataSize int) {
	m.baseFee = baseFee
	m.maxTxDataSize = maxTxDataSize

	// Re-key the queued transactions, compacting in place.
	kept := 0
	for _, entry := range m.heap {
		if err := entry.tx.GetContext().Err(); err != nil {
			entry.tx.ReturnResult(err)
			continue
		}
		// maxTxDataSize is hot-reloadable, so a tx accepted under a larger limit can
		// linger in the queue; drop it if it no longer fits the current block.
		if entry.tx.GetSize() > m.maxTxDataSize {
			entry.tx.ReturnResult(txpool.ErrOversizedData)
			continue
		}
		if err := entry.setPriority(m.baseFee); err != nil {
			entry.tx.ReturnResult(err)
			continue
		}
		m.heap[kept] = entry
		kept++
	}
	// Clear the vacated tail so dropped or moved entries aren't pinned.
	for i := kept; i < len(m.heap); i++ {
		m.heap[i] = txItem[T]{}
	}
	m.heap = m.heap[:kept]
	heap.Init(&m.heap)
}

// StartNewPGARound promotes a snapshot of the waiting list into the priority queue, then re-establishes the heap.
// Intake drops expired contexts and fee caps below the basefee, and rejects oversized transactions.
func (m *Mempool[T]) StartNewPGARound() {
	// n (the waiting-list length) is captured once; we are the sole consumer, so these receives never block, and
	// arrivals after the snapshot stay buffered for the next round.
	n := len(m.txQueue)
	for range n {
		item := <-m.txQueue
		if err := item.GetContext().Err(); err != nil {
			item.ReturnResult(err)
			continue
		}
		if item.GetSize() > m.maxTxDataSize {
			item.ReturnResult(txpool.ErrOversizedData)
			continue
		}
		entry := txItem[T]{tx: item}
		if err := entry.setPriority(m.baseFee); err != nil {
			item.ReturnResult(err)
			continue
		}
		m.heap = append(m.heap, entry)
	}
	// Re-heapify only when the waiting list had arrivals; an empty round appends nothing and leaves the heap (set up
	// by StartNewBlock) untouched.
	if n != 0 {
		heap.Init(&m.heap)
	}
}

// Peek returns the highest-priority transaction without removing it. The caller must check Len() > 0 first.
func (m *Mempool[T]) Peek() T {
	return m.heap[0].tx
}

// Pop removes and returns the highest-priority transaction. The caller must check Len() > 0 first; popping an empty
// queue panics, like container/heap.
func (m *Mempool[T]) Pop() T {
	entry, ok := heap.Pop(&m.heap).(txItem[T])
	if !ok {
		panic("Mempool.Pop: unexpected heap element type") // impossible
	}
	return entry.tx
}

// Push re-inserts a transaction popped from the queue, re-keying it against the block's basefee.
func (m *Mempool[T]) Push(item T) {
	entry := txItem[T]{tx: item}
	if err := entry.setPriority(m.baseFee); err != nil {
		item.ReturnResult(err)
		return
	}
	heap.Push(&m.heap, entry)
}
