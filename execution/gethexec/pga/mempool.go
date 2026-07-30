// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package pga implements the priority mempool that backs the sequencer's Priority Gas Auction.
package pga

import (
	"math/big"
)

// Mempool is the priority queue backing one block's PGA rounds: transactions are keyed against the basefee fixed at
// construction, so the sequencer builds a fresh mempool per block. It is not safe for concurrent use; every method
// runs on the block-production goroutine.
type Mempool[T Tx] struct {
	heap                 txHeap[T] // the priority queue
	baseFee              *big.Int  // basefee of the block under construction
	boostDivisor         uint64    // used to compute the priority boost
	lastIncludedPriority uint64    // the priority of the last transaction included in the pga round
}

func NewMempool[T Tx](roundsPerBlock uint, baseFee *big.Int) *Mempool[T] {
	if roundsPerBlock == 0 {
		panic("roundsPerBlock is zero; impossible")
	}
	return &Mempool[T]{
		boostDivisor: 2 * uint64(roundsPerBlock),
		baseFee:      baseFee,
	}
}

// PriorityQueueLen returns the number of transactions in the priority queue.
func (m *Mempool[T]) PriorityQueueLen() int {
	return m.heap.Len()
}

// ApplyRoundBoost advances the mempool to a new PGA round: it applies the anti-starvation boost, derived from the
// last included transaction, to the transactions still in the priority queue.
func (m *Mempool[T]) ApplyRoundBoost() {
	if delta := m.lastIncludedPriority / m.boostDivisor; delta != 0 {
		m.heap.addBoost(delta)
	}
	m.lastIncludedPriority = 0
}

// RecordIncludedTx records the priority of a transaction just included in the block during the current round.
func (m *Mempool[T]) RecordIncludedTx(priority uint64) {
	m.lastIncludedPriority = priority
}

// Pop removes and returns the highest-priority valid entry, dropping candidates whose submission context has expired.
func (m *Mempool[T]) Pop() (PrioritizedTx[T], bool) {
	for m.heap.Len() > 0 {
		entry := m.heap.popConcrete()
		if entry.validate() {
			return entry, true
		}
	}
	return PrioritizedTx[T]{}, false
}

// Push adds a transaction with no accumulated boost, keying it against the mempool's basefee.
func (m *Mempool[T]) Push(item T) {
	m.PushPrioritized(item, 0)
}

// PushPrioritized adds a transaction, folding its previously accumulated anti-starvation boost into the priority.
// Use it for a transaction re-entering the queue, such as a leftover requeued from the previous block.
func (m *Mempool[T]) PushPrioritized(item T, boost uint64) {
	entry := PrioritizedTx[T]{tx: item, boost: boost}
	if !entry.setPriority(m.baseFee) {
		return
	}
	m.heap.pushConcrete(entry)
}
