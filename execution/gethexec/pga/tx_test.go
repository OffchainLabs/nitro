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

func (m mockTx) ReturnResult(err error) {
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
	if err := item.setPriority(big.NewInt(7)); err != nil {
		t.Fatalf("unexpected err: %v", err)
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
	if err := item.setPriority(big.NewInt(99)); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if seen == nil || seen.Int64() != 99 {
		t.Fatalf("basefee passed to ComputePgaPriority = %v, want 99", seen)
	}
}

func TestPgaPrioritizedTxSetPriorityPropagatesError(t *testing.T) {
	item := prioritizedTx[mockTx]{tx: mockTx{fee: failFee(errFeeCapTooLow)}}
	if err := item.setPriority(big.NewInt(7)); !errors.Is(err, errFeeCapTooLow) {
		t.Fatalf("err = %v, want errors.Is(errFeeCapTooLow)", err)
	}
}
