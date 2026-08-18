// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"math/big"
	"testing"
	"time"
)

// priorityFeeFunc computes a transaction's priority fee against a basefee, reporting ok=false when the tx must be
// dropped, as when its fee cap falls below the basefee.
type priorityFeeFunc func(baseFee *big.Int) (uint64, bool)

func constFee(fee uint64) priorityFeeFunc {
	return func(*big.Int) (uint64, bool) { return fee, true }
}

// droppedFee stands in for a fee computation that drops the tx, like a fee cap below the basefee.
func droppedFee() priorityFeeFunc {
	return func(*big.Int) (uint64, bool) { return 0, false }
}

// mockTx implements Tx for the heap and mempool tests, mirroring how txQueueItem embeds *PGAState so copies share the
// priority state.
type mockTx struct {
	*PGAState
	id              int
	fee             priorityFeeFunc
	expired         *bool // Validate reports the tx dropped; shared like the submission ctx, so tests can expire an in-heap entry
	firstAppearance time.Time
}

var _ Tx = mockTx{}

func (m mockTx) ComputePgaPriority(baseFee *big.Int) bool {
	base, ok := m.fee(baseFee)
	if !ok {
		return false
	}
	m.SetTip(base)
	return true
}

func (m mockTx) Validate() bool { return m.expired == nil || !*m.expired }

func (m mockTx) GetFirstAppearance() time.Time { return m.firstAppearance }

func TestPgaPriorityGetPriorityFoldsBoost(t *testing.T) {
	p := &PGAState{}
	p.SetTip(42)
	if p.GetPriority() != 42 {
		t.Fatalf("priority = %d, want 42", p.GetPriority())
	}
	p.AddBoost(8)
	p.SetTip(42) // re-key, as when the tx enters the next block's mempool
	if p.GetPriority() != 50 {
		t.Fatalf("priority = %d, want 50 (42 base + 8 boost)", p.GetPriority())
	}
}

func TestPgaPriorityAddBoostAccumulates(t *testing.T) {
	p := &PGAState{}
	p.SetTip(10)
	p.AddBoost(5)
	p.AddBoost(7)
	if p.GetPriority() != 22 {
		t.Fatalf("priority = %d, want 22 (10 base + 12 boost)", p.GetPriority())
	}
}

func TestPgaPriorityResetBoostForfeitsBoost(t *testing.T) {
	p := &PGAState{}
	p.SetTip(10)
	p.AddBoost(15)
	p.ResetBoost()
	if p.GetPriority() != 10 {
		t.Fatalf("priority = %d, want 10 (boost forfeited)", p.GetPriority())
	}
	// A later re-key must not resurrect the forfeited boost.
	p.SetTip(10)
	if p.GetPriority() != 10 {
		t.Fatalf("priority after re-key = %d, want 10", p.GetPriority())
	}
}
