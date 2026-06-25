// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/txpool"
)

// Tx is a transaction managed by the priority mempool. The sequencer's txQueueItem implements it; the mempool depends
// only on this interface so it stays decoupled from that concrete type.
type Tx interface {
	// ComputePgaPriority returns the transaction's priority for PGA.
	ComputePgaPriority(baseFee *big.Int) (uint64, error)
	// IncreaseBoost adds delta to the transaction's accumulated anti-starvation boost, which ComputePgaPriority folds
	// into the priority. The mempool calls it on every queued transaction at a round boundary.
	IncreaseBoost(delta uint64)
	// ReportError resolves the submitting client's result channel with err.
	ReportError(err error)
	// GetContext returns the submission context, used to drop expired entries.
	GetContext() context.Context
	// GetSize returns the size in bytes of the marshalled transaction.
	GetSize() int
	// GetFirstAppearance returns when the transaction first reached the sequencer; it breaks ties between
	// equal-priority entries.
	GetFirstAppearance() time.Time
}

// PrioritizedTx pairs a queued transaction with its priority key.
type PrioritizedTx[T Tx] struct {
	tx       T
	priority uint64
}

// Tx returns the wrapped transaction.
func (item PrioritizedTx[T]) Tx() T { return item.tx }

// Priority returns the entry's priority key.
func (item PrioritizedTx[T]) Priority() uint64 { return item.priority }

// setPriority computes the item's priority from ComputePgaPriority. On error it returns false, signalling that the
// transaction was dropped.
func (item *PrioritizedTx[T]) setPriority(baseFee *big.Int) bool {
	fee, err := item.tx.ComputePgaPriority(baseFee)
	if err != nil {
		item.tx.ReportError(err)
		return false
	}
	item.priority = fee
	return true
}

// validate returns false if the transaction was dropped.
func (item *PrioritizedTx[T]) validate(maxTxDataSize int) bool {
	if err := item.tx.GetContext().Err(); err != nil {
		item.tx.ReportError(err)
		return false
	}
	if item.tx.GetSize() > maxTxDataSize {
		item.tx.ReportError(txpool.ErrOversizedData)
		return false
	}
	return true
}
