// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package pga implements the priority mempool that backs the sequencer's Priority Gas Auction.
package pga

import (
	"math/big"

	"github.com/ethereum/go-ethereum/metrics"
)

var (
	txsAddedCounter     = metrics.NewRegisteredCounter("arb/sequencer/pga/mempool/txsadded", nil)
	txsProcessedCounter = metrics.NewRegisteredCounter("arb/sequencer/pga/mempool/txsprocessed", nil)
	txsDroppedCounter   = metrics.NewRegisteredCounter("arb/sequencer/pga/mempool/txsdropped", nil)
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
	m.heap.applyRoundBoundary(m.lastIncludedPriority / m.boostDivisor)
	m.lastIncludedPriority = 0
}

// RecordIncludedTx records the priority of a transaction just included in the block during the current round.
func (m *Mempool[T]) RecordIncludedTx(priority uint64) {
	m.lastIncludedPriority = priority
}

// Peek returns the highest-priority valid transaction without removing it, discarding entries whose submission
// context has expired.
func (m *Mempool[T]) Peek() (emptyEntry T, found bool) {
	for m.heap.Len() > 0 {
		if entry := m.heap.peekConcrete(); entry.Validate() {
			return entry, true
		}
		m.heap.popConcrete() // expired: discard
		txsDroppedCounter.Inc(1)
	}
	return emptyEntry, false
}

// PopPeeked removes the entry a preceding Peek returned. It does not re-validate: Peek already did, and a second
// check could drop the entry the caller is about to sequence.
func (m *Mempool[T]) PopPeeked() {
	if m.heap.Len() == 0 {
		return
	}
	m.heap.popConcrete()
	txsProcessedCounter.Inc(1)
}

// Push adds a transaction, keying it against the mempool's basefee with its carried boost folded in. It reports
// whether the transaction was inserted rather than dropped.
func (m *Mempool[T]) Push(item T) bool {
	if !item.ComputePgaPriority(m.baseFee) {
		txsDroppedCounter.Inc(1)
		return false
	}
	m.heap.pushConcrete(item)
	txsAddedCounter.Inc(1)
	return true
}

// PushBatch adds a batch of transactions, keying them against the mempool's basefee with their carried boosts folded
// in. More efficient than pushing one at a time, it re-heapifies the entire queue in a single O(n) pass. It returns
// the inserted transactions, excluding the ones dropped at priority computation.
func (m *Mempool[T]) PushBatch(items []T) []T {
	validated := items[:0]
	for _, item := range items {
		if item.ComputePgaPriority(m.baseFee) {
			validated = append(validated, item)
		} else {
			txsDroppedCounter.Inc(1)
		}
	}
	m.heap.pushBatch(validated)
	txsAddedCounter.Inc(int64(len(validated)))
	return validated
}

// TakeRemaining returns all remaining transactions in the mempool.
func (m *Mempool[T]) TakeRemaining() []T {
	return m.heap.takeRemaining()
}
