// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"math/big"
	"time"
)

// Tx is a transaction managed by the priority mempool. The sequencer's txQueueItem implements it; the mempool depends
// only on this interface so it stays decoupled from that concrete type.
type Tx interface {
	// ComputePgaPriority returns the transaction's priority for PGA.
	ComputePgaPriority(baseFee *big.Int) (uint64, error)
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

// prioritizedTx pairs a queued transaction with its priority key.
type prioritizedTx[T Tx] struct {
	tx       T
	priority uint64
}

// setPriority sets the item's priority from ComputePgaPriority.
func (item *prioritizedTx[T]) setPriority(baseFee *big.Int) error {
	fee, err := item.tx.ComputePgaPriority(baseFee)
	if err != nil {
		return err
	}
	item.priority = fee
	return nil
}
