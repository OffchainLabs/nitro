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
	txQueue       <-chan T  // stage one: the waiting list
	heap          txHeap[T] // stage two: the priority queue
	baseFee       *big.Int  // basefee of the block under construction
	maxTxDataSize int       // max promoted-transaction size, for the block under construction
}

func NewMempool[T Tx](txQueue <-chan T) *Mempool[T] {
	return &Mempool[T]{txQueue: txQueue}
}

func (m *Mempool[T]) Len() int {
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

// StartNewPGARound promotes a snapshot of the waiting list into the priority queue, then re-establishes the heap.
func (m *Mempool[T]) StartNewPGARound() {
	// n (the waiting-list length) is captured once; we are the sole consumer, so these receivers never block, and
	// arrivals after the snapshot stay buffered for the next round.
	n := len(m.txQueue)
	promoted := make([]prioritizedTx[T], 0, n)
	for range n {
		entry := prioritizedTx[T]{tx: <-m.txQueue}
		if !entry.setPriority(m.baseFee) {
			continue
		}
		promoted = append(promoted, entry)
	}
	m.heap.pushBatch(promoted)
}

// Pop removes and returns the highest-priority valid transaction. It validates each candidate against the block's max
// transaction size and context, dropping those that fail.
func (m *Mempool[T]) Pop() (tx T, ok bool) {
	for m.heap.Len() > 0 {
		entry := m.heap.popConcrete()
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
	m.heap.pushConcrete(entry)
}
