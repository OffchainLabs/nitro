// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package pga implements the priority mempool that backs the sequencer's Priority Gas Auction.
package pga

import (
	"math/big"
)

// Mempool is a two-stage mempool for PGA. The first stage is a channel with the waiting list and the second stage is a
// priority queue. It is not safe for concurrent use; every method runs on the block-production goroutine.
type Mempool[T Tx] struct {
	txQueue              <-chan T  // stage one: the waiting list
	heap                 txHeap[T] // stage two: the priority queue
	baseFee              *big.Int  // basefee of the block under construction
	maxTxDataSize        int       // max promoted-transaction size, for the block under construction
	boostDivisor         uint64    // used to compute the priority boost
	lastIncludedPriority uint64    // the priority of the last transaction included in the pga round
}

func NewMempool[T Tx](txQueue <-chan T, roundsPerBlock uint) *Mempool[T] {
	if roundsPerBlock == 0 {
		panic("roundsPerBlock is zero; impossible")
	}
	return &Mempool[T]{
		txQueue:      txQueue,
		boostDivisor: 2 * uint64(roundsPerBlock),
	}
}

// PriorityQueueLen returns the number of transactions promoted into the priority queue (stage two); it does not count
// the waiting list.
func (m *Mempool[T]) PriorityQueueLen() int {
	return m.heap.Len()
}

// AreThereTxsForNextRound reports whether the next PGA round would have anything to work with.
func (m *Mempool[T]) AreThereTxsForNextRound() bool {
	return m.heap.Len() != 0 || len(m.txQueue) != 0
}

// StartNewBlock begins a block: it records the block's basefee and max transaction size, then re-keys the queued
// transactions against the new basefee, dropping any whose fee cap fell below it. It finishes by calling
// StartNewPGARound, which promotes the waiting list and re-establishes the heap, so it doubles as the block's first PGA
// round.
func (m *Mempool[T]) StartNewBlock(baseFee *big.Int, maxTxDataSize int) {
	m.baseFee = baseFee
	m.maxTxDataSize = maxTxDataSize
	m.heap.rekey(m.baseFee)
	m.StartNewPGARound()
}

// StartNewPGARound advances the mempool to a new PGA round. It applies the anti-starvation boost to transactions that
// are still in the priority queue, and promotes transactions from the waiting list.
func (m *Mempool[T]) StartNewPGARound() {
	if delta := m.lastIncludedPriority / m.boostDivisor; delta != 0 {
		m.heap.addBoost(delta)
	}
	m.lastIncludedPriority = 0

	// n (the waiting-list length) is captured once; we are the sole consumer, so these receivers never block, and
	// arrivals after the snapshot stay buffered for the next round.
	n := len(m.txQueue)
	promoted := make([]PrioritizedTx[T], 0, n)
	for range n {
		entry := PrioritizedTx[T]{tx: <-m.txQueue}
		if !entry.setPriority(m.baseFee) {
			continue
		}
		promoted = append(promoted, entry)
	}
	m.heap.pushBatch(promoted)
}

// RecordIncludedTx records the priority of a transaction just included in the block during the current round.
func (m *Mempool[T]) RecordIncludedTx(priority uint64) {
	m.lastIncludedPriority = priority
}

// Pop removes and returns the highest-priority valid entry, dropping candidates that fail the size or context checks.
func (m *Mempool[T]) Pop() (PrioritizedTx[T], bool) {
	for m.heap.Len() > 0 {
		entry := m.heap.popConcrete()
		if entry.validate(m.maxTxDataSize) {
			return entry, true
		}
	}
	return PrioritizedTx[T]{}, false
}

// Push adds a boost-free transaction, keying it against the current basefee. Use it for a transaction revived from the
// nonce-failure cache, which re-enters the queue fresh without any boost it accumulated before being cached.
func (m *Mempool[T]) Push(item T) {
	m.PushPrioritized(PrioritizedTx[T]{tx: item})
}

// PushPrioritized re-adds an already-prioritized entry, re-keying it against the current basefee while preserving its
// accumulated boost. Use it for a transaction that did not fit in the block and returns to the queue.
func (m *Mempool[T]) PushPrioritized(entry PrioritizedTx[T]) {
	if !entry.setPriority(m.baseFee) {
		return
	}
	m.heap.pushConcrete(entry)
}
