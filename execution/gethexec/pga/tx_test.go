// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pga

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/txpool"
)

// errFeeCapTooLow stands in for the fee-cap-below-basefee error that the real ComputePgaPriority returns.
// The mempool only propagates it, so its identity is all that matters here.
var errFeeCapTooLow = errors.New("fee cap below base fee")

// priorityFeeFunc computes a transaction's priority fee against a basefee.
type priorityFeeFunc func(baseFee *big.Int) (uint64, error)

func constFee(fee uint64) priorityFeeFunc {
	return func(*big.Int) (uint64, error) { return fee, nil }
}

func failFee(err error) priorityFeeFunc {
	return func(*big.Int) (uint64, error) { return 0, err }
}

type mockTx struct {
	id              int
	fee             priorityFeeFunc
	size            int
	ctx             context.Context
	firstAppearance time.Time
	resultChan      chan error
	returnedResult  *atomic.Bool
}

func (m mockTx) ComputePgaPriority(baseFee *big.Int) (uint64, error) { return m.fee(baseFee) }

func (m mockTx) ReportError(err error) {
	if m.returnedResult.Swap(true) {
		return
	}
	m.resultChan <- err
	close(m.resultChan)
}

func (m mockTx) GetContext() context.Context { return m.ctx }

func (m mockTx) GetSize() int { return m.size }

func (m mockTx) GetFirstAppearance() time.Time { return m.firstAppearance }

func TestPgaPrioritizedTxSetPriorityFromComputedFee(t *testing.T) {
	item := prioritizedTx[mockTx]{tx: mockTx{fee: constFee(42)}}
	if !item.setPriority(big.NewInt(7)) {
		t.Fatal("setPriority returned false, want true")
	}
	if item.priority != 42 {
		t.Fatalf("priority = %d, want 42", item.priority)
	}
}

func TestPgaPrioritizedTxSetPriorityForwardsBaseFee(t *testing.T) {
	var seen *big.Int
	item := prioritizedTx[mockTx]{tx: mockTx{fee: func(baseFee *big.Int) (uint64, error) {
		seen = baseFee
		return 0, nil
	}}}
	if !item.setPriority(big.NewInt(99)) {
		t.Fatal("setPriority returned false, want true")
	}
	if seen == nil || seen.Int64() != 99 {
		t.Fatalf("basefee passed to ComputePgaPriority = %v, want 99", seen)
	}
}

func TestPgaPrioritizedTxSetPriorityReportsError(t *testing.T) {
	resultChan := make(chan error, 1)
	item := prioritizedTx[mockTx]{tx: mockTx{
		fee:            failFee(errFeeCapTooLow),
		resultChan:     resultChan,
		returnedResult: &atomic.Bool{},
	}}
	if item.setPriority(big.NewInt(7)) {
		t.Fatal("setPriority returned true, want false")
	}
	expectResult(t, resultChan, errFeeCapTooLow)
}

func TestPgaPrioritizedTxValidateAcceptsValidTx(t *testing.T) {
	// size == maxTxDataSize is allowed: the check is a strict greater-than.
	item := prioritizedTx[mockTx]{tx: mockTx{ctx: context.Background(), size: 100}}
	if !item.validate(100) {
		t.Fatal("validate returned false, want true")
	}
}

func TestPgaPrioritizedTxValidateDropsExpiredContext(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	resultChan := make(chan error, 1)
	item := prioritizedTx[mockTx]{tx: mockTx{
		ctx:            canceledCtx,
		size:           10,
		resultChan:     resultChan,
		returnedResult: &atomic.Bool{},
	}}
	if item.validate(100) {
		t.Fatal("validate returned true, want false")
	}
	expectResult(t, resultChan, context.Canceled)
}

func TestPgaPrioritizedTxValidateDropsOversized(t *testing.T) {
	resultChan := make(chan error, 1)
	item := prioritizedTx[mockTx]{tx: mockTx{
		ctx:            context.Background(),
		size:           101,
		resultChan:     resultChan,
		returnedResult: &atomic.Bool{},
	}}
	if item.validate(100) {
		t.Fatal("validate returned true, want false")
	}
	expectResult(t, resultChan, txpool.ErrOversizedData)
}

func TestPgaPrioritizedTxValidateChecksContextBeforeSize(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	resultChan := make(chan error, 1)
	// Both checks would fail; context is checked first, so the reported error is the context error.
	item := prioritizedTx[mockTx]{tx: mockTx{
		ctx:            canceledCtx,
		size:           101,
		resultChan:     resultChan,
		returnedResult: &atomic.Bool{},
	}}
	if item.validate(100) {
		t.Fatal("validate returned true, want false")
	}
	expectResult(t, resultChan, context.Canceled)
}
