// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"math/big"
	"time"

	"github.com/offchainlabs/nitro/util/arbmath"
)

// Tx is a transaction managed by the priority mempool. The sequencer's txQueueItem implements it; the mempool depends
// only on this interface so it stays decoupled from that concrete type.
type Tx interface {
	// ComputePgaPriority returns the transaction's base priority for PGA, independent of any anti-starvation boost.
	ComputePgaPriority(baseFee *big.Int) (uint64, error)
	// ReportError resolves the submitting client's result channel with err.
	ReportError(err error)
	// GetContext returns the submission context, used to drop expired entries.
	GetContext() context.Context
	// GetFirstAppearance returns when the transaction first reached the sequencer; it breaks ties between
	// equal-priority entries.
	GetFirstAppearance() time.Time
	// GetBoost returns the anti-starvation boost the transaction carries in, accumulated in a previous block.
	GetBoost() uint64
}

// PrioritizedTx pairs a queued transaction with its priority key.
type PrioritizedTx[T Tx] struct {
	tx T
	// cachedPriority is the ordering key: the priority fee plus the accumulated boost. The priority fee is cached
	// rather than recomputed on every access because it changes only with the basefee at a block boundary, not during
	// the block.
	cachedPriority uint64
	boost          uint64 // accumulated anti-starvation boost
}

// Tx returns the wrapped transaction.
func (item PrioritizedTx[T]) Tx() T { return item.tx }

// Priority returns the entry's priority key.
func (item PrioritizedTx[T]) Priority() uint64 { return item.cachedPriority }

// Boost returns the entry's accumulated anti-starvation boost.
func (item PrioritizedTx[T]) Boost() uint64 { return item.boost }

// setPriority recomputes the base priority from ComputePgaPriority and folds in the accumulated boost. On error it
// returns false, signalling that the transaction was dropped.
func (item *PrioritizedTx[T]) setPriority(baseFee *big.Int) bool {
	base, err := item.tx.ComputePgaPriority(baseFee)
	if err != nil {
		item.tx.ReportError(err)
		return false
	}
	item.cachedPriority = arbmath.SaturatingUAdd(base, item.boost)
	return true
}

// addBoost adds delta to the accumulated boost and the priority key. The add saturates so a key near the uint64 ceiling
// cannot wrap.
func (item *PrioritizedTx[T]) addBoost(delta uint64) {
	item.boost = arbmath.SaturatingUAdd(item.boost, delta)
	item.cachedPriority = arbmath.SaturatingUAdd(item.cachedPriority, delta)
}

// validate returns false if the transaction was dropped.
func (item *PrioritizedTx[T]) validate() bool {
	if err := item.tx.GetContext().Err(); err != nil {
		item.tx.ReportError(err)
		return false
	}
	return true
}
