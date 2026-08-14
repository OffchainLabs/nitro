// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"math/big"
	"time"

	"github.com/offchainlabs/nitro/util/arbmath"
)

// Tx is a transaction managed by the priority mempool. The sequencer's txQueueItem implements it;
// the mempool depends only on this interface so it stays decoupled from that concrete type.
type Tx interface {
	// Priority returns the priority of the transaction.
	GetPriority() uint64
	// AddBoost adds the value to the anti-starvation boost.
	AddBoost(uint64)
	// ComputePgaPriority computes the transaction priority for PGA and stores it in the tx,
	// returning false if the transaction was dropped.
	ComputePgaPriority(baseFee *big.Int) bool
	// Validate returns false if the transaction was dropped.
	Validate() bool
	// GetFirstAppearance returns when the transaction first reached the sequencer; it breaks ties between
	// equal-priority entries.
	GetFirstAppearance() time.Time
}

// Priority should be embedded by the Tx. It provides the GetPriority and AddBoost methods.
type Priority struct {
	boost          uint64 // accumulated PGA anti-starvation boost
	cachedPriority uint64 // PGA priority of the tx
}

func (p *Priority) SetPriority(value uint64) {
	// Overflow should not happen, use saturating function just to be safe.
	p.cachedPriority = arbmath.SaturatingUAdd(value, p.boost)
}

func (p *Priority) GetPriority() uint64 {
	return p.cachedPriority
}

func (p *Priority) ResetBoost() {
	// Underflow should not happen, use saturating function just to be safe.
	p.cachedPriority = arbmath.SaturatingUSub(p.cachedPriority, p.boost)
	p.boost = 0
}

func (p *Priority) AddBoost(delta uint64) {
	// Overflow should not happen, use saturating function just to be safe.
	p.boost = arbmath.SaturatingUAdd(p.boost, delta)
	p.cachedPriority = arbmath.SaturatingUAdd(p.cachedPriority, delta)
}
