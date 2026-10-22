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
	// ApplyRoundBoundary marks a PGA round boundary: the tx counts it as a round waited in the
	// priority queue and folds the given anti-starvation boost into its priority.
	ApplyRoundBoundary(boostDelta uint64)
	// ComputePgaPriority computes the transaction priority for PGA and stores it in the tx,
	// returning false if the transaction was dropped.
	ComputePgaPriority(baseFee *big.Int) bool
	// Validate returns false if the transaction was dropped.
	Validate() bool
	// GetFirstAppearance returns when the transaction first reached the sequencer; it breaks ties between
	// equal-priority entries.
	GetFirstAppearance() time.Time
}

// PGAState should be embedded by the Tx. It provides the GetPriority and ApplyRoundBoundary methods.
type PGAState struct {
	tip          uint64 // effective tip per gas as of the last SetTip
	boost        uint64 // accumulated PGA anti-starvation boost
	roundsWaited uint64 // PGA round boundaries the tx sat through in the priority queue
	promoted     bool   // whether the tx has ever entered the priority queue
}

func (p *PGAState) SetTip(tip uint64) {
	p.tip = tip
}

func (p *PGAState) GetPriority() uint64 {
	// Overflow should not happen, use saturating function just to be safe.
	return arbmath.SaturatingUAdd(p.tip, p.boost)
}

func (p *PGAState) GetBoost() uint64 {
	return p.boost
}

func (p *PGAState) GetTip() uint64 {
	return p.tip
}

func (p *PGAState) GetRoundsWaited() uint64 {
	return p.roundsWaited
}

func (p *PGAState) ResetBoost() {
	p.boost = 0
}

// ApplyRoundBoundary counts a PGA round boundary the tx sat through in the priority queue,
// applying the round's anti-starvation boost.
func (p *PGAState) ApplyRoundBoundary(boostDelta uint64) {
	p.roundsWaited++
	p.boost = arbmath.SaturatingUAdd(p.boost, boostDelta)
}

// MarkPromoted records the tx's first promotion into the priority queue, reporting whether this
// call was the first.
func (p *PGAState) MarkPromoted() bool {
	if p.promoted {
		return false
	}
	p.promoted = true
	return true
}
