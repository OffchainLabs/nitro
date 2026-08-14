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
func (m *Mempool[T]) Pop() (emptyEntry T, found bool) {
	for m.heap.Len() > 0 {
		entry := m.heap.popConcrete()
		if entry.Validate() {
			return entry, true
		}
	}
	return emptyEntry, false
}

// Push adds a transaction, keying it against the mempool's basefee with its carried boost folded in.
func (m *Mempool[T]) Push(item T) {
	if !item.ComputePgaPriority(m.baseFee) {
		return
	}
	m.heap.pushConcrete(item)
}

// PushBatch adds a batch of transactions, keying them against the mempool's basefee with their carried boosts folded
// in. More efficient than pushing one at a time, it re-heapifies the entire queue in a single O(n) pass.
func (m *Mempool[T]) PushBatch(items []T) {
	validated := items[:0] // filter in place
	for _, item := range items {
		if item.ComputePgaPriority(m.baseFee) {
			validated = append(validated, item)
		}
	}
	m.heap.pushBatch(validated)
}

// TakeRemaining returns all remaining transactions in the mempool.
func (m *Mempool[T]) TakeRemaining() []T {
	return m.heap.takeRemaining()
}
